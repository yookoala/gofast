package gofast_test

import (
	"math/rand"
	"testing"
	"time"

	"github.com/yookoala/gofast"
)

func TestClient_NewRequest(t *testing.T) {
	c := gofast.NewClient(nil)

	for i := uint32(0); i <= 65535; i++ {
		r := c.NewRequest()
		if want, have := uint16(i), r.ID; want != have {
			t.Errorf("expected %d, got %d", want, have)
		}
	}

	// test if client can allocate new request
	// when all request ids are already allocated
	newAlloc := make(chan uint16)
	go func(c gofast.Client, newAlloc chan<- uint16) {
		r := c.NewRequest() // should be blocked before releaseID call
		newAlloc <- r.ID
	}(c, newAlloc)

	select {
	case reqID := <-newAlloc:
		t.Errorf("unexpected new allocation: %d", reqID)
	case <-time.After(time.Millisecond * 100):
		t.Log("blocks as expected")
	}

	// now, release a random ID
	released := uint16(rand.Int31n(65535))
	go func(c gofast.Client, released uint16) {
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
