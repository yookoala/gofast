package gofast

import (
	"testing"
	"time"
)

func TestPoolClient_Expired(t *testing.T) {
	// client that expired
	pc := &PoolClient{
		expires: time.Now().Add(-time.Millisecond),
	}
	if want, have := true, pc.Expired(); want != have {
		t.Errorf("expected: %#v, got: %#v", want, have)
	}

	// client has not expired
	pc.expires = time.Now().Add(time.Millisecond)
	if want, have := false, pc.Expired(); want != have {
		t.Errorf("expected: %#v, got: %#v", want, have)
	}
}

func TestPoolClient_Close(t *testing.T) {
	// client that expired should be closed for real
	// while the pool gets nothing
	ch := make(chan *PoolClient)
	defer close(ch)
	pc := &PoolClient{
		Client:       &client{},
		expires:      time.Now().Add(-time.Millisecond),
		returnClient: ch,
	}
	pc.Close()
	select {
	case pcClosed := <-ch:
		t.Errorf("unexpected client from the pool: %#v", pcClosed)
	case <-time.After(10 * time.Millisecond):
		t.Logf("no getting anything from pool, as expected.")
	}

	// client has not expired should got returned
	pc.expires = time.Now().Add(time.Millisecond)
	pc.Close()
	select {
	case pcClosed := <-ch:
		if want, have := pc, pcClosed; want != have {
			t.Errorf("expected %#v, got %#v", want, have)
		}
	case <-time.After(10 * time.Millisecond):
		t.Errorf("expected to get returned client, got nothing but blocked")
	}
}
