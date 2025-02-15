package sender

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/notification"
)

func DummySenderBuilder(id string, properties map[string]interface{}) (abstraction.AbstractChannelComponent, error) {
	return NewSender(&dummySenderImpl{
		id:     id,
		logger: nil,
	}), nil
}

type dummySenderImpl struct {
	id     string
	logger *slog.Logger
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
		time.Sleep(5 * time.Second)
	}

	go func() {
		defer close(retCh)

		c := time.Tick(10 * time.Second)
		for {
			select {
			case n, ok := <-inputCh:
				if !ok {
					dsi.GetLogger().Info("inputCh closed")
				} else {
					dsi.GetLogger().Info("Notify send from dummySender", "notification", n)
				}

			case <-c:
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
