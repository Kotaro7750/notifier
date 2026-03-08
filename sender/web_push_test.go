package sender

import (
	"testing"

	"github.com/Kotaro7750/notifier/test_util"
)

func TestWebPushSenderBuilderUsesTypedProperties(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
listenAddress: :8091
repositoryType: InMemory
defaultSubscriber: tester@example.com
`)

	component, err := WebPushSenderBuilder("sender-1", properties)
	if err != nil {
		t.Fatalf("WebPushSenderBuilder returned error: %v", err)
	}

	impl := component.(*Sender).impl.(*webPushSenderImpl)
	if impl.listenAddress != ":8091" {
		t.Fatalf("listenAddress = %q, want %q", impl.listenAddress, ":8091")
	}
	if impl.defaultSubscriber != "tester@example.com" {
		t.Fatalf("defaultSubscriber = %q, want %q", impl.defaultSubscriber, "tester@example.com")
	}
	if _, ok := impl.subscriptionRepository.(*InMemorySubscriptionRepository); !ok {
		t.Fatalf("subscriptionRepository = %T, want *InMemorySubscriptionRepository", impl.subscriptionRepository)
	}
}

func TestWebPushSenderBuilderRequiresRepositoryType(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
listenAddress: :8091
`)

	if _, err := WebPushSenderBuilder("sender-1", properties); err == nil {
		t.Fatal("WebPushSenderBuilder unexpectedly succeeded")
	}
}
