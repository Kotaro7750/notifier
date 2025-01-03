package main

import (
	"fmt"
)

type dummySender struct {
	id string
	c  chan Notification
}

func (d dummySender) GetChan() chan Notification {
	return d.c
}

func (d dummySender) Start() error {
	for {
		select {
		case n := <-d.c:
			fmt.Printf("Send in %s, message: %s\n", d.id, n.Message)
		}
	}
}
