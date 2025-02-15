package receiver

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/notification"
)

func DummyReceiverBuilder(id string, properties map[string]interface{}) (abstraction.AbstractChannelComponent, error) {
	return NewReceiver(&dummyReceiverImpl{
		id:     id,
		logger: nil,
	}), nil
}

type dummyReceiverImpl struct {
	id     string
	logger *slog.Logger
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
		time.Sleep(6 * time.Second)
	}

	go func() {
		defer close(retCh)

		c := time.Tick(1 * time.Second)
		d := time.Tick(10 * time.Second)
		for {
			select {
			case <-c:
				outputCh <- notification.Notification{
					Title:    "Dummy Title",
					Message:  fmt.Sprintf("Hello from %s", dri.id),
					Severity: slog.LevelInfo,
				}

			case <-d:
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
