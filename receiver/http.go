package receiver

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/notification"
)

func HTTPReceiverBuilder(id string, properties map[string]interface{}) (abstraction.AbstractChannelComponent, error) {
	listenAddr, ok := properties["listenAddress"]
	if !ok {
		return nil, fmt.Errorf("listenAddress is required")
	}

	listenAddrStr, ok := listenAddr.(string)
	if !ok {
		return nil, fmt.Errorf("listenAddress should be string")
	}

	return NewReceiver(&HTTPReceiverImpl{
		id:         id,
		listenAddr: listenAddrStr,
		logger:     nil,
	}), nil
}

type HTTPReceiverImpl struct {
	id         string
	logger     *slog.Logger
	listenAddr string
}

func (hri *HTTPReceiverImpl) GetId() string {
	return fmt.Sprintf("%s", hri.id)
}

func (hri *HTTPReceiverImpl) GetLogger() *slog.Logger {
	return hri.logger
}

func (hri *HTTPReceiverImpl) SetLogger(logger *slog.Logger) {
	hri.logger = logger
}

func (hri HTTPReceiverImpl) Start(outputCh chan<- notification.Notification, done <-chan struct{}) <-chan error {
	retCh := make(chan error)

	serveMux := http.NewServeMux()

	serveMux.HandleFunc("POST /notifications", func(w http.ResponseWriter, r *http.Request) {
		var notification notification.Notification
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
