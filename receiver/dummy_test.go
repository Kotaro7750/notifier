package receiver

import (
	"testing"
	"time"

	"github.com/Kotaro7750/notifier/test_util"
	"gopkg.in/yaml.v3"
)

func TestDummyReceiverBuilderUsesDefaults(t *testing.T) {
	component, err := DummyReceiverBuilder("receiver-1", yaml.Node{})
	if err != nil {
		t.Fatalf("DummyReceiverBuilder returned error: %v", err)
	}

	impl := component.(*Receiver).impl.(*dummyReceiverImpl)
	if impl.errorInterval != 10*time.Second {
		t.Fatalf("errorInterval = %v, want %v", impl.errorInterval, 10*time.Second)
	}
	if impl.shutdownDuration != 6*time.Second {
		t.Fatalf("shutdownDuration = %v, want %v", impl.shutdownDuration, 6*time.Second)
	}
	if impl.receiveInterval != 1*time.Second {
		t.Fatalf("receiveInterval = %v, want %v", impl.receiveInterval, 1*time.Second)
	}
	if impl.notificationSource != "" {
		t.Fatalf("notificationSource = %q, want empty", impl.notificationSource)
	}
	if impl.labels != nil {
		t.Fatalf("labels = %v, want nil", impl.labels)
	}
}

func TestDummyReceiverBuilderUsesTypedProperties(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
errorInterval: 0s
shutdownDuration: 2s
receiveInterval: 3s
notification_source: billing
labels:
  env: prod
  team: ops
`)

	component, err := DummyReceiverBuilder("receiver-1", properties)
	if err != nil {
		t.Fatalf("DummyReceiverBuilder returned error: %v", err)
	}

	impl := component.(*Receiver).impl.(*dummyReceiverImpl)
	if impl.errorInterval != 0 {
		t.Fatalf("errorInterval = %v, want 0", impl.errorInterval)
	}
	if impl.shutdownDuration != 2*time.Second {
		t.Fatalf("shutdownDuration = %v, want %v", impl.shutdownDuration, 2*time.Second)
	}
	if impl.receiveInterval != 3*time.Second {
		t.Fatalf("receiveInterval = %v, want %v", impl.receiveInterval, 3*time.Second)
	}
	if impl.notificationSource != "billing" {
		t.Fatalf("notificationSource = %q, want %q", impl.notificationSource, "billing")
	}
	if got := impl.labels["env"]; got != "prod" {
		t.Fatalf("labels[env] = %q, want %q", got, "prod")
	}
	if got := impl.labels["team"]; got != "ops" {
		t.Fatalf("labels[team] = %q, want %q", got, "ops")
	}
}

func TestDummyReceiverBuilderRejectsZeroReceiveInterval(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
receiveInterval: 0s
`)

	if _, err := DummyReceiverBuilder("receiver-1", properties); err == nil {
		t.Fatal("DummyReceiverBuilder unexpectedly succeeded")
	}
}
