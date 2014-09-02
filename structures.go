package main

import "os"

type BososObjectStorage struct {
	Listen string
	Seed   []string

	Disabled_notification_file string

	ConcurrentGets, ConcurrentPuts int
}

//var cassos *gocassos.ObjectStorage
var system_log *os.File

//var system_tee io.Writer

var BuildDate string
var GitHash string
