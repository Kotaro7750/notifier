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
}

func TestDummyReceiverBuilderUsesTypedProperties(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
errorInterval: 0s
shutdownDuration: 2s
receiveInterval: 3s
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
}

func TestDummyReceiverBuilderRejectsZeroReceiveInterval(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
receiveInterval: 0s
`)

	if _, err := DummyReceiverBuilder("receiver-1", properties); err == nil {
		t.Fatal("DummyReceiverBuilder unexpectedly succeeded")
	}
}
