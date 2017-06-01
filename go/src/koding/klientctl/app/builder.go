package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"sync"

	kstack "koding/kites/kloud/stack"
	"koding/kites/kloud/stack/provider"
	"koding/klientctl/app/mixin"
	"koding/klientctl/endpoint/credential"
	"koding/klientctl/endpoint/kloud"
	"koding/klientctl/endpoint/remoteapi"
	"koding/klientctl/endpoint/stack"
	"koding/klientctl/helper"

	"github.com/koding/logging"
	"github.com/kr/pretty"
)

// DefaultBuilder is used by global functions, like app.BuildTemplate,
// app.BuildStack etc.
var DefaultBuilder = &Builder{}

// Builder allows for manipulating contents of stack
// templates in a provider-agnostic manner.
type Builder struct {
	Desc kstack.Descriptions // provider description; retrieved from kloud, if nil
	Log  logging.Logger      // logger; kloud.DefaultLog if nil

	Koding     *remoteapi.Client  // remote.api client; remoteapi.DefaultClient if nil
	Kloud      *kloud.Client      // kloud client; kloud.DefaultClient if nil
	Credential *credential.Client // credential client; credential.DefaultClient if nil
	Stack      *stack.Client      // stack client; stack.DefaultClient if nil

	Stdout io.Writer // stdout to write; os.Stdout if nil
	Stderr io.Writer // stderr to write; os.Stderr if nil

	once sync.Once // for b.init()
}

// TemplateOptions are used when building a template.
type TemplateOptions struct {
	UseDefaults bool         // forces default value when true; disables interactive mode
	Provider    string       // provider name; inferred from Template if empty
	Template    string       // base template to use; retrieved from remote.api's samples, if empty
	Mixin       *mixin.Mixin // optionally a mixin to replace default's user_data
}

// BuildTemplate builds a template with the given options.
//
// If method finishes successfully, returning nil error, b.Stack field
// will hold the built template.
func (b *Builder) BuildTemplate(opts *TemplateOptions) (interface{}, error) {
	b.init()

	var (
		defaults map[string]interface{}
		tmpl     = opts.Template
		prov     = opts.Provider
		err      error
	)

	if tmpl == "" {
		if prov == "" {
			return nil, errors.New("either provider or template is required to be non-empty")
		}

		tmpl, defaults, err = b.koding().SampleTemplate(prov)
		if err != nil {
			return nil, err
		}
	}

	if prov == "" {
		prov, err = kstack.ReadProvider([]byte(tmpl))
		if err != nil {
			return nil, err
		}
	}

	desc, ok := b.Desc[prov]
	if !ok {
		return nil, fmt.Errorf("provider %q does not exist", prov)
	}

	if opts.Mixin != nil {
		if !desc.CloudInit {
			return nil, fmt.Errorf("provider %q does not support cloud-init files", prov)
		}

		t, err := replaceUserData(tmpl, opts.Mixin, desc)
		if err != nil {
			return nil, err
		}

		p, err := json.Marshal(t)
		if err != nil {
			return nil, err
		}

		tmpl = string(p)
	}

	vars := provider.ReadVariables(tmpl)
	input := make(map[string]string, len(vars))

	for _, v := range vars {
		if !strings.HasPrefix(v.Name, "userInput_") {
			continue
		}

		name := v.Name[len("userInput_"):]
		defValue := ""
		if v, ok := defaults[name]; ok && v != nil {
			defValue = fmt.Sprintf("%v", v)
		}

		var value string

		if !opts.UseDefaults {
			if value, err = helper.Ask("Set %q to [%s]: ", name, defValue); err != nil {
				return nil, err
			}
		}

		if value == "" {
			value = defValue
		}

		input[v.Name] = value
	}

	tmpl = provider.ReplaceVariablesFunc(tmpl, vars, func(v *provider.Variable) string {
		if s, ok := input[v.Name]; ok {
			return s
		}

		return v.String()
	})

	var v interface{}

	if err := json.Unmarshal([]byte(tmpl), &v); err != nil {
		return nil, err
	}

	return v, nil
}

type StackOptions struct {
	Team        string
	Title       string
	Credentials []string
	Template    []byte
	File        string
}

func (opts *StackOptions) template() ([]byte, error) {
	if opts.Template != nil {
		return opts.Template, nil
	}

	var err error

	switch opts.File {
	case "":
		err = errors.New("no template file was provided")
	case "-":
		opts.Template, err = ioutil.ReadAll(os.Stdin)
	default:
		opts.Template, err = ioutil.ReadFile(opts.File)
	}

	return opts.Template, err
}

