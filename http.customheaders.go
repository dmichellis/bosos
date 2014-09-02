package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/dmichellis/gocassos"
	. "github.com/dmichellis/gocassos/logging"
)

/*
Bosos headers:

	Bosos-Transfer-Mode
	Bosos-Do-Not-Update

	Chunk-Size
	Expires-In
*/

type BososHeaders struct {
	Expiration   time.Time
	TransferMode int
	ChunkSize    int64
	DoNotUpdate  bool
}

func ParseBososHeaders(r *http.Request) (BososHeaders, error) {
	var b BososHeaders

	// safe defaults
	b.TransferMode = cfg.transfer_mode
	b.ChunkSize = cfg.backend.ChunkSize

	if exp := r.Header.Get("Expires-In"); exp != "" {
		// Try ParseDuration straight away; retry ParseDuration with an extra "s"
		seconds, err := time.ParseDuration(r.Header.Get("Expires-In"))
		if err != nil {
			seconds, err = time.ParseDuration(fmt.Sprintf("%ss", r.Header.Get("Expires-In")))
			if err != nil {
				return b, fmt.Errorf("Could not parse Expires-In header: %s", err)
			}
		}
		b.Expiration = time.Now().Add(seconds)
		if cfg.expiration_round_up.Seconds() != 0 {
			round_up := b.Expiration.Round(cfg.expiration_round_up)
			if round_up.Before(b.Expiration) {
				round_up.Add(cfg.expiration_round_up)
			}
			NVM.Printf("EXPIRATION: Rounding up (%s) expiration time from %s to %s", cfg.expiration_round_up.String(), b.Expiration.Format("2006-01-02 15:04:05 MST"), round_up.Format("2006-01-02 15:04:05 MST"))
			b.Expiration = round_up
		}
	}

	if trx := r.Header.Get("Bosos-Transfer-Mode"); trx != "" && trx != "head" {
		if value, present := gocassos.TransferModes[trx]; present {
			b.TransferMode = value
		} else {
			return b, fmt.Errorf("Could not parse Bosos-Transfer-Mode %s", trx)
		}
	}

	if chunksize := r.Header.Get("Chunk-Size"); chunksize != "" {
		if size, err := strconv.Atoi(chunksize); err != nil {
			return b, fmt.Errorf("Could not parse Chunk-Size: %s", err)
		} else {
			b.ChunkSize = int64(size)
		}
	}

	if dontupd := r.Header.Get("Bosos-Should-Not-Update"); dontupd == "true" {
		b.DoNotUpdate = true
	}

	return b, nil
}
