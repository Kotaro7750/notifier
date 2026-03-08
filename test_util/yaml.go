package test_util

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func MustPropertiesNode(t *testing.T, body string) yaml.Node {
	t.Helper()

	var wrapper struct {
		Properties yaml.Node `yaml:"properties"`
	}

	source := "properties:\n" + indent(body)
	if err := yaml.Unmarshal([]byte(source), &wrapper); err != nil {
		t.Fatalf("yaml.Unmarshal returned error: %v", err)
	}

	return wrapper.Properties
}

func indent(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}

	return "  " + strings.ReplaceAll(trimmed, "\n", "\n  ") + "\n"
}
