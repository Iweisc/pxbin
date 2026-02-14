package logging

import (
	"context"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/sertdev/pxbin/internal/store"
)

type LogEntry struct {
	KeyID              uuid.UUID
	Timestamp          time.Time
	Method             string
	Path               string
	Model              string
	InputFormat        string // "anthropic" or "openai"
	UpstreamID         *uuid.UUID
	StatusCode         int
	LatencyMS          int
	InputTokens        int
	OutputTokens       int
	CacheCreationTokens int
	CacheReadTokens    int
	Cost               float64
	ErrorMessage       string
	RequestMetadata    map[string]interface{}
}

type AsyncLogger struct {
	ch      chan *LogEntry
	store   *store.Store
	wg      sync.WaitGroup
	done    chan struct{}
	dropped int64 // atomic counter
}

func NewAsyncLogger(s *store.Store, bufferSize int) *AsyncLogger {
	if bufferSize <= 0 {
		bufferSize = 10000
	}
	al := &AsyncLogger{
		ch:    make(chan *LogEntry, bufferSize),
		store: s,
		done:  make(chan struct{}),
	}
	al.wg.Add(1)
	go al.worker()
	return al
}

func (al *AsyncLogger) Log(entry *LogEntry) {
	select {
	case al.ch <- entry:
	default:
		// Channel full, drop entry
		atomic.AddInt64(&al.dropped, 1)
	}
}

func (al *AsyncLogger) Dropped() int64 {
	return atomic.LoadInt64(&al.dropped)
}

func (al *AsyncLogger) Close() {
	close(al.done)
	al.wg.Wait()
}

// worker reads from channel, batches entries, and inserts them.
func (al *AsyncLogger) worker() {
	defer al.wg.Done()

	batch := make([]*store.LogEntry, 0, 100)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	flush := func() {
		if len(batch) == 0 {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := al.store.InsertLogBatch(ctx, batch); err != nil {
			log.Printf("async logger: batch insert failed: %v", err)
		}
		batch = batch[:0]
	}

	for {
		select {
		case entry := <-al.ch:
			batch = append(batch, convertToStoreEntry(entry))
			if len(batch) >= 100 {
				flush()
			}
		case <-ticker.C:
			flush()
		case <-al.done:
			// Drain remaining
			for {
				select {
				case entry := <-al.ch:
					batch = append(batch, convertToStoreEntry(entry))
				default:
					flush()
					return
				}
			}
		}
	}
}

func convertToStoreEntry(e *LogEntry) *store.LogEntry {
	return &store.LogEntry{
		KeyID:              e.KeyID,
		Timestamp:          e.Timestamp,
		Method:             e.Method,
		Path:               e.Path,
		Model:              e.Model,
		InputFormat:        e.InputFormat,
		UpstreamID:         e.UpstreamID,
		StatusCode:         e.StatusCode,
		LatencyMS:          e.LatencyMS,
		InputTokens:        e.InputTokens,
		OutputTokens:       e.OutputTokens,
		CacheCreationTokens: e.CacheCreationTokens,
		CacheReadTokens:    e.CacheReadTokens,
		Cost:               e.Cost,
		ErrorMessage:       e.ErrorMessage,
		RequestMetadata:    e.RequestMetadata,
	}
}
