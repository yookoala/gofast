package gofast

import (
	"math/rand"
	"testing"
	"time"
)

// requestId is supposed to be unique among all active requests in a connection. So a requestId
// should not be reused until the previous request of the same id is inactive (releasing the id).
func TestIDPool_Alloc(t *testing.T) {
	ids := newIDs()
	idToReserve := uint16(rand.Int31n(int32(MaxRequestID)))

	// Loop over all ids to make sure it is sequencely returning
	// 1 to 65535.
	//
	// Note: Use uint as loop counter so it can loop past 65535
	// to end the loop (also keep the code readable)
	for i := uint(1); i <= uint(MaxRequestID); i++ {
		if want, have := uint16(i), ids.Alloc(); want != have {
			t.Fatalf("expected %v, got %v", want, have)
		}
		if i != uint(idToReserve) {
			ids.Release(uint16(i))
		}
	}

	// Loop over all requestids 5 times
	for i := 0; i < 5; i++ {
		for j := uint(1); j <= uint(MaxRequestID-1); j++ {
			id := ids.Alloc()
			if id == 0 {
				t.Fatal("ids.Alloc() is never allowed to return 0")
			} else if id == idToReserve {
				t.Fatalf("The requestId %v was not reserved as expect", id)
			} else if j < uint(idToReserve) {
				if want, have := uint(id), j; want != have {
					t.Fatalf("expected %v, got %v", want, have)
				}
			} else if j >= uint(idToReserve) {
				if want, have := uint(id), j+1; want != have {
					t.Fatalf("expected %v, got %v", want, have)
				}
			}
			ids.Release(id) // always release the allocated id
		}
	}

	// release the reserved id
	ids.Release(idToReserve)

	// make sure all ids are available again
	for i := uint(1); i <= uint(MaxRequestID); i++ {
		if want, have := uint16(i), ids.Alloc(); want != have {
			t.Fatalf("expected %v, got %v", want, have)
		}
	}
}

// If all IDs are used up, pool is supposed to block on alloc after exhaustion.
func TestIDPool_block(t *testing.T) {

	ids := newIDs()

	// Test allocating all ids once.
	for i := uint(1); i <= uint(MaxRequestID); i++ {
		id := ids.Alloc()
		if want, have := i, uint(id); want != have {
			t.Errorf("expected to allocate %v, got %v", want, have)
			t.FailNow()
		}
	}

	newAlloc := make(chan uint16)
	waitAlloc := func(ids *idPool, newAlloc chan<- uint16) {
		newAlloc <- ids.Alloc()
	}
	go waitAlloc(ids, newAlloc)
	go waitAlloc(ids, newAlloc)
	go waitAlloc(ids, newAlloc)
	go waitAlloc(ids, newAlloc)
	go waitAlloc(ids, newAlloc)

	// wait some time to see if we can allocate id again
	select {
	case reqID := <-newAlloc:
		t.Fatalf("unexpected new allocation: %d", reqID)
	case <-time.After(time.Millisecond * 100):
		t.Log("blocks as expected")
	}

	// now, release a random ID
	released := uint16(rand.Int31n(int32(MaxRequestID)))
	go func(ids *idPool, released uint16) {
		// release an id
		ids.Release(released)
		t.Logf("id released: %v", released)
	}(ids, released)

	// wait some time to see if we can allocate id again
	select {
	case reqID := <-newAlloc:
		if want, have := released, reqID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	case <-time.After(time.Millisecond * 100):
		t.Errorf("unexpected blocking")
	}
}
