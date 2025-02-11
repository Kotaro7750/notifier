package receiver

import (
	"log/slog"

	"github.com/Kotaro7750/notifier/notification"
)

type Receiver struct {
	impl ReceiverImpl
}

func NewReceiver(impl ReceiverImpl) *Receiver {
	return &Receiver{impl: impl}
}

type ReceiverImpl interface {
	GetId() string
	GetLogger() *slog.Logger
	SetLogger(logger *slog.Logger)
	Start(outputCh chan<- notification.Notification, done <-chan struct{}) <-chan error
}

func (r *Receiver) Start(outputCh chan notification.Notification, done <-chan struct{}) <-chan error {
	return r.impl.Start(outputCh, done)
}
func (r *Receiver) GetId() string {
	return r.impl.GetId()
}

func (r *Receiver) GetLogger() *slog.Logger {
	return r.impl.GetLogger()
}

func (r *Receiver) SetLogger(logger *slog.Logger) {
	r.impl.SetLogger(logger)
}
