package transport

import (
	"context"

	"github.com/cloudevents/sdk-go/pkg/binding"
)

// Sender sends messages.
type Sender interface {
	// Send a message.
	//
	// Send returns when the "outbound" message has been sent. The Sender may
	// still be expecting acknowledgment or holding other state for the message.
	//
	// m.Finish() is called when sending is finished: expected acknowledgments (or
	// errors) have been received, the Sender is no longer holding any state for
	// the message. m.Finish() may be called during or after Send().
	Send(ctx context.Context, m binding.Message) error
}

// Requester sends a message and receives a response
//
// Optional interface that may be implemented by protocols that support
// request/response correlation.
type Requester interface {
	Sender

	// Request sends m like Sender.Send() but also arranges to receive a response.
	Request(ctx context.Context, m binding.Message) (binding.Message, error)
}

// SendCloser is a Sender that can be closed.
type SendCloser interface {
	Sender
	Closer
}
