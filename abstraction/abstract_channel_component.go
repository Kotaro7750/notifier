package abstraction

import (
	"fmt"
	"log/slog"

	"github.com/Kotaro7750/notifier/notification"
)

type AbstractChannelComponentBuilder func(id string, properties map[string]interface{}) (AbstractChannelComponent, error)

type AbstractChannelComponent interface {
	GetId() string
	GetLogger() *slog.Logger
	SetLogger(logger *slog.Logger)
	Start(ch chan notification.Notification, done <-chan struct{}) <-chan error
}

type AbstractChannelComponentConfig struct {
	Id         string                 `yaml:"id"`
	Kind       string                 `yaml:"kind"`
	Properties map[string]interface{} `yaml:"properties"`
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
