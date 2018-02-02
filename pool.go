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
