package main

import (
	"sync"

	"github.com/Kotaro7750/notifier/abstraction"
	"github.com/Kotaro7750/notifier/notification"
)

type Router struct {
	senders []*abstraction.AutonomousChannelComponent
}

func (r Router) Route(n notification.Notification) {
	var wg sync.WaitGroup

	for _, sender := range r.senders {
		wg.Add(1)
		go func() {
			c := sender.GetChannel()

			c <- n
			wg.Done()
		}()
	}

	wg.Wait()
}
