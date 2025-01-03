package main

import (
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

func (d dummyReceiver) Start() error {
	for {
		d.c <- Notification{Message: fmt.Sprintf("Hello from %s", d.id)}
		time.Sleep(1 * time.Second)
	}
}

type HTTPReceiver struct {
	id string
	c  chan Notification
}

func (h HTTPReceiver) GetChan() chan Notification {
	return h.c
}

func (h HTTPReceiver) Start() error {
	s := &http.Server{
		Addr:    ":8080",
		Handler: nil,
	}

	http.HandleFunc("/notifications", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
			return
		}

		var notification Notification
		err := json.NewDecoder(r.Body).Decode(&notification)
		if err != nil {
			http.Error(w, "Error decoding JSON", http.StatusBadRequest)
			return
		}

		h.c <- notification
	})

	return s.ListenAndServe()
}
