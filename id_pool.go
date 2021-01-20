package gofast

import (
	"sync"
)

// IdPool handles allocation and releasing of ids
type IdPool interface {
	// Alloc allocates a new id from the pool
	Alloc() uint16

	// Release releases an id back to the pool
	Release(id uint16)

	// Close closes the pool
	Close() error
}

type staticIdPool struct {
}

// NewStaticIdPool creates a static id pool
func NewStaticIdPool() IdPool {
	return &staticIdPool{}
}

// Alloc implements IdPool.Alloc
func (p *staticIdPool) Alloc() uint16 {
	return 1
}

// Release implements IdPool.Release
func (p *staticIdPool) Release(id uint16) {
	// Noop
}

// Close implements IdPool.Close
func (p *staticIdPool) Close() error {
	return nil
}


type dynamicIdPool struct {
	IDs chan uint16
	done chan bool
	wg sync.WaitGroup
}

func NewDynamicIdPool(limit uint16) IdPool {
	// Sanitize limit
	if limit == 0 {
		limit = 65535
	}

	pool := &dynamicIdPool{
		IDs: make(chan uint16),
		done: make(chan bool),
	}

	go pool.sequencer(limit)

	return pool
}

func (p *dynamicIdPool) sequencer(limit uint16) {
	p.wg.Add(1)

	defer p.wg.Done()

	// pool requestID for the client
	//
	// requestID: Identifies the FastCGI request to which the record belongs.
	// The Web server re-uses FastCGI request IDs; the application
	// keeps track of the current state of each request ID on a given
	// transport connection.
	//
	// Ref: https://fast-cgi.github.io/spec#33-records
	for i := uint16(0); i < limit; i++ {
		select {
			case <-p.done:
				return
			case p.IDs <- i:
		}
	}
}

// Alloc implements IdPool.Alloc
func (p *dynamicIdPool) Alloc() uint16 {
	return <-p.IDs
}

// Release implements IdPool.Release
func (p *dynamicIdPool) Release(id uint16) {
	go func() {
		// Recover when IDs channel is closed
		defer func() {
			recover()
		}()

		p.wg.Add(1)

		defer p.wg.Done()

		select {
		case <-p.done:
		case p.IDs <- id:
		}
	}()
}

// Close implements IdPool.Close
func (p *dynamicIdPool) Close() error {
	close(p.done)

	p.wg.Wait()

	close(p.IDs)

	return nil
}
