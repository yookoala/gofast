package gofast

// Request hold information of a standard
// FastCGI request
type Request struct {
	ID       uint16
	Params   map[string]string
	Stdin    []byte
	KeepConn bool
}
