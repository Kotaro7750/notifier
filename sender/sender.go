package sender

import (
	"log/slog"
	"sync"

	"github.com/Kotaro7750/notifier/config"
	"github.com/Kotaro7750/notifier/notification"
)

type Sender struct {
	impl  SenderImpl
	match MatchCondition
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
	if !s.match.hasConditions() {
		return s.impl.Start(inputCh, done)
	}

	filteredCh := make(chan notification.Notification)
	implErrCh := s.impl.Start(filteredCh, done)
	retCh := make(chan error)

	go func() {
		defer close(retCh)

		// The wrapper owns filteredCh and closes it exactly once when it stops forwarding.
		closeFilteredCh := sync.OnceFunc(func() {
			close(filteredCh)
		})

		finishWithImplResult := func(err error, ok bool) {
			closeFilteredCh()
			if ok {
				retCh <- err
			}
		}

		// Do not report wrapper shutdown until the wrapped sender has actually stopped.
		waitForImplStop := func() {
			err, ok := <-implErrCh
			finishWithImplResult(err, ok)
		}

		for {
			select {
			case <-done:
				waitForImplStop()
				return
			case err, ok := <-implErrCh:
				finishWithImplResult(err, ok)
				return
			case n, ok := <-inputCh:
				if !ok {
					waitForImplStop()
					return
				}
				if !s.match.IsMatched(n) {
					continue
				}
				// Also watch implErrCh here so a matched send cannot block forever after the
				// wrapped sender has already stopped consuming filteredCh.
				select {
				case filteredCh <- n:
				case err, ok := <-implErrCh:
					finishWithImplResult(err, ok)
					return
				case <-done:
					waitForImplStop()
					return
				}
			}
		}
	}()

	return retCh
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

func (s *Sender) SetMatch(match config.MetadataCondition) {
	s.match = NewMatchCondition(match)
}
