package main

import (
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

type Notification struct {
	Message string `json:"message"`
}

type StatefulChannel interface {
	GetChan() chan Notification
	Start(errCh chan error)
	Stop() error
	Shutdown() error
}

var Logger = slog.New(slog.NewJSONHandler(os.Stdout, nil))

func main() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, os.Interrupt, os.Kill)

	sender1 := NewSender(dummySenderImpl{id: "1"})
	sender2 := NewSender(dummySenderImpl{id: "2"})
	senders := []*Sender{&sender1, &sender2}

	dr1 := NewReceiver(dummyReceiverImpl{id: "1"})
	httpr1 := NewReceiver(HTTPReceiverImpl{id: "2"})
	receivers := []*Receiver{&dr1, &httpr1}

	statefulChannels := []StatefulChannel{&sender1, &sender2, &dr1, &httpr1}

	for _, statefulChannel := range statefulChannels {
		go func() {
			for {
				errCh := make(chan error)
				statefulChannel.Start(errCh)

			SELECT_LOOP:
				for {
					select {
					case err := <-errCh:
						Logger.Error(err.Error())
						if err = statefulChannel.Stop(); err != nil {
							Logger.Error(err.Error())
						}

						time.Sleep(1 * time.Second)
						break SELECT_LOOP
					}
				}
			}
		}()
	}

	router := Router{senders: senders}
	routerCh := make(chan Notification)

	for _, receiver := range receivers {
		go func() {
		SELECT_LOOP:
			for {
				select {
				case n, ok := <-receiver.GetChan():
					if !ok {
						Logger.Info("Channel closed")
						break SELECT_LOOP
					}
					routerCh <- n
				}
			}
		}()
	}

	go func() {
		for {
			select {
			case n := <-routerCh:
				router.Route(n)
			}
		}
	}()

	<-sigCh
	Logger.Info("Received signal")

	Logger.Info("Shutting down receivers")

	wg := sync.WaitGroup{}
	for _, receiver := range receivers {
		wg.Add(1)
		go func() {
			if err := receiver.Shutdown(); err != nil {
				Logger.Error(err.Error())
			}
			wg.Done()
		}()
	}

	wg.Wait()
	Logger.Info("All receivers are shut down")

	Logger.Info("Shutting down senders")

	wg = sync.WaitGroup{}
	for _, sender := range senders {
		wg.Add(1)
		go func() {
			if err := sender.Shutdown(); err != nil {
				Logger.Error(err.Error())
			}
			wg.Done()
		}()
	}

	wg.Wait()
	Logger.Info("All senders are shut down")

	close(routerCh)
}
