package main

type Notification struct {
	Message string `json:"message"`
}

type Sender interface {
	GetChan() chan Notification
	Start() error
}

type Receiver interface {
	GetChan() chan Notification
	Start() error
}

func main() {
	sender1 := dummySender{c: make(chan Notification), id: "1"}
	sender2 := dummySender{c: make(chan Notification), id: "2"}

	go sender1.Start()
	go sender2.Start()

	router := Router{senders: []Sender{sender1, sender2}}
	routerCh := make(chan Notification)

	dr1 := dummyReceiver{c: make(chan Notification), id: "1"}
	httpr1 := HTTPReceiver{c: make(chan Notification), id: "2"}

	receivers := []Receiver{dr1, httpr1}

	// receiverの起動
	for _, receiver := range receivers {
		go func() {
			go receiver.Start()
			c := receiver.GetChan()

			for {
				select {
				case n := <-c:
					routerCh <- n
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
