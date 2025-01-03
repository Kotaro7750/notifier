package main

type dummySender struct {
	id string
	c  chan Notification
}

func (d dummySender) GetChan() chan Notification {
	return d.c
}

func (d dummySender) Start(errCh chan error) func() error {
	go func() {
		for {
			select {
			case n := <-d.c:
				Logger.Info("Notify send from dummySender", "id", d.id, "message", n.Message)
			}
		}
	}()

	return func() error {
		return nil
	}
}
