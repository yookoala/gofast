package gofast

import (
	"testing"
)

// requestId is supposed to be unique among all active requests in a connection. So a requestId
// should not be reused until the previous request of the same id is inactive (releasing the id).
func TestIdPool(t *testing.T) {
	pool := newClient()

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
