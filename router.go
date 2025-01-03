package main

import (
	"sync"
)

type Router struct {
	senders []Sender
}

func (r Router) Route(n Notification) {
	var wg sync.WaitGroup

	for _, sender := range r.senders {
		wg.Add(1)
		go func() {
			c := sender.GetChan()

			c <- n
			wg.Done()
		}()
	}

	wg.Wait()
}
