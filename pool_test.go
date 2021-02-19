package gofast

import (
	"fmt"
	"net"
	"sync/atomic"
	"testing"
	"time"
)

// mockConn is a net.Conn implementation only
// indicates if its Close method been called or not.
// If it is true, means the Close method has been called.
type mockConn bool

func (mc *mockConn) Read(b []byte) (n int, err error) {
	return 0, nil
}

func (mc *mockConn) Write(b []byte) (n int, err error) {
	return 0, nil
}

func (mc *mockConn) Close() error {
	*mc = true
	return nil
}

func (mc *mockConn) LocalAddr() net.Addr {
	return nil
}

func (mc *mockConn) RemoteAddr() net.Addr {
	return nil
}

func (mc *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (mc *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (mc *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

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

func TestClientPool_CreateClient_withErr(t *testing.T) {

	// buffered client with error
	cpHasError := NewClientPool(
		SimpleClientFactory(func() (net.Conn, error) {
			return nil, fmt.Errorf("dummy error")
		}),
		10, 1*time.Millisecond,
	)
	c, err := cpHasError.CreateClient()
	if c != nil {
		t.Errorf("expected nil, got %#v", c)
	}

	if err == nil {
		t.Errorf("expected error, got nil")
	} else if want, have := "dummy error", err.Error(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

	// unbuffered client with error
	cpHasError = NewClientPool(
		SimpleClientFactory(func() (net.Conn, error) {
			return nil, fmt.Errorf("dummy error")
		}),
		0, 1*time.Millisecond,
	)
	c, err = cpHasError.CreateClient()
	if c != nil {
		t.Errorf("expected nil, got %#v", c)
	}

	if err == nil {
		t.Errorf("expected error, got nil")
	} else if want, have := "dummy error", err.Error(); want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}

}

func TestClientPool_CreateClient_Return_0(t *testing.T) {

	var counter uint64

	// buffered client with error
	cp := NewClientPool(
		SimpleClientFactory(func() (net.Conn, error) {
			conn := mockConn(false)
			atomic.AddUint64(&counter, 1)
			return &conn, nil
		}),
		0, 1000*time.Millisecond,
	)

	// create first client
	c1, err := cp.CreateClient()
	if c1 == nil {
		t.Error("expected client, got nil")
	}
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	c1.Close()

	reused := make(chan Client)
	go func() {
		// loop until getting the supposedly returned client
		for {
			c, err := cp.CreateClient()
			if c == nil {
				t.Error("expected client, got nil")
			}
			if err != nil {
				t.Errorf("unexpected error: %s", err.Error())
			}
			if c1 == c {
				reused <- c
				break
			}
		}
	}()

	select {
	case <-reused:
		total := atomic.LoadUint64(&counter)
		t.Logf("returned client got reused with %d concurrent connections",
			total)
	case <-time.After(10 * time.Millisecond):
		t.Errorf("client is not reused")
	}
}

func TestClientPool_CreateClient_Return_40(t *testing.T) {

	var counter uint64

	// buffered client with error
	cp := NewClientPool(
		SimpleClientFactory(func() (net.Conn, error) {
			conn := mockConn(false)
			atomic.AddUint64(&counter, 1)
			return &conn, nil
		}),
		40, 1000*time.Millisecond,
	)

	// create first client
	c1, err := cp.CreateClient()
	if c1 == nil {
		t.Error("expected client, got nil")
	}
	if err != nil {
		t.Errorf("unexpected error: %s", err.Error())
	}
	c1.Close()

	reused := make(chan Client)
	go func() {
		// loop until getting the supposedly returned client
		for {
			c, err := cp.CreateClient()
			if c == nil {
				t.Error("expected client, got nil")
			}
			if err != nil {
				t.Errorf("unexpected error: %s", err.Error())
			}
			if c1 == c {
				reused <- c
				break
			}
		}
	}()

	select {
	case <-reused:
		total := atomic.LoadUint64(&counter)
		t.Logf("returned client got reused with %d concurrent connections",
			total)
	case <-time.After(10 * time.Millisecond):
		t.Errorf("client is not reused")
	}
}
