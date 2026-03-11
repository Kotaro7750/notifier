package config

import (
	"bytes"
	"testing"
	"time"

	"github.com/Kotaro7750/notifier/test_util"
	"gopkg.in/yaml.v3"
)

func TestDecodePropertiesRejectsUnknownFields(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
known: 1s
unknown: 2s
`)

	var parsed struct {
		Known time.Duration `yaml:"known"`
	}

	if err := DecodeProperties(properties, &parsed); err == nil {
		t.Fatal("DecodeProperties unexpectedly succeeded")
	}
}

func TestConfigurationValidateAcceptsMetadataCondition(t *testing.T) {
	cfg := mustDecodeConfiguration(t, `
receivers:
  - id: receiver-1
    kind: dummy
    properties: {}
senders:
  - id: sender-1
    kind: dummy
    match:
      notification_source: billing,payments
      labels:
        env: prod,stg
    properties: {}
`)

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Configuration.Validate() returned error: %v", err)
	}
}

func TestConfigurationValidateRejectsReceiverMatch(t *testing.T) {
	cfg := mustDecodeConfiguration(t, `
receivers:
  - id: receiver-1
    kind: dummy
    match:
      notification_source: billing
    properties: {}
senders:
  - id: sender-1
    kind: dummy
    properties: {}
`)

	if err := cfg.Validate(); err == nil {
		t.Fatal("Configuration.Validate() unexpectedly succeeded")
	}
}

func TestDecodeConfigurationRejectsUnknownMatchField(t *testing.T) {
	if _, err := decodeConfiguration(`
receivers:
  - id: receiver-1
    kind: dummy
    properties: {}
senders:
  - id: sender-1
    kind: dummy
    match:
      unknown: billing
    properties: {}
`); err == nil {
		t.Fatal("decodeConfiguration() unexpectedly succeeded")
	}
}

func TestDecodeConfigurationRejectsInvalidLabelsType(t *testing.T) {
	if _, err := decodeConfiguration(`
receivers:
  - id: receiver-1
    kind: dummy
    properties: {}
senders:
  - id: sender-1
    kind: dummy
    match:
      labels: prod,stg
    properties: {}
`); err == nil {
		t.Fatal("decodeConfiguration() unexpectedly succeeded")
	}
}

func mustDecodeConfiguration(t *testing.T, raw string) Configuration {
	t.Helper()

	cfg, err := decodeConfiguration(raw)
	if err != nil {
		t.Fatalf("decodeConfiguration() returned error: %v", err)
	}

	return cfg
}

func decodeConfiguration(raw string) (Configuration, error) {
	var cfg Configuration

	decoder := yaml.NewDecoder(bytes.NewReader([]byte(raw)))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return Configuration{}, err
	}

	return cfg, nil
}
