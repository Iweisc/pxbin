package logging

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/sertdev/pxbin/internal/store"
)

type LogCleaner struct {
	store     *store.Store
	retention time.Duration
	wg        sync.WaitGroup
	done      chan struct{}
}

func NewLogCleaner(s *store.Store, retentionDays int) *LogCleaner {
	lc := &LogCleaner{
		store: s,
		done:  make(chan struct{}),
	}
	if retentionDays <= 0 {
		return lc
	}
	lc.retention = time.Duration(retentionDays) * 24 * time.Hour
	lc.wg.Add(1)
	go lc.worker()
	return lc
}

func (lc *LogCleaner) Close() {
	close(lc.done)
	lc.wg.Wait()
}

func (lc *LogCleaner) worker() {
	defer lc.wg.Done()

	// Run once at startup, then every hour.
	lc.cleanup()

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lc.cleanup()
		case <-lc.done:
			return
		}
	}
}

func (lc *LogCleaner) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	cutoff := time.Now().Add(-lc.retention)
	deleted, err := lc.store.DeleteOldLogs(ctx, cutoff)
	if err != nil {
		log.Printf("log cleaner: failed to delete old logs: %v", err)
		return
	}
	if deleted > 0 {
		log.Printf("log cleaner: deleted %d logs older than %d days", deleted, int(lc.retention.Hours()/24))
	}
}
