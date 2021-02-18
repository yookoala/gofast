package gofast

import (
	"testing"
	"time"
)

// requestId is supposed to be unique among all active requests in a connection. So a requestId
// should not be reused until the previous request of the same id is inactive (releasing the id).
func TestIdPool(t *testing.T) {
	pool := newIDs()

	// First entry should be 1
	id := pool.Alloc()
	if id != 1 {
		t.Fatal("pool.Alloc() first entry should start at 1")
	}

	reserveTest := false
	// Loop over all requestids 5 times
	for i := 0; i < int(MaxUint)*5; i++ {
		id := pool.Alloc()
		if id == 0 {
			t.Fatal("pool.Alloc() is never allowed to return 0")
		}
		if reserveTest && id == 2 {
			t.Fatal("Received requestId that is still in use")
		}

		// Preserve id=2 (to check if it is skipped in the next run)
		if id == 2 {
			reserveTest = true
		} else {
			pool.Release(id)
		}
	}
}

// If all IDs are used up, pool is supposed to block on alloc.
func TestIdPool_block(t *testing.T) {
	pool := newIDs()

	// Test allocating all ids once.
	for i := uint16(1); i < MaxUint; i++ {
		id := pool.Alloc()
		if want, have := i, id; want != have {
			t.Errorf("expected to allocate %v, got %v", want, have)
			t.FailNow()
		}
	}

	alloc := make(chan uint16)
	go func() {
		alloc <- pool.Alloc()
	}()

	// wait some time to see if we can allocate id again
	select {
	case <-time.After(time.Millisecond * 10):
		t.Logf("alloc before release timeout as expected")
	case id := <-alloc:
		t.Errorf("allocated id unexpectedly: %v", id)
	}

	go func() {
		// release an id
		pool.Release(42)
		t.Logf("id released")
	}()

	// wait some time to see if we can allocate id again
	select {
	case <-time.After(time.Millisecond * 10):
		t.Errorf("alloc after release timeout unexpectedly")
	case id := <-alloc:
		if want, have := uint16(42), id; want != have {
			t.Errorf("expected %v, got %v", want, have)
		}
	}
}
