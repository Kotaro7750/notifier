package sender

import (
	"testing"

	"github.com/Kotaro7750/notifier/test_util"
)

func TestDatadogEventSenderBuilderUsesTypedProperties(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
site: datadoghq.com
apiKey: test-api-key
`)

	component, err := DatadogEventSenderBuilder("sender-1", properties)
	if err != nil {
		t.Fatalf("DatadogEventSenderBuilder returned error: %v", err)
	}

	impl := component.(*Sender).impl.(*datadogEventSenderImpl)
	if impl.site != "datadoghq.com" {
		t.Fatalf("site = %q, want %q", impl.site, "datadoghq.com")
	}
	if impl.apiKey != "test-api-key" {
		t.Fatalf("apiKey = %q, want %q", impl.apiKey, "test-api-key")
	}
	if impl.eventsAPI == nil {
		t.Fatal("eventsAPI is nil")
	}
}

func TestDatadogEventSenderBuilderRequiresSite(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
apiKey: test-api-key
`)

	if _, err := DatadogEventSenderBuilder("sender-1", properties); err == nil {
		t.Fatal("DatadogEventSenderBuilder unexpectedly succeeded")
	}
}

func TestDatadogEventSenderBuilderRequiresAPIKey(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
site: datadoghq.com
`)

	if _, err := DatadogEventSenderBuilder("sender-1", properties); err == nil {
		t.Fatal("DatadogEventSenderBuilder unexpectedly succeeded")
	}
}

func TestDatadogEventSenderBuilderRejectsUnknownProperties(t *testing.T) {
	properties := test_util.MustPropertiesNode(t, `
site: datadoghq.com
apiKey: test-api-key
unknown: value
`)

	if _, err := DatadogEventSenderBuilder("sender-1", properties); err == nil {
		t.Fatal("DatadogEventSenderBuilder unexpectedly succeeded")
	}
}
