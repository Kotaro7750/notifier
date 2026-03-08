package receiver

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/notification"

	"gopkg.in/yaml.v3"
)

type DummyReceiverProperties struct {
	ErrorInterval    time.Duration `yaml:"errorInterval"`
	ShutdownDuration time.Duration `yaml:"shutdownDuration"`
	ReceiveInterval  time.Duration `yaml:"receiveInterval"`
}

func NewDummyReceiverProperties() DummyReceiverProperties {
	return DummyReceiverProperties{
		ErrorInterval:    10 * time.Second,
		ShutdownDuration: 6 * time.Second,
		ReceiveInterval:  1 * time.Second,
	}
}

func (p DummyReceiverProperties) Validate() error {
	if p.ErrorInterval < 0 {
		return fmt.Errorf("errorInterval should be greater than or equal to 0")
	}
	if p.ShutdownDuration < 0 {
		return fmt.Errorf("shutdownDuration should be greater than or equal to 0")
	}
	if p.ReceiveInterval <= 0 {
		return fmt.Errorf("receiveInterval should be greater than 0")
	}

	return nil
}

func DummyReceiverBuilder(id string, properties yaml.Node) (abstraction.AbstractChannelComponent, error) {
	parsedProperties := NewDummyReceiverProperties()
	if err := abstraction.DecodeProperties(properties, &parsedProperties); err != nil {
		return nil, err
	}
	if err := parsedProperties.Validate(); err != nil {
		return nil, err
	}

	return NewReceiver(&dummyReceiverImpl{
		id:               id,
		logger:           nil,
		errorInterval:    parsedProperties.ErrorInterval,
		shutdownDuration: parsedProperties.ShutdownDuration,
		receiveInterval:  parsedProperties.ReceiveInterval,
	}), nil
}

type dummyReceiverImpl struct {
	id               string
	logger           *slog.Logger
	errorInterval    time.Duration
	shutdownDuration time.Duration
	receiveInterval  time.Duration
}

func (dri *dummyReceiverImpl) GetId() string {
	return fmt.Sprintf("%s", dri.id)
}

func (dri *dummyReceiverImpl) GetLogger() *slog.Logger {
	return dri.logger
}

func (dri *dummyReceiverImpl) SetLogger(logger *slog.Logger) {
	dri.logger = logger
}

func (dri *dummyReceiverImpl) Start(outputCh chan<- notification.Notification, done <-chan struct{}) <-chan error {
	retCh := make(chan error)

	shutdownFunc := func() {
		time.Sleep(dri.shutdownDuration)
	}

	go func() {
		defer close(retCh)

		var receiveTickCh <-chan time.Time
		if dri.receiveInterval > 0 {
			receiveTicker := time.NewTicker(dri.receiveInterval)
			defer receiveTicker.Stop()
			receiveTickCh = receiveTicker.C
		}

		var errorTickCh <-chan time.Time
		if dri.errorInterval > 0 {
			errorTicker := time.NewTicker(dri.errorInterval)
			defer errorTicker.Stop()
			errorTickCh = errorTicker.C
		}

		for {
			select {
			case <-receiveTickCh:
				outputCh <- notification.Notification{
					Title:    "Dummy Title",
					Message:  fmt.Sprintf("Hello from %s", dri.id),
					Severity: slog.LevelInfo,
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
