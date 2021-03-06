package social_channel

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"fmt"
	"io"

	"github.com/go-openapi/runtime"

	strfmt "github.com/go-openapi/strfmt"

	"koding/remoteapi/models"
)

// SocialChannelFetchActivityCountReader is a Reader for the SocialChannelFetchActivityCount structure.
type SocialChannelFetchActivityCountReader struct {
	formats strfmt.Registry
}

// ReadResponse reads a server response into the received o.
func (o *SocialChannelFetchActivityCountReader) ReadResponse(response runtime.ClientResponse, consumer runtime.Consumer) (interface{}, error) {
	switch response.Code() {

	case 200:
		result := NewSocialChannelFetchActivityCountOK()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return result, nil

	case 401:
		result := NewSocialChannelFetchActivityCountUnauthorized()
		if err := result.readResponse(response, consumer, o.formats); err != nil {
			return nil, err
		}
		return nil, result

	default:
		return nil, runtime.NewAPIError("unknown error", response, response.Code())
	}
}

// NewSocialChannelFetchActivityCountOK creates a SocialChannelFetchActivityCountOK with default headers values
func NewSocialChannelFetchActivityCountOK() *SocialChannelFetchActivityCountOK {
	return &SocialChannelFetchActivityCountOK{}
}

/*SocialChannelFetchActivityCountOK handles this case with default header values.

Request processed successfully
*/
type SocialChannelFetchActivityCountOK struct {
	Payload *models.DefaultResponse
}

func (o *SocialChannelFetchActivityCountOK) Error() string {
	return fmt.Sprintf("[POST /remote.api/SocialChannel.fetchActivityCount][%d] socialChannelFetchActivityCountOK  %+v", 200, o.Payload)
}

func (o *SocialChannelFetchActivityCountOK) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.DefaultResponse)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}

// NewSocialChannelFetchActivityCountUnauthorized creates a SocialChannelFetchActivityCountUnauthorized with default headers values
func NewSocialChannelFetchActivityCountUnauthorized() *SocialChannelFetchActivityCountUnauthorized {
	return &SocialChannelFetchActivityCountUnauthorized{}
}

/*SocialChannelFetchActivityCountUnauthorized handles this case with default header values.

Unauthorized request
*/
type SocialChannelFetchActivityCountUnauthorized struct {
	Payload *models.UnauthorizedRequest
}

func (o *SocialChannelFetchActivityCountUnauthorized) Error() string {
	return fmt.Sprintf("[POST /remote.api/SocialChannel.fetchActivityCount][%d] socialChannelFetchActivityCountUnauthorized  %+v", 401, o.Payload)
}

func (o *SocialChannelFetchActivityCountUnauthorized) readResponse(response runtime.ClientResponse, consumer runtime.Consumer, formats strfmt.Registry) error {

	o.Payload = new(models.UnauthorizedRequest)

	// response payload
	if err := consumer.Consume(response.Body(), o.Payload); err != nil && err != io.EOF {
		return err
	}

	return nil
}
