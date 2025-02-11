package sender

import (
	"log/slog"

	"github.com/Kotaro7750/notifier/notification"
)

type Sender struct {
	impl SenderImpl
}

func NewSender(impl SenderImpl) *Sender {
	return &Sender{impl: impl}
}

type SenderImpl interface {
	GetId() string
	GetLogger() *slog.Logger
	SetLogger(logger *slog.Logger)
	Start(inputCh <-chan notification.Notification, done <-chan struct{}) <-chan error
}

func (s *Sender) Start(inputCh chan notification.Notification, done <-chan struct{}) <-chan error {
	return s.impl.Start(inputCh, done)
}

func (s *Sender) GetLogger() *slog.Logger {
	return s.impl.GetLogger()
}

func (s *Sender) SetLogger(logger *slog.Logger) {
	s.impl.SetLogger(logger)
}

func (s *Sender) GetId() string {
	return s.impl.GetId()
}
