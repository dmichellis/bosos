package main

import (
	"net"
	"os"

	. "github.com/dmichellis/gocassos/logging"
)

func ReloadHandler(l net.Listener) error {
	FYI.Printf("RELOAD: Received SIGHUP; reloading")

	new_cfg, err := ParseConfig()
	if err != nil {
		FUUU.Printf("RELOAD: Aborting reload due to invalid configuration: %s", err)
		return err
	}
	err = new_cfg.CassandraConnect()
	if err != nil {
		FUUU.Printf("RELOAD: Aborting reload due to cassandra connection failure: %s", err)
		return err
	}
	if cfg.Listen != new_cfg.Listen {
		StopListener()
		StartListener()
	}

	old_cassos := cfg
	cfg = new_cfg
	cfg.SetupLoggers()
	SetLogLevel(cfg.Log_level)
	old_cassos.Close()
	FYI.Printf("RELOAD: Operation sucessful")
	return nil
}

func (c *ParsedConfig) Close() {
	if c == nil {
		return
	}
	FYI.Printf("CLOSE: Waiting for ongoing operations to finish...")
	// TODO timeout here using scrub grace time
	c.backend.Wait()
	c.CassandraClose()
	FYI.Printf("CLOSE: All operations done!")
}

func (c *ParsedConfig) ExitHandler() {
	if c == nil {
		return
	}

	FYI.Printf("EXIT: Received the Bat Signal! (or shutdown signal)")
	c.Close()
	FYI.Printf("EXIT: I may be dying, but I WILL NEVER QUIT!")
	os.Exit(0)
}
