package main

import (
	. "github.com/dmichellis/gocassos/logging"
)

// Concurrency control for requests
//
// ClownCar, because you can only fit so many bozos in a Clown Car :)
type ClownCar struct {
	seats chan struct{}
	Tag   string
}

func (c *ClownCar) Enter() {
	if c == nil || c.seats == nil {
		return
	}
	NVM.Printf("Clown trying to get in '%s' ClownCar...", c.Tag)
	c.seats <- struct{}{}
	NVM.Printf("Clown got in the '%s' ClownCar!", c.Tag)
}

func (c *ClownCar) Leave() {
	if c == nil || c.seats == nil {
		return
	}
	NVM.Printf("Clown leaving the '%s' ClownCar", c.Tag)
	<-c.seats
}

func (c *ClownCar) Init(tag string, l int) {
	if l <= 0 {
		return
	}
	NVM.Printf("Putting together a '%s' ClownCar with %d seats", tag, l)
	c.seats = make(chan struct{}, l)
	c.Tag = tag
	return
}