// BuildStack builds a compute stack with the given options.
//
// After the stack is successfully created it waits until
// the stack is successfully built and until user_data
// script finishes execution.
func (b *Builder) BuildStack(opts *StackOptions) error {
	p, err := opts.template()
	if err != nil {
		return err
	}

	o := &stack.CreateOptions{
		Team:        opts.Team,
		Title:       opts.Title,
		Credentials: opts.Credentials,
		Template:    p,
	}

	fmt.Fprintln(b.stderr(), "Creating stack... ")

	resp, err := b.stack().Create(o)
	if err != nil {
		return errors.New("error creating stack: " + err.Error())
	}

	fmt.Fprintf(b.stderr(), "\nCreated %q stack with %s ID.\nWaiting for the stack to finish building...\n\n", resp.Title, resp.StackID)

	for e := range b.kloud().Wait(resp.EventID) {
		if e.Error != nil {
			return fmt.Errorf("\nBuilding %q stack failed:\n%s\n", resp.Title, e.Error)
		}

		fmt.Fprintf(b.stderr(), "[%d%%] %s\n", e.Event.Percentage, e.Event.Message)
	}

	s, err := b.koding().ListStacks(&remoteapi.Filter{ID: resp.StackID})
	if err != nil {
		return err
	}

	pretty.Println(s)

	return nil
}

func (b *Builder) init() {
	b.once.Do(b.initBuilder)
}

func (b *Builder) initBuilder() {
	if b.Desc == nil {
		var err error
		if b.Desc, err = b.credential().Describe(); err != nil {
			b.log().Warning("unable to retrieve credential description: %s", err)
		}
	}
}

func (b *Builder) log() logging.Logger {
	if b.Log != nil {
		return b.Log
	}
	return kloud.DefaultLog
}

func (b *Builder) koding() *remoteapi.Client {
	if b.Koding != nil {
		return b.Koding
	}
	return remoteapi.DefaultClient
}

func (b *Builder) credential() *credential.Client {
	if b.Credential != nil {
		return b.Credential
	}
	return credential.DefaultClient
}

func (b *Builder) stack() *stack.Client {
	if b.Stack != nil {
		return b.Stack
	}
	return stack.DefaultClient
}

func (b *Builder) stdout() io.Writer {
	if b.Stdout != nil {
		return b.Stdout
	}
	return os.Stdout
}

func (b *Builder) stderr() io.Writer {
	if b.Stderr != nil {
		return b.Stderr
	}
	return os.Stderr
}

func (b *Builder) kloud() *kloud.Client {
	if b.Kloud != nil {
		return b.Kloud
	}
	return kloud.DefaultClient
}

func replaceUserData(tmpl string, m *mixin.Mixin, desc *kstack.Description) (map[string]interface{}, error) {
	var (
		root map[string]interface{}
		key  string
		keys = append([]string{"resource"}, desc.UserData...)
		ok   bool
	)

	if err := json.Unmarshal([]byte(tmpl), &root); err != nil {
		return nil, err
	}

	machines := root

	// The desc.UserData is a JSON path to a user_data value
	// inside the resource tree. Considering the following desc.UserData:
	//
	//   []string{"google_compute_instance", "*", "metadata", "user-data"}
	//
	// the following is going to traverse the template until we assign
	// to machines a value under the * key.
	for len(keys) > 0 {
		key, keys = keys[0], keys[1:]

		if key == "*" {
			break
		}

		machines, ok = machines[key].(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("template does not contain %q key: %v", key, desc.UserData)
		}
	}

	// The following loop assignes extra attributes to each machine and
	// replaces user_data with mixin.
	for _, v := range machines {
		machine, ok := v.(map[string]interface{})
		if !ok {
			continue
		}

		for k, v := range m.Machine {
			machine[k] = v
		}

		// For each machine we locate the user_data by traversing
		// the path after the *.
		for len(keys) > 0 {
			key, keys = keys[0], keys[1:]

			if len(keys) == 0 {
				machine[key] = m.CloudInit.String()
				break
			}

			machine, ok = machine[key].(map[string]interface{})
			if !ok {
				return nil, fmt.Errorf("template does not contain %q key: %v", key, desc.UserData)
			}
		}
	}

	return root, nil
}

// BuildStack builds a compute stack with the given options.
//
// After the stack is successfully created it waits until
// the stack is successfully built and until user_data
// script finishes execution.
func BuildStack(opts *StackOptions) error {
	return DefaultBuilder.BuildStack(opts)
}

// BuildTemplate builds a template with the given options.
//
// If method finishes successfully, returning nil error, b.Stack field
// will hold the built template.
func BuildTemplate(opts *TemplateOptions) (interface{}, error) {
	return DefaultBuilder.BuildTemplate(opts)
}
