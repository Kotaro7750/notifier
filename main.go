package main

import (
	"fmt"
	"time"
)

type Notification struct {
	Message string `json:"message"`
}

type Sender interface {
	GetChan() chan Notification
	Start(errCh chan error) (stop func() error)
}

type Receiver interface {
	GetChan() chan Notification
	Start(errCh chan error) (stop func() error)
}

func main() {
	sender1 := dummySender{c: make(chan Notification), id: "1"}
	sender2 := dummySender{c: make(chan Notification), id: "2"}

	senders := []Sender{sender1, sender2}

	// senderの起動
	for _, sender := range senders {
		go func() {
			for {
				errCh := make(chan error)
				stop := sender.Start(errCh)

			SELECT_LOOP:
				for {
					select {
					case err := <-errCh:
						fmt.Println(err)
						if err = stop(); err != nil {
							fmt.Println(err)
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

	dr1 := dummyReceiver{c: make(chan Notification), id: "1"}
	httpr1 := HTTPReceiver{c: make(chan Notification), id: "2"}

	receivers := []Receiver{dr1, httpr1}

	// receiverの起動
	for _, receiver := range receivers {
		go func() {
			for {
				errCh := make(chan error)
				stop := receiver.Start(errCh)
				c := receiver.GetChan()

			SELECT_LOOP:
				for {
					select {
					case n := <-c:
						routerCh <- n

					case err := <-errCh:
						fmt.Println(err)
						if err = stop(); err != nil {
							fmt.Println(err)
						}

						time.Sleep(1 * time.Second)
						break SELECT_LOOP
					}
				}
			}
		}()
	}

	// routerの起動
	for {
		select {
		case n := <-routerCh:
			router.Route(n)
		}
	}
}
