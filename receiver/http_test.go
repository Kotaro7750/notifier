package receiver

import (
	"testing"

	"github.com/Kotaro7750/notifier/test_util"
)

func TestHTTPReceiverBuilderUsesTypedProperties(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
listenAddress: :8080
`)

	component, err := HTTPReceiverBuilder("receiver-1", properties)
	if err != nil {
		t.Fatalf("HTTPReceiverBuilder returned error: %v", err)
	}

	impl := component.(*Receiver).impl.(*HTTPReceiverImpl)
	if impl.listenAddr != ":8080" {
		t.Fatalf("listenAddr = %q, want %q", impl.listenAddr, ":8080")
	}
}

func TestHTTPReceiverBuilderRequiresListenAddress(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
defaultSubscriber: ignored@example.com
`)

	if _, err := HTTPReceiverBuilder("receiver-1", properties); err == nil {
		t.Fatal("HTTPReceiverBuilder unexpectedly succeeded")
	}
}
