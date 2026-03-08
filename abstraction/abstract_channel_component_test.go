package abstraction

import (
	"testing"
	"time"

	"github.com/Kotaro7750/notifier/test_util"
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
