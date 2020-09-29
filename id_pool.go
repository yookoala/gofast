package gofast

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
}

func NewDynamicIdPool(limit uint32) IdPool {
	// Sanitize limit
	if limit == 0 || limit > 65536 {
		limit = 65536
	}

	// pool requestID for the client
	//
	// requestID: Identifies the FastCGI request to which the record belongs.
	// The Web server re-uses FastCGI request IDs; the application
	// keeps track of the current state of each request ID on a given
	// transport connection.
	//
	// Ref: https://fast-cgi.github.io/spec#33-records
	ids := make(chan uint16)
	go func(maxID uint16) {
		// Recover when IDs channel is closed
		defer func() {
			recover()
		}()

		for i := uint16(0); i < maxID; i++ {
			ids <- i
		}
		ids <- uint16(maxID)
	}(uint16(limit - 1))

	return &dynamicIdPool{IDs: ids}
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

		// release the ID back to channel for reuse
		// use goroutine to prev0, ent blocking ReleaseID
		p.IDs <- id
	}()
}

// Close implements IdPool.Close
func (p *dynamicIdPool) Close() error {
	if p.IDs != nil {
		close(p.IDs)
		p.IDs = nil
	}

	return nil
}
