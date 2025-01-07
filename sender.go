package main

import (
	"fmt"
	"time"
)

type SenderImpl interface {
	GetId() string
	Start(inputCh <-chan Notification, done <-chan struct{}) <-chan error
}

type Sender struct {
	impl SenderImpl
}

func (s *Sender) Start(inputCh chan Notification, done <-chan struct{}) <-chan error {
	return s.impl.Start(inputCh, done)
}

func (s *Sender) GetId() string {
	return s.impl.GetId()
}

type dummySenderImpl struct {
	id string
}

func (dsi *dummySenderImpl) GetId() string {
	return fmt.Sprintf("dummySender %s", dsi.id)
}

func (dsi *dummySenderImpl) Start(inputCh <-chan Notification, done <-chan struct{}) <-chan error {
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
					Logger.Info("inputCh closed", "id", dsi.id)
				} else {
					Logger.Info("Notify send from dummySender", "id", dsi.id, "message", n.Message)
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
