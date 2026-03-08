package sender

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/DataDog/datadog-api-client-go/v2/api/datadog"
	"github.com/DataDog/datadog-api-client-go/v2/api/datadogV1"
	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/notification"

	"gopkg.in/yaml.v3"
)

const defaultDatadogEventRequestTimeout = 10 * time.Second

type DatadogEventSenderProperties struct {
	Site   string `yaml:"site"`
	APIKey string `yaml:"apiKey"`
}

func (p DatadogEventSenderProperties) Validate() error {
	if strings.TrimSpace(p.Site) == "" {
		return fmt.Errorf("site is required")
	}

	if strings.TrimSpace(p.APIKey) == "" {
		return fmt.Errorf("apiKey is required")
	}

	return nil
}

func (p DatadogEventSenderProperties) eventsURL() string {
	site := strings.TrimSpace(p.Site)
	site = strings.TrimPrefix(site, "https://")
	site = strings.TrimPrefix(site, "http://")
	site = strings.TrimSuffix(site, "/")

	return site
}

func DatadogEventSenderBuilder(id string, properties yaml.Node) (abstraction.AbstractChannelComponent, error) {
	var parsedProperties DatadogEventSenderProperties
	if err := abstraction.DecodeProperties(properties, &parsedProperties); err != nil {
		return nil, err
	}

	if err := parsedProperties.Validate(); err != nil {
		return nil, err
	}

	site := parsedProperties.eventsURL()
	configuration := datadog.NewConfiguration()
	configuration.HTTPClient = &http.Client{
		Timeout: defaultDatadogEventRequestTimeout,
	}

	ctx := context.WithValue(
		context.Background(),
		datadog.ContextServerVariables,
		map[string]string{"site": site},
	)
	ctx = context.WithValue(
		ctx,
		datadog.ContextAPIKeys,
		map[string]datadog.APIKey{
			"apiKeyAuth": {
				Key: parsedProperties.APIKey,
			},
		},
	)

	return NewSender(&datadogEventSenderImpl{
		id:        id,
		logger:    nil,
		site:      site,
		apiKey:    parsedProperties.APIKey,
		ctx:       ctx,
		eventsAPI: datadogV1.NewEventsApi(datadog.NewAPIClient(configuration)),
	}), nil
}

type datadogEventSenderImpl struct {
	id        string
	logger    *slog.Logger
	site      string
	apiKey    string
	ctx       context.Context
	eventsAPI *datadogV1.EventsApi
}

func (dsi *datadogEventSenderImpl) GetId() string {
	return fmt.Sprintf("%s", dsi.id)
}

func (dsi *datadogEventSenderImpl) GetLogger() *slog.Logger {
	return dsi.logger
}

func (dsi *datadogEventSenderImpl) SetLogger(logger *slog.Logger) {
	dsi.logger = logger
}

func (dsi *datadogEventSenderImpl) Start(inputCh <-chan notification.Notification, done <-chan struct{}) <-chan error {
	retCh := make(chan error)

	go func() {
		defer close(retCh)

		for {
			select {
			case n, ok := <-inputCh:
				if !ok {
					dsi.GetLogger().Info("inputCh closed")
					return
				}

				if err := dsi.send(n); err != nil {
					retCh <- err
					return
				}

			case <-done:
				return
			}
		}
	}()

	return retCh
}

func (dsi *datadogEventSenderImpl) send(n notification.Notification) error {
	body := *datadogV1.NewEventCreateRequest(n.Message, n.Title)
	body.SetAlertType(datadogAlertType(n.Severity))

	_, resp, err := dsi.eventsAPI.CreateEvent(dsi.ctx, body)
	if err != nil {
		if resp != nil {
			return fmt.Errorf("send datadog event: status %s: %w", resp.Status, err)
		}
		return fmt.Errorf("send datadog event: %w", err)
	}

	return nil
}

func datadogAlertType(severity slog.Level) datadogV1.EventAlertType {
	switch {
	case severity >= slog.LevelError:
		return datadogV1.EVENTALERTTYPE_ERROR
	case severity >= slog.LevelWarn:
		return datadogV1.EVENTALERTTYPE_WARNING
	default:
		return datadogV1.EVENTALERTTYPE_INFO
	}
}
