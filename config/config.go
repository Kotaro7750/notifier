package config

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

type Configuration struct {
	ReceiverConfigurations []ChannelComponentConfig `yaml:"receivers,flow"`
	SenderConfigurations   []ChannelComponentConfig `yaml:"senders,flow"`
}

func (c Configuration) Validate() error {
	if c.ReceiverConfigurations == nil {
		return fmt.Errorf("receivers is not defined")
	}

	if len(c.ReceiverConfigurations) == 0 {
		return fmt.Errorf("At least one receiver is required")
	}

	for _, receiverConfig := range c.ReceiverConfigurations {
		if err := receiverConfig.Validate(); err != nil {
			return err
		}
		if receiverConfig.Match != nil {
			return fmt.Errorf("receiver %s does not support match", receiverConfig.Id)
		}
	}

	if c.SenderConfigurations == nil {
		return fmt.Errorf("senders is not defined")
	}

	if len(c.SenderConfigurations) == 0 {
		return fmt.Errorf("At least one sender is required")
	}

	for _, senderConfig := range c.SenderConfigurations {
		if err := senderConfig.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// ChannelComponentConfig holds the first-stage decoded YAML for a channel component.
// The top-level config is decoded into this shared shape first, and Properties is then
// decoded a second time into a component-specific typed properties struct inside each builder.
type ChannelComponentConfig struct {
	Id         string             `yaml:"id"`
	Kind       string             `yaml:"kind"`
	Match      *MetadataCondition `yaml:"match,omitempty"`
	Properties yaml.Node          `yaml:"properties"`
}

func (c ChannelComponentConfig) Validate() error {
	if c.Id == "" {
		return fmt.Errorf("id is required")
	}

	if c.Kind == "" {
		return fmt.Errorf("kind is required")
	}

	if c.Match != nil {
		if err := c.Match.Validate(); err != nil {
			return fmt.Errorf("match is invalid: %w", err)
		}
	}

	return nil
}

type MetadataCondition struct {
	NotificationSource string            `yaml:"notification_source"`
	Labels             map[string]string `yaml:"labels"`
}

func (m MetadataCondition) Validate() error {
	if !m.HasConditions() {
		return fmt.Errorf("at least one match condition is required")
	}

	if len(NormalizeCSVValues(m.NotificationSource)) == 0 && strings.TrimSpace(m.NotificationSource) != "" {
		return fmt.Errorf("notification_source must contain at least one non-empty value")
	}

	for key, value := range m.Labels {
		if strings.TrimSpace(key) == "" {
			return fmt.Errorf("labels must not contain an empty key")
		}
		if len(NormalizeCSVValues(value)) == 0 {
			return fmt.Errorf("labels.%s must contain at least one non-empty value", key)
		}
	}

	return nil
}

func (m MetadataCondition) HasConditions() bool {
	return strings.TrimSpace(m.NotificationSource) != "" || len(m.Labels) > 0
}

func NormalizeCSVValues(raw string) []string {
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value == "" {
			continue
		}
		values = append(values, value)
	}
	return values
}

// DecodeProperties decodes component-specific properties from the shared YAML node into a
// concrete typed configuration struct used by a sender or receiver builder.
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
