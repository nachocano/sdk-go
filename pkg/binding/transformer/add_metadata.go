package transformer

import (
	"fmt"

	"github.com/cloudevents/sdk-go/pkg/binding"
	"github.com/cloudevents/sdk-go/pkg/binding/spec"
	"github.com/cloudevents/sdk-go/pkg/event"
)

// Add cloudevents attribute (if missing) during the encoding process
func AddAttribute(attributeKind spec.Kind, value interface{}) binding.TransformerFactory {
	return setAttributeTranscoderFactory{attributeKind: attributeKind, value: value}
}

// Add cloudevents extension (if missing) during the encoding process
func AddExtension(name string, value interface{}) binding.TransformerFactory {
	return setExtensionTranscoderFactory{name: name, value: value}
}

type setAttributeTranscoderFactory struct {
	attributeKind spec.Kind
	value         interface{}
}

func (a setAttributeTranscoderFactory) StructuredTransformer(binding.StructuredWriter) binding.StructuredWriter {
	return nil
}

func (a setAttributeTranscoderFactory) BinaryTransformer(encoder binding.BinaryWriter) binding.BinaryWriter {
	return &setAttributeTransformer{
		BinaryWriter:  encoder,
		attributeKind: a.attributeKind,
		value:         a.value,
		found:         false,
	}
}

func (a setAttributeTranscoderFactory) EventTransformer() binding.EventTransformer {
	return func(event *event.Event) error {
		v := spec.VS.Version(event.SpecVersion())
		if v == nil {
			return fmt.Errorf("spec version %s invalid", event.SpecVersion())
		}
		if v.AttributeFromKind(a.attributeKind).Get(event.Context) == nil {
			return v.AttributeFromKind(a.attributeKind).Set(event.Context, a.value)
		}
		return nil
	}
}

type setExtensionTranscoderFactory struct {
	name  string
	value interface{}
}

func (a setExtensionTranscoderFactory) StructuredTransformer(binding.StructuredWriter) binding.StructuredWriter {
	return nil
}

func (a setExtensionTranscoderFactory) BinaryTransformer(encoder binding.BinaryWriter) binding.BinaryWriter {
	return &setExtensionTransformer{
		BinaryWriter: encoder,
		name:         a.name,
		value:        a.value,
		found:        false,
	}
}

func (a setExtensionTranscoderFactory) EventTransformer() binding.EventTransformer {
	return func(event *event.Event) error {
		if _, ok := event.Extensions()[a.name]; !ok {
			return event.Context.SetExtension(a.name, a.value)
		}
		return nil
	}
}

type setAttributeTransformer struct {
	binding.BinaryWriter
	attributeKind spec.Kind
	value         interface{}
	version       spec.Version
	found         bool
}

func (b *setAttributeTransformer) SetAttribute(attribute spec.Attribute, value interface{}) error {
	if attribute.Kind() == b.attributeKind {
		b.found = true
	}
	b.version = attribute.Version()
	return b.BinaryWriter.SetAttribute(attribute, value)
}

func (b *setAttributeTransformer) End() error {
	if !b.found {
		err := b.BinaryWriter.SetAttribute(b.version.AttributeFromKind(b.attributeKind), b.value)
		if err != nil {
			return err
		}
	}
	return b.BinaryWriter.End()
}

type setExtensionTransformer struct {
	binding.BinaryWriter
	name  string
	value interface{}
	found bool
}

func (b *setExtensionTransformer) SetExtension(name string, value interface{}) error {
	if name == b.name {
		b.found = true
	}
	return b.BinaryWriter.SetExtension(name, value)
}

func (b *setExtensionTransformer) End() error {
	if !b.found {
		err := b.BinaryWriter.SetExtension(b.name, b.value)
		if err != nil {
			return err
		}
	}
	return b.BinaryWriter.End()
}
