package main

import (
	"log"

	"os"
	"time"

	"github.com/dmichellis/cronologo"
	. "github.com/dmichellis/gocassos/logging"
	"github.com/rcrowley/goagain"
)

var LogRotator cronologo.Rotator

var SystemLogR = cronologo.LogFile{
	NamePrefix: "",
	TimeFormat: "2006-01-02",
	Symlink:    true,
	CallBack:   func(f *os.File) { log.SetOutput(f); os.Stdout = f; os.Stderr = f },
}

func init() {
	LogRotator.Start(1 * time.Minute)

	if BuildDate == "" {
		BuildDate = "UNKNOWN"
	}

	if GitHash == "" {
		GitHash = "UNKNOWN"
	}
}

var cfg *ParsedConfig

func main() {
	var err error
	if *test_only {
		SetLogLevel(FUUU.Level())
	} else {
		SetLogLevel(FYI.Level())
	}
	FYI.Printf("BoSOS System Info: BuildDate %s GitHash %s", BuildDate, GitHash)

	if cfg, err = ParseConfig(); err != nil {
		FUUU.Fatalf("CONFIG: Could not parse configuration: %s", err)
		if *test_only {
			os.Exit(1)
		}
	}
	if !*test_only {
		cfg.SetupLoggers()
	}

	if !*test_only {
		SetLogLevel(cfg.Log_level)
	}
	// anonymous func as cfg might change value on a reload
	defer func() { cfg.ExitHandler() }()

	if err = cfg.CassandraConnect(); err != nil {
		FUUU.Printf("CASSANDRA: Failed to connect to any seeds (%s) - exiting", err)
		os.Exit(1)
	}

	if *test_only {
		FYI.Printf("Config OK")
		os.Exit(0)
	}

	goagain.OnSIGHUP = ReloadHandler
	current_listener, err = goagain.Listener()
	if err != nil {
		SetupListener()
		StartListener()
	} else {
		StartListener()
		if err := goagain.Kill(); nil != err {
			log.Fatalln(err)
		}
	}

	if _, err := goagain.Wait(current_listener); nil != err {
		log.Fatalln(err)
	}

}
