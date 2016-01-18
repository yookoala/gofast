package gofast

// Request hold information of a standard
// FastCGI request
type Request struct {
	ID       uint16
	Params   map[string]string
	Content  []byte
	KeepConn bool
}

// GetID implements Request interface
func (r *Request) GetID() uint16 {
	return r.ID
}
