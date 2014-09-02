package main

import (
	"time"

	. "github.com/dmichellis/gocassos/logging"
	"github.com/gocql/gocql"
)

func (c *ParsedConfig) CassandraClose() {
	NVM.Printf("CASSANDRA: Closing connections")
	c.backend.Conn.Close()
}

func (c *ParsedConfig) CassandraConnect() error {
	FYI.Printf("CASSANDRA: Connecting to seeds %s", c.Seed_list)

	cluster := gocql.NewCluster(c.Seed_list...)
	cluster.Keyspace = c.Keyspace
	cluster.Consistency = gocql.One
	cluster.NumConns = c.Conns_per_node
	cluster.RetryPolicy.NumRetries = c.Retries
	cluster.Timeout = time.Duration(c.Cassandra_timeout) * time.Second
	cluster.Discovery = gocql.DiscoveryConfig{
		DcFilter: c.Preferdc,
		Sleep:    c.cassandra_discovery_time,
	}
	cluster.DiscoverHosts = c.Cassandra_auto_discovery

	var err error
	if c.backend.Conn, err = cluster.CreateSession(); err != nil {
		FUUU.Printf("CASSANDRA: Error connecting to seed: %s", err)
		return err
	}

	return nil
}
