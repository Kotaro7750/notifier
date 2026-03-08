package sender

import (
	"testing"
	"time"

	"github.com/Kotaro7750/notifier/test_util"
	"gopkg.in/yaml.v3"
)

func TestDummySenderBuilderUsesDefaults(t *testing.T) {
	component, err := DummySenderBuilder("sender-1", yaml.Node{})
	if err != nil {
		t.Fatalf("DummySenderBuilder returned error: %v", err)
	}

	impl := component.(*Sender).impl.(*dummySenderImpl)
	if impl.errorInterval != 10*time.Second {
		t.Fatalf("errorInterval = %v, want %v", impl.errorInterval, 10*time.Second)
	}
	if impl.shutdownDuration != 5*time.Second {
		t.Fatalf("shutdownDuration = %v, want %v", impl.shutdownDuration, 5*time.Second)
	}
}

func TestDummySenderBuilderUsesTypedProperties(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
errorInterval: 0s
shutdownDuration: 2s
`)

	component, err := DummySenderBuilder("sender-1", properties)
	if err != nil {
		t.Fatalf("DummySenderBuilder returned error: %v", err)
	}

	impl := component.(*Sender).impl.(*dummySenderImpl)
	if impl.errorInterval != 0 {
		t.Fatalf("errorInterval = %v, want 0", impl.errorInterval)
	}
	if impl.shutdownDuration != 2*time.Second {
		t.Fatalf("shutdownDuration = %v, want %v", impl.shutdownDuration, 2*time.Second)
	}
}
