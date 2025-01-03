package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type dummyReceiver struct {
	id string
	c  chan Notification
}

func (d dummyReceiver) GetChan() chan Notification {
	return d.c
}

func (d dummyReceiver) Start(errCh chan error) func() error {
	go func() {
		for {
			d.c <- Notification{Message: fmt.Sprintf("Hello from %s", d.id)}
			time.Sleep(1 * time.Second)
		}
	}()

	return func() error {
		return nil
	}
}

type HTTPReceiver struct {
	id string
	c  chan Notification
}

func (h HTTPReceiver) GetChan() chan Notification {
	return h.c
}

func (h HTTPReceiver) Start(errCh chan error) func() error {
	serveMux := http.NewServeMux()

	serveMux.HandleFunc("POST /notifications", func(w http.ResponseWriter, r *http.Request) {
		var notification Notification
		err := json.NewDecoder(r.Body).Decode(&notification)
		if err != nil {
			http.Error(w, "Error decoding JSON", http.StatusBadRequest)
			return
		}

		h.c <- notification
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
