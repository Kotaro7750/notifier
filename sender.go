package main

import (
	"context"
	"encoding/json"
	"fmt"
	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/rs/cors"
	"net/http"
	"sync"
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

var subscriptionMap = sync.Map{}

type webPushSenderImpl struct {
	id                string
	listenAddress     string
	defaultSubscriber string
}

func (wpsi *webPushSenderImpl) GetId() string {
	return fmt.Sprintf("webPushSender %s", wpsi.id)
}

func (wpsi *webPushSenderImpl) Start(inputCh <-chan Notification, done <-chan struct{}) <-chan error {
	retCh := make(chan error)

	vapidPrivateKey, vapidPublicKey, _ := webpush.GenerateVAPIDKeys()

	serveMux := http.NewServeMux()

	serveMux.HandleFunc("GET /publickey", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.Write([]byte(vapidPublicKey))
	})

	serveMux.HandleFunc("POST /subscriptions", func(w http.ResponseWriter, r *http.Request) {
		var subscription webpush.Subscription
		err := json.NewDecoder(r.Body).Decode(&subscription)
		if err != nil {
			Logger.Error("Decoding posted subscription to JSON failed", "id", wpsi.GetId(), "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		subscriptionMap.Store(subscription.Endpoint, subscription)

		Logger.Info("Receive subscription", "id", wpsi.GetId())
		w.WriteHeader(http.StatusOK)
	})

	serveMux.HandleFunc("DELETE /subscriptions", func(w http.ResponseWriter, r *http.Request) {
		var subscription webpush.Subscription
		err := json.NewDecoder(r.Body).Decode(&subscription)
		if err != nil {
			Logger.Error("Decoding passed subscription to JSON failed", "id", wpsi.GetId(), "err", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_, ok := subscriptionMap.Load(subscription.Endpoint)
		if ok {
			Logger.Info("Delete subscription", "id", wpsi.GetId())
			subscriptionMap.Delete(subscription.Endpoint)
			w.WriteHeader(http.StatusNoContent)
		} else {
			Logger.Info("Subscription not found. Additional operation is not needed", "id", wpsi.GetId())
			w.WriteHeader(http.StatusNotFound)
		}
	})

	serveMux.HandleFunc("GET /subscriptions", func(w http.ResponseWriter, r *http.Request) {
		endpoints := make([]string, 0)

		subscriptionMap.Range(func(key, value interface{}) bool {
			endpoints = append(endpoints, key.(string))
			return true
		})

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(endpoints)
	})

	s := &http.Server{
		Addr: wpsi.listenAddress,
		// TODO セキュリティ的によくないので環境変数経由で指定できるように設定する
		Handler: cors.AllowAll().Handler(serveMux),
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
		for {
			select {
			case n, ok := <-inputCh:
				if !ok {
					Logger.Info("inputCh closed", "id", wpsi.id)
				} else {
					subscriptionMap.Range(func(key, value interface{}) bool {
						subscription := value.(webpush.Subscription)

						res, err := webpush.SendNotification([]byte(n.Message), &subscription, &webpush.Options{
							Subscriber:      wpsi.defaultSubscriber,
							VAPIDPublicKey:  vapidPublicKey,
							VAPIDPrivateKey: vapidPrivateKey,
						})

						Logger.Info("Notify send to WebPush Endpoint from webPushSender", "id", wpsi.id, "response", res.Status)

						if err != nil {
							Logger.Error("SendNotification failed", "id", wpsi.GetId(), "err", err)
							errCh <- err
							return false
						}

						return true
					})
				}
			case <-done:
				return
			}
		}
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
