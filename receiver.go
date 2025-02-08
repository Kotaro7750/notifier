package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type ReceiverImpl interface {
	GetId() string
	Start(outputCh chan<- Notification, done <-chan struct{}) <-chan error
}

type Receiver struct {
	impl ReceiverImpl
}

func (r *Receiver) Start(outputCh chan Notification, done <-chan struct{}) <-chan error {
	return r.impl.Start(outputCh, done)
}
func (r *Receiver) GetId() string {
	return r.impl.GetId()
}

type dummyReceiverImpl struct {
	id string
}

func (dri *dummyReceiverImpl) GetId() string {
	return fmt.Sprintf("dummyReceiver %s", dri.id)
}

func (dri *dummyReceiverImpl) Start(outputCh chan<- Notification, done <-chan struct{}) <-chan error {
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
				outputCh <- Notification{Message: fmt.Sprintf("Hello from %s", dri.id)}

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

type HTTPReceiverImpl struct {
	id         string
	listenAddr string
}

func (hri *HTTPReceiverImpl) GetId() string {
	return fmt.Sprintf("HTTPReceiver %s", hri.id)
}

func (hri HTTPReceiverImpl) Start(outputCh chan<- Notification, done <-chan struct{}) <-chan error {
	retCh := make(chan error)

	serveMux := http.NewServeMux()

	serveMux.HandleFunc("POST /notifications", func(w http.ResponseWriter, r *http.Request) {
		var notification Notification
		err := json.NewDecoder(r.Body).Decode(&notification)
		if err != nil {
			http.Error(w, "Error decoding JSON", http.StatusBadRequest)
			return
		}

		outputCh <- notification
	})

	s := &http.Server{
		Addr:    hri.listenAddr,
		Handler: serveMux,
	}

	shutdownFunc := func() {
		s.Shutdown(context.Background())
	}

	errCh := make(chan error)
	go func() {
		defer close(errCh)
		err := s.ListenAndServe()
		errCh <- err
	}()

	go func() {
		defer close(retCh)
		select {
		case err := <-errCh:
			shutdownFunc()
			retCh <- err
			return
		case <-done:
			shutdownFunc()
			return
		}
	}()

	return retCh
}
