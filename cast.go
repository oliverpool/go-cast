package cast

import (
	"net"

	"golang.org/x/net/context"
)

type Scanner interface {
	// Scan scans for chromecast and pushes them onto the results channel (eventually multiple times)
	// It must close the results channel before returning when the ctx is done
	Scan(ctx context.Context, results chan<- *Chromecast) error
}

type Message struct {
	Header
	Payload
}

type Header struct {
	Type          string
	RequestID     *uint32
	DestinationID string
	SourceID      string
	Namespace     string
}

type Payload []byte

type Serializer interface {
	Send(payload interface{}, sourceId, destinationId, namespace string) error
	Receive() (*Message, error)
	Close() error
}

type IdentifiablePayload interface {
	SetRequestID(uint32)
}

type PayloadWithID struct {
	Type      string  `json:"type"`
	RequestID *uint32 `json:"requestId,omitempty"`
}

func (p *PayloadWithID) SetRequestID(id uint32) {
	p.RequestID = &id
}

/*
type Client interface {
	Listen(responseType string, ch chan<- Payload)
	Send(payload interface{}) error
	Request(payload IdentifiablePayload) (<-chan Payload, error)
	Dispatch() error
	Close() error
}
*/
