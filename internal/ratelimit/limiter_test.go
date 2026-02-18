package ratelimit

import (
	"sync"
	"testing"
)

func TestAllowBasic(t *testing.T) {
	l := NewLimiter(10, 10)
	defer l.Close()

	// Should allow up to burst requests.
	for i := 0; i < 10; i++ {
		if !l.Allow("key1") {
			t.Fatalf("request %d should be allowed", i)
		}
	}

	// Next request should be denied (tokens exhausted).
	if l.Allow("key1") {
		t.Fatal("request 11 should be denied")
	}
}

func TestAllowDifferentKeys(t *testing.T) {
	l := NewLimiter(10, 5)
	defer l.Close()

	// Exhaust key1.
	for i := 0; i < 5; i++ {
		l.Allow("key1")
	}

	// key2 should still be allowed.
	if !l.Allow("key2") {
		t.Fatal("different key should be allowed")
	}
}

func TestAllowConcurrent(t *testing.T) {
	l := NewLimiter(1000, 100)
	defer l.Close()

	var allowed, denied int64
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok := l.Allow("concurrent-key")
			mu.Lock()
			if ok {
				allowed++
			} else {
				denied++
			}
			mu.Unlock()
		}()
	}

	wg.Wait()

	// With burst=100 and 200 requests, we should have some allowed and some denied.
	if allowed == 0 {
		t.Fatal("expected some requests to be allowed")
	}
	if denied == 0 {
		t.Fatal("expected some requests to be denied")
	}
	t.Logf("allowed=%d denied=%d", allowed, denied)
}

func TestBurstBehavior(t *testing.T) {
	l := NewLimiter(1, 20)
	defer l.Close()

	// Burst of 20 should all succeed.
	for i := 0; i < 20; i++ {
		if !l.Allow("burst") {
			t.Fatalf("burst request %d should be allowed", i)
		}
	}

	// 21st should fail.
	if l.Allow("burst") {
		t.Fatal("request beyond burst should be denied")
	}
}
