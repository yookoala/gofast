package gofast

import (
	"time"
)

// PoolClient wraps a client and alter the Close
// method for pool return / destroy.
type PoolClient struct {
	Client
	Err          error
	returnClient chan<- *PoolClient
	expires      time.Time
}

// Expired check if the client expired
func (pc *PoolClient) Expired() bool {
	return time.Now().After(pc.expires)
}

// Close close the inner client only
// if it is expired. Otherwise it will
// return itself to the pool.
func (pc *PoolClient) Close() error {
	if pc.Expired() {
		return pc.Client.Close()
	}
	go func() {
		// block wait until the client
		// is returned.
		pc.returnClient <- pc
	}()
	return nil
}

// NewClientPool creates a *ClientPool
// from the given ClientFactory and pool
// it to scale with expiration.
func NewClientPool(
	clientFactory ClientFactory,
	scale uint,
	expires time.Duration,
) *ClientPool {
	pool := make(chan *PoolClient, scale)
	go func() {
		for {
			c, err := clientFactory()
			pc := &PoolClient{
				Client:       c,
				Err:          err,
				returnClient: pool,
				expires:      time.Now().Add(expires),
			}
			pool <- pc
		}
	}()
	return &ClientPool{
		createClient: pool,
	}
}

// ClientPool pools client created from
// a given ClientFactory.
type ClientPool struct {
	createClient <-chan *PoolClient
}

// CreateClient implements ClientFactory
func (p *ClientPool) CreateClient() (c Client, err error) {
	pc := <-p.createClient
	if c, err = pc, pc.Err; err != nil {
		return nil, err
	}
	return
}
