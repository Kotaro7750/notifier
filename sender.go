package main

import (
	"sync"
)

type SenderImpl interface {
	Start(pickNotify func() <-chan Notification, errCh chan error) (stop func() error)
}

type Sender struct {
	sender    SenderImpl
	c         chan Notification
	stopFunc  func() error
	isStarted bool
	lock      *sync.Mutex
}

func NewSender(sender SenderImpl) Sender {
	return Sender{
		sender:    sender,
		c:         make(chan Notification),
		isStarted: false,
		lock:      &sync.Mutex{},
		stopFunc:  func() error { return nil },
	}
}

func (s Sender) GetChan() chan Notification {
	return s.c
}

func (s *Sender) Start(errCh chan error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.isStarted {
		return
	}

	s.isStarted = true
	s.stopFunc = s.sender.Start(func() <-chan Notification { return s.c }, errCh)
	return
}

func (s *Sender) Stop() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isStarted {
		return nil
	}

	s.isStarted = false
	return s.stopFunc()
}

func (s *Sender) Shutdown() error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isStarted {
		return nil
	}

	s.isStarted = false
	close(s.c)
	return s.stopFunc()
}

type dummySenderImpl struct {
	id string
}

func (d dummySenderImpl) Start(pickNotify func() <-chan Notification, errCh chan error) func() error {
	var stopChan = make(chan struct{})
	go func() {
		for {
			select {
			case n := <-pickNotify():
				Logger.Info("Notify send from dummySender", "id", d.id, "message", n.Message)
			case <-stopChan:
				return
			}
		}
	}()

	return func() error {
		stopChan <- struct{}{}
		close(stopChan)
		return nil
	}
}
