package sender

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/notification"

	"gopkg.in/yaml.v3"
)

type DummySenderProperties struct {
	ErrorInterval    time.Duration `yaml:"errorInterval"`
	ShutdownDuration time.Duration `yaml:"shutdownDuration"`
}

func NewDummySenderProperties() DummySenderProperties {
	return DummySenderProperties{
		ErrorInterval:    10 * time.Second,
		ShutdownDuration: 5 * time.Second,
	}
}

func (p DummySenderProperties) Validate() error {
	if p.ErrorInterval < 0 {
		return fmt.Errorf("errorInterval should be greater than or equal to 0")
	}
	if p.ShutdownDuration < 0 {
		return fmt.Errorf("shutdownDuration should be greater than or equal to 0")
	}

	return nil
}

func DummySenderBuilder(id string, properties yaml.Node) (abstraction.AbstractChannelComponent, error) {
	parsedProperties := NewDummySenderProperties()
	if err := abstraction.DecodeProperties(properties, &parsedProperties); err != nil {
		return nil, err
	}
	if err := parsedProperties.Validate(); err != nil {
		return nil, err
	}

	return NewSender(&dummySenderImpl{
		id:               id,
		logger:           nil,
		errorInterval:    parsedProperties.ErrorInterval,
		shutdownDuration: parsedProperties.ShutdownDuration,
	}), nil
}

type dummySenderImpl struct {
	id               string
	logger           *slog.Logger
	errorInterval    time.Duration
	shutdownDuration time.Duration
}

func (dsi *dummySenderImpl) GetId() string {
	return fmt.Sprintf("%s", dsi.id)
}

func (dsi *dummySenderImpl) GetLogger() *slog.Logger {
	return dsi.logger
}

func (dsi *dummySenderImpl) SetLogger(logger *slog.Logger) {
	dsi.logger = logger
}

func (dsi *dummySenderImpl) Start(inputCh <-chan notification.Notification, done <-chan struct{}) <-chan error {
	retCh := make(chan error)

	shutdownFunc := func() {
		time.Sleep(dsi.shutdownDuration)
	}

	go func() {
		defer close(retCh)

		var errorTickCh <-chan time.Time
		if dsi.errorInterval > 0 {
			errorTicker := time.NewTicker(dsi.errorInterval)
			defer errorTicker.Stop()
			errorTickCh = errorTicker.C
		}

		for {
			select {
			case n, ok := <-inputCh:
				if !ok {
					dsi.GetLogger().Info("inputCh closed")
				} else {
					dsi.GetLogger().Info("Notify send from dummySender", "notification", n)
				}

			case <-errorTickCh:
				shutdownFunc()
				retCh <- fmt.Errorf("timeout")
				return

			case <-done:
				shutdownFunc()
				return
			}
		}
	}()

	return retCh
}
