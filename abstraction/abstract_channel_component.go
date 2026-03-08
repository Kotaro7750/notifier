package abstraction

import (
	"bytes"
	"fmt"
	"log/slog"

	"github.com/Kotaro7750/notifier/notification"
	"gopkg.in/yaml.v3"
)

type AbstractChannelComponentBuilder func(id string, properties yaml.Node) (AbstractChannelComponent, error)

type AbstractChannelComponent interface {
	GetId() string
	GetLogger() *slog.Logger
	SetLogger(logger *slog.Logger)
	Start(ch chan notification.Notification, done <-chan struct{}) <-chan error
}

// AbstractChannelComponentConfig holds the first-stage decoded YAML for a channel component.
// The top-level config is decoded into this shared shape first, and Properties is then
// decoded a second time into a component-specific typed properties struct inside each builder.
type AbstractChannelComponentConfig struct {
	Id         string    `yaml:"id"`
	Kind       string    `yaml:"kind"`
	Properties yaml.Node `yaml:"properties"`
}

func (c *AbstractChannelComponentConfig) Validate() error {
	if c.Id == "" {
		return fmt.Errorf("id is required")
	}

	if c.Kind == "" {
		return fmt.Errorf("kind is required")
	}

	return nil
}

// Helper function to decode properties of AbstractChannelComponentConfig into concrete Configuration struct
func DecodeProperties(properties yaml.Node, target any) error {
	// If properties is unset, skip decoding
	if properties.Kind == 0 {
		return nil
	}

	body, err := yaml.Marshal(properties)
	if err != nil {
		return fmt.Errorf("marshal properties: %w", err)
	}

	decoder := yaml.NewDecoder(bytes.NewReader(body))
	decoder.KnownFields(true)

	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("decode properties: %w", err)
	}

	return nil
}
