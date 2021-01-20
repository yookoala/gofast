package gofast

import (
	"math/rand"
	"testing"
	"time"
)

func TestNewStaticIdPool(t *testing.T) {
	pool := NewStaticIdPool()

	id := pool.Alloc()
	if id != 1 {
		t.Errorf("expected: 1, got %d", id)
	}

	id = pool.Alloc()
	if id != 1 {
		t.Errorf("expected: 1, got %d", id)
	}

	pool.Release(1)
	pool.Release(2)
	pool.Release(3)

	err := pool.Close()
	if err != nil {
		t.Errorf("could not close pool: %s", err)
	}
}

func TestNewDynamicIdPool(t *testing.T) {
	pool := NewDynamicIdPool(4)

	id := pool.Alloc()
	if id != 0 {
		t.Errorf("expected: 0, got %d", id)
	}

	id = pool.Alloc()
	if id != 1 {
		t.Errorf("expected: 1, got %d", id)
	}

	id = pool.Alloc()
	if id != 2 {
		t.Errorf("expected: 2, got %d", id)
	}

	id = pool.Alloc()
	if id != 3 {
		t.Errorf("expected: 3, got %d", id)
	}

	pool.Release(2)

	id = pool.Alloc()
	if id != 2 {
		t.Errorf("expected: 2, got %d", id)
	}

	err := pool.Close()
	if err != nil {
		t.Errorf("could not close pool: %s", err)
	}
}

func TestDynamicIdPool_Alloc(t *testing.T) {
	t.Logf("default limit: %d", 65535)
	ids := NewDynamicIdPool(0)
	for i := uint16(0); i < 65535; i++ {
		if want, have := i, ids.Alloc(); want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	}

	// test if new id can be allocated
	// when all ids are already allocated
	newAlloc := make(chan uint16)
	go func(ids IdPool, newAlloc chan<- uint16) {
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
	go func(ids IdPool, released uint16) {
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

func TestDynamicIdPool_Alloc_withLimit(t *testing.T) {
	limit := uint16(rand.Int31n(100) + 10)
	t.Logf("random limit: %d", limit)

	ids := NewDynamicIdPool(limit)
	for i := uint16(0); i < limit; i++ {
		if want, have := i, ids.Alloc(); want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	}

	// test if new id can be allocated
	// when all ids are already allocated
	newAlloc := make(chan uint16)
	go func(ids IdPool, newAlloc chan<- uint16) {
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
	go func(ids IdPool, released uint16) {
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
