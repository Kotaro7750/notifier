package abstraction

import (
	"log/slog"
	"sync"
	"time"

	"github.com/Kotaro7750/notifier/notification"
)

type AutonomousChannelComponent struct {
	chanComponent  AbstractChannelComponent
	ch             chan notification.Notification
	shutdownCh     chan struct{}
	isStarted      bool
	isShuttingDown bool
	lock           sync.Mutex
}

func NewAutonomousChannelComponent(chanComponent AbstractChannelComponent) *AutonomousChannelComponent {
	return &AutonomousChannelComponent{
		chanComponent:  chanComponent,
		ch:             make(chan notification.Notification),
		shutdownCh:     make(chan struct{}),
		isStarted:      false,
		isShuttingDown: false,
		lock:           sync.Mutex{},
	}
}

func (acc *AutonomousChannelComponent) Start() <-chan struct{} {
	acc.chanComponent.GetLogger().Info("Start invoked")
	acc.lock.Lock()
	defer acc.lock.Unlock()

	if acc.isStarted {
		acc.chanComponent.GetLogger().Info("Already started")
		return nil
	}
	acc.isStarted = true
	acc.shutdownCh = make(chan struct{})

	completedCh := make(chan struct{})

	go func(stopCh <-chan struct{}) {
		defer close(completedCh)
		for {
			acc.chanComponent.GetLogger().Info("Starting")
			select {
			case err := <-acc.chanComponent.Start(acc.ch, stopCh):
				if err != nil {
					acc.chanComponent.GetLogger().Error("Error in channel component", "error", err)
				}

				acc.lock.Lock()
				if acc.isShuttingDown {
					defer acc.lock.Unlock()
					defer close(acc.ch)
					acc.chanComponent.GetLogger().Info("Shutting down")

					acc.isStarted = false
					acc.isShuttingDown = false
					return
				}
				acc.lock.Unlock()
			}

			acc.chanComponent.GetLogger().Info("Restart after 1s")

			time.Sleep(1 * time.Second)
		}
	}(acc.shutdownCh)

	return completedCh
}

func (acc *AutonomousChannelComponent) GetChannel() chan notification.Notification {
	return acc.ch
}

func (acc *AutonomousChannelComponent) GetLogger() *slog.Logger {
	return acc.chanComponent.GetLogger()
}

func (acc *AutonomousChannelComponent) Shutdown() {
	acc.chanComponent.GetLogger().Info("Shutdown invoked")
	acc.lock.Lock()
	defer acc.lock.Unlock()

	if !acc.isStarted {
		acc.chanComponent.GetLogger().Info("Not started")
		return
	}
	acc.isShuttingDown = true

	close(acc.shutdownCh)
	return
}
