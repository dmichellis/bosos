package main

import (
	"net"
	"net/http"

	. "github.com/dmichellis/gocassos/logging"
)

var current_listener net.Listener

func StopListener() {
	current_listener.Close()
	FYI.Printf("The Bosos listener has been closed. I haz a sad!")
}

func SetupListener() {
	/*
		socket, err_s := net.ResolveTCPAddr("tcp", cfg.Listen)
		if err_s != nil {
			FUUU.Fatalf("Failed to resolve Listen address! %s", err_s)
		}
	*/

	var err error
	current_listener, err = net.Listen("tcp", cfg.Listen)
	if err != nil {
		FUUU.Fatalf("Failed to listen on socket! %s", err)
	}

	FYI.Printf("You are now listening to the Bosos radio! I mean, listening to %s :P", cfg.Listen)

}
func StartListener() {
	go http.Serve(current_listener, nil)
}
