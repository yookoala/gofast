package gofast

import (
	"math/rand"
	"testing"
	"time"
)

func TestIDPool_Alloc(t *testing.T) {
	t.Logf("default limit: %d", 65535)
	ids := newIDs(0)
	for i := uint32(0); i <= 65535; i++ {
		if want, have := uint16(i), ids.Alloc(); want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	}

	// test if new id can be allocated
	// when all ids are already allocated
	newAlloc := make(chan uint16)
	go func(ids idPool, newAlloc chan<- uint16) {
		newAlloc <- ids.Alloc()
	}(ids, newAlloc)

	select {
	case reqID := <-newAlloc:
		t.Errorf("unexpected new allocation: %d", reqID)
	case <-time.After(time.Millisecond * 100):
		t.Log("blocks as expected")
	}

	// now, release a random ID
	released := uint16(rand.Int31n(65535))
	go func(ids idPool, released uint16) {
		ids.Release(released)
	}(ids, released)

	select {
	case reqID := <-newAlloc:
		if want, have := released, reqID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	case <-time.After(time.Millisecond * 100):
		t.Errorf("unexpected blocking")
	}
}

func TestIDPool_Alloc_withLimit(t *testing.T) {

	limit := uint32(rand.Int31n(100) + 10)
	t.Logf("random limit: %d", limit)

	ids := newIDs(limit)
	for i := uint32(0); i < limit; i++ {
		if want, have := uint16(i), ids.Alloc(); want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	}

	// test if new id can be allocated
	// when all ids are already allocated
	newAlloc := make(chan uint16)
	go func(ids idPool, newAlloc chan<- uint16) {
		newAlloc <- ids.Alloc()
	}(ids, newAlloc)

	select {
	case reqID := <-newAlloc:
		t.Errorf("unexpected new allocation: %d", reqID)
	case <-time.After(time.Millisecond * 100):
		t.Log("blocks as expected")
	}

	// now, release a random ID
	released := uint16(rand.Int31n(int32(limit)))
	go func(ids idPool, released uint16) {
		ids.Release(released)
	}(ids, released)

	select {
	case reqID := <-newAlloc:
		if want, have := released, reqID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	case <-time.After(time.Millisecond * 100):
		t.Errorf("unexpected blocking")
	}
}
