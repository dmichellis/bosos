package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/dmichellis/gocassos"
	. "github.com/dmichellis/gocassos/logging"
)

var config_file = flag.String("config", "/etc/bosos/config.json", "JSON configuration file")
var test_only = flag.Bool("test", false, "Parse, connect and test configuration options")
var pid_file = flag.String("pid", "/var/run/bosos.pid", "pid file for the running daemon")

type JsonConfig struct {
	Seed_list []string
	Username,
	Password,
	Keyspace,
	Preferdc,
	Dc string
	Conns_per_node           int
	Retries                  int
	Cassandra_timeout        int
	Cassandra_auto_discovery bool
	Cassandra_discovery_time string

	Listen   string
	Pid_file string
	Lb_file  string
	// Log_to_syslog bool
	Log_level int

	Access_log string
	System_log string

	Scrub_grace_time int
	Populate_paths   bool
	Allow_updates    bool

	Expiration_round_up string

	Write_consistency []string
	Read_consistency  []string

	Default_chunk_size int64

	Cassandra_reqs_per_get,
	Cassandra_reqs_per_put int

	Concurrent_fetch_requests int
	Concurrent_push_requests  int
	Concurrent_requests       int

	Transfer_mode string

	Inline_payload_max int
}

type ParsedConfig struct {
	JsonConfig

	fetchers,
	pushers,
	global ClownCar

	transfer_mode int

	expiration_round_up,
	cassandra_discovery_time time.Duration

	backend gocassos.ObjectStorage
}

var config_defaults = ParsedConfig{
	JsonConfig: JsonConfig{
		Username:                 "",
		Password:                 "",
		Seed_list:                []string{"127.0.0.1"},
		Keyspace:                 "bosos",
		Preferdc:                 "",
		Dc:                       "",
		Conns_per_node:           3,
		Retries:                  3,
		Cassandra_timeout:        3,
		Cassandra_auto_discovery: true,
		Cassandra_discovery_time: "0s",

		Listen:  "localhost:8091",
		Lb_file: "/etc/bosos/lb_disable",

		Access_log: "/var/log/bosos/access.log",
		System_log: "/var/log/bosos/system.log",

		Scrub_grace_time: 10,
		Populate_paths:   true,
		Allow_updates:    true,

		Expiration_round_up: "0s",

		Write_consistency:  []string{"all", "quorum", "one"},
		Read_consistency:   []string{"one", "one", "quorum"},
		Default_chunk_size: 500000,

		Cassandra_reqs_per_get: 5,
		Cassandra_reqs_per_put: 5,

		Concurrent_fetch_requests: 0,
		Concurrent_push_requests:  0,
		Concurrent_requests:       0,

		Transfer_mode: "stream",
		Log_level:     FYI.Level(),

		Inline_payload_max: 0,
	},
	fetchers: ClownCar{},
	pushers:  ClownCar{},
	global:   ClownCar{},

	transfer_mode:            gocassos.StreamMode,
	expiration_round_up:      time.Duration(0),
	cassandra_discovery_time: time.Duration(0),
	backend:                  gocassos.ObjectStorage{},
}

func init() {
	flag.Parse()
	if *test_only {
		SetLogLevel(FUUU.Level())
	}
}

func ParseConfig() (*ParsedConfig, error) {
	FYI.Printf("CONFIG: Loading config file '%s'", *config_file)

	file, err := os.Open(*config_file)
	if err != nil {
		FUUU.Printf("CONFIG: Could not open config file: %s", err)
		return nil, err
	}

	// Set defaults
	new_cfg := config_defaults
	new_cfg.backend.Init()

	decoder := json.NewDecoder(file)
	if err = decoder.Decode(&new_cfg); err != nil {
		FYI.Printf("CONFIG: Failed to parse configuration: %s", err)
		return nil, err
	}
	file.Close()

	new_cfg.backend.InlinePayloadMax = new_cfg.Inline_payload_max
	new_cfg.backend.ScrubGraceTime = new_cfg.Scrub_grace_time
	new_cfg.backend.PopulatePaths = new_cfg.Populate_paths
	new_cfg.backend.AllowUpdates = new_cfg.Allow_updates
	new_cfg.backend.ChunkSize = new_cfg.Default_chunk_size
	new_cfg.backend.ConcurrentGetsPerObj = new_cfg.Cassandra_reqs_per_get
	new_cfg.backend.ConcurrentPutsPerObj = new_cfg.Cassandra_reqs_per_put

	if new_cfg.Preferdc == "" {
		new_cfg.Preferdc = new_cfg.Dc
	}

	new_cfg.fetchers.Init("fetcher", new_cfg.Concurrent_fetch_requests)
	new_cfg.pushers.Init("pusher", new_cfg.Concurrent_push_requests)
	new_cfg.global.Init("global", new_cfg.Concurrent_requests)

	if err = new_cfg.backend.SetConsistencies(new_cfg.Read_consistency, new_cfg.Write_consistency); err != nil {
		FUUU.Printf("CONFIG: Failed to parse consistencies: %s", err)
		return nil, err
	}
	if value, present := gocassos.TransferModes[new_cfg.Transfer_mode]; present {
		new_cfg.transfer_mode = value
	} else {
		FUUU.Printf("CONFIG: Failed to parse transfer mode %s", new_cfg.Transfer_mode)
		return nil, err
	}

	if new_cfg.Expiration_round_up != "" {
		new_cfg.expiration_round_up, err = time.ParseDuration(new_cfg.Expiration_round_up)
		if err != nil {
			FUUU.Printf("CONFIG: Failed to parse expiration_round_up: %s", err)
			return nil, err
		}
	}

	if new_cfg.Cassandra_discovery_time != "" {
		new_cfg.cassandra_discovery_time, err = time.ParseDuration(new_cfg.Cassandra_discovery_time)
		if err != nil {
			FUUU.Printf("CONFIG: Failed to parse cassandra_discovery_time: %s", err)
			return nil, err
		}
	}

	if *pid_file != "" && !*test_only {
		if f, err := os.Create(*pid_file); err != nil {
			FUUU.Printf("CONFIG: Failed to write pidfile: %s", err)
		} else {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
		}
	}

	tmp_output := bytes.NewBufferString("")
	encoder := json.NewEncoder(tmp_output)
	encoder.Encode(new_cfg)
	if *test_only || NVM.Enabled() {
		FYI.Printf("CONFIG: Parsed JSON config: %s", tmp_output.String())
	}

	return &new_cfg, nil
}

func (c *ParsedConfig) SetupLoggers() {
	LogRotator.Del(&AccessLogR)
	LogRotator.Del(&SystemLogR)
	AccessLogR.NamePrefix = c.Access_log
	SystemLogR.NamePrefix = c.System_log
	if c.Access_log != "" {
		LogRotator.Add(&AccessLogR)
	}
	if c.System_log != "" {
		LogRotator.Add(&SystemLogR)
	}
}
