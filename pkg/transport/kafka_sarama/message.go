package kafka_sarama

import (
	"bytes"
	"context"
	"strings"

	"github.com/pkg/errors"

	ce "github.com/cloudevents/sdk-go"
	"github.com/cloudevents/sdk-go/pkg/binding"
	"github.com/cloudevents/sdk-go/pkg/binding/format"
	"github.com/cloudevents/sdk-go/pkg/binding/spec"

	"github.com/Shopify/sarama"
)

const (
	prefix            = "ce_"
	contentTypeHeader = "content-type"
)

var specs = spec.WithPrefix(prefix)

// Message holds a Kafka Message.
// This message *can* be read several times safely
type Message struct {
	Key, Value  []byte
	Headers     map[string][]byte
	ContentType string
	format      format.Format
	version     spec.Version
}

// Check if http.Message implements binding.Message
var _ binding.Message = (*Message)(nil)

// Returns a binding.Message that holds the provided ConsumerMessage.
// The returned binding.Message *can* be read several times safely
// This function *doesn't* guarantee that the returned binding.Message is always a kafka_sarama.Message instance
func NewMessageFromConsumerMessage(cm *sarama.ConsumerMessage) (binding.Message, error) {
	var contentType string
	headers := make(map[string][]byte, len(cm.Headers))
	for _, r := range cm.Headers {
		k := strings.ToLower(string(r.Key))
		if k == contentTypeHeader {
			contentType = string(r.Value)
		}
		headers[k] = r.Value
	}
	return NewMessage(cm.Key, cm.Value, contentType, headers)
}

// Returns a binding.Message that holds the provided kafka message components.
// The returned binding.Message *can* be read several times safely
// This function *doesn't* guarantee that the returned binding.Message is always a kafka_sarama.Message instance
func NewMessage(key []byte, value []byte, contentType string, headers map[string][]byte) (binding.Message, error) {
	if ft := format.Lookup(contentType); ft != nil {
		// If the message is structured and has a key,
		// then it's cheaper to go through event message
		// because we need to add the key as extension
		if key != nil {
			event := ce.Event{}
			err := ft.Unmarshal(value, &event)
			if err != nil {
				return nil, errors.Wrap(err, "Error while trying to convert the kafka message to event")
			}
			event.SetExtension("key", string(key))
			return binding.EventMessage(event), nil
		} else {
			return &Message{
				Key:         key,
				Value:       value,
				ContentType: contentType,
				Headers:     headers,
				format:      ft,
			}, nil
		}
	} else if v := specs.Version(string(headers[specs.PrefixedSpecVersionName()])); v != nil {
		return &Message{
			Key:         key,
			Value:       value,
			ContentType: contentType,
			Headers:     headers,
			version:     v,
		}, nil
	}

	return &Message{
		Key:         key,
		Value:       value,
		ContentType: contentType,
		Headers:     headers,
	}, nil
}

func (m *Message) ReadEncoding() binding.Encoding {
	if m.version != nil {
		return binding.EncodingBinary
	}
	if m.format != nil {
		return binding.EncodingStructured
	}
	return binding.EncodingUnknown
}

func (m *Message) ReadStructured(ctx context.Context, encoder binding.StructuredWriter) error {
	if m.format != nil {
		return encoder.SetStructuredEvent(ctx, m.format, bytes.NewReader(m.Value))
	}
	return binding.ErrNotStructured
}

func (m *Message) ReadBinary(ctx context.Context, encoder binding.BinaryWriter) error {
	if m.version == nil {
		return binding.ErrNotBinary
	}

	err := encoder.Start(ctx)
	if err != nil {
		return err
	}

	for k, v := range m.Headers {
		if strings.HasPrefix(k, prefix) {
			attr := m.version.Attribute(k)
			if attr != nil {
				err = encoder.SetAttribute(attr, string(v))
			} else {
				err = encoder.SetExtension(strings.TrimPrefix(k, prefix), string(v))
			}
		} else if k == contentTypeHeader {
			err = encoder.SetAttribute(m.version.AttributeFromKind(spec.DataContentType), string(v))
		}
		if err != nil {
			return err
		}
	}

	if m.Key != nil {
		err = encoder.SetExtension("key", string(m.Key))
		if err != nil {
			return err
		}
	}

	if m.Value != nil {
		err = encoder.SetData(bytes.NewReader(m.Value))
		if err != nil {
			return err
		}
	}

	return encoder.End()
}

func (m *Message) Finish(error) error {
	return nil
}
