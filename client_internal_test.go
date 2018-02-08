package gofast

import (
	"math/rand"
	"net"
	"testing"
	"time"
)

func TestClient_AllocID(t *testing.T) {
	t.Logf("default limit: %d", 65535)
	c, _ := SimpleClientFactory(
		func() (net.Conn, error) { return nil, nil }, // dummy conn factory
		0,
	)()
	for i := uint32(0); i <= 65535; i++ {
		if want, have := uint16(i), c.AllocID(); want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	}

	// test if new id can be allocated
	// when all ids are already allocated
	newAlloc := make(chan uint16)
	go func(c Client, newAlloc chan<- uint16) {
		newAlloc <- c.AllocID()
	}(c, newAlloc)

	select {
	case reqID := <-newAlloc:
		t.Errorf("unexpected new allocation: %d", reqID)
	case <-time.After(time.Millisecond * 100):
		t.Log("blocks as expected")
	}

	// now, release a random ID
	released := uint16(rand.Int31n(65535))
	go func(c Client, released uint16) {
		c.ReleaseID(released)
	}(c, released)

	select {
	case reqID := <-newAlloc:
		if want, have := released, reqID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	case <-time.After(time.Millisecond * 100):
		t.Errorf("unexpected blocking")
	}
}

func TestClient_AllocID_withLimit(t *testing.T) {

	limit := uint32(rand.Int31n(100) + 10)
	t.Logf("random limit: %d", limit)

	c, _ := SimpleClientFactory(
		func() (net.Conn, error) { return nil, nil }, // dummy conn factory
		limit,
	)()
	for i := uint32(0); i < limit; i++ {
		if want, have := uint16(i), c.AllocID(); want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	}

	// test if new id can be allocated
	// when all ids are already allocated
	newAlloc := make(chan uint16)
	go func(c Client, newAlloc chan<- uint16) {
		newAlloc <- c.AllocID()
	}(c, newAlloc)

	select {
	case reqID := <-newAlloc:
		t.Errorf("unexpected new allocation: %d", reqID)
	case <-time.After(time.Millisecond * 100):
		t.Log("blocks as expected")
	}

	// now, release a random ID
	released := uint16(rand.Int31n(int32(limit)))
	go func(c Client, released uint16) {
		c.ReleaseID(released)
	}(c, released)

	select {
	case reqID := <-newAlloc:
		if want, have := released, reqID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	case <-time.After(time.Millisecond * 100):
		t.Errorf("unexpected blocking")
	}
}
