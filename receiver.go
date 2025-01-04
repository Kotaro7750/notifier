package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type ReceiverImpl interface {
	Start(onReceive func(n Notification), errCh chan error) (stop func() error)
}

type Receiver struct {
	receiver  ReceiverImpl
	c         chan Notification
	stopFunc  func() error
	isStarted bool
	lock      *sync.Mutex
}

func NewReceiver(receiver ReceiverImpl) Receiver {
	return Receiver{
		receiver:  receiver,
		c:         make(chan Notification),
		isStarted: false,
		lock:      &sync.Mutex{},
		stopFunc:  func() error { return nil },
	}
}

func (r Receiver) GetChan() chan Notification {
	return r.c
}

func (r *Receiver) Start(errCh chan error) {
	r.lock.Lock()
	defer r.lock.Unlock()

	if r.isStarted {
		return
	}

	r.isStarted = true
	r.stopFunc = r.receiver.Start(func(n Notification) { r.c <- n }, errCh)
	return
}

func (r *Receiver) Stop() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.isStarted {
		return nil
	}

	r.isStarted = false
	return r.stopFunc()
}

func (r *Receiver) Shutdown() error {
	r.lock.Lock()
	defer r.lock.Unlock()

	if !r.isStarted {
		return nil
	}

	r.isStarted = false
	err := r.stopFunc()
	close(r.c)
	return err
}

type dummyReceiverImpl struct {
	id string
}

func (di dummyReceiverImpl) Start(onReceive func(n Notification), errCh chan error) func() error {
	stopChan := make(chan struct{})
	go func() {
		for {
			select {
			case <-stopChan:
				return
			default:
				onReceive(Notification{Message: fmt.Sprintf("Hello from %s", di.id)})
				time.Sleep(1 * time.Second)
			}
		}
	}()

	return func() error {
		time.Sleep(5 * time.Second)
		stopChan <- struct{}{}
		close(stopChan)
		return nil
	}
}

type HTTPReceiverImpl struct {
	id string
}

func (hi HTTPReceiverImpl) Start(onReceive func(n Notification), errCh chan error) func() error {
	serveMux := http.NewServeMux()

	serveMux.HandleFunc("POST /notifications", func(w http.ResponseWriter, r *http.Request) {
		var notification Notification
		err := json.NewDecoder(r.Body).Decode(&notification)
		if err != nil {
			http.Error(w, "Error decoding JSON", http.StatusBadRequest)
			return
		}

		onReceive(notification)
	})

	s := &http.Server{
		Addr:    ":8080",
		Handler: serveMux,
	}

	go func() {
		errCh <- s.ListenAndServe()
	}()

	return func() error {
		return s.Shutdown(context.Background())
	}
}
