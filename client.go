// Copyright 2016 Yeung Shu Hung and The Go Authors.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements the web server side for FastCGI
// as specified in http://www.mit.edu/~yandros/doc/specs/fcgi-spec.html

// A part of this file is from golang package net/http/cgi,
// in particular https://golang.org/src/net/http/cgi/host.go

package gofast

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
)

// Request hold information of a standard
// FastCGI request
type Request struct {
	Raw      *http.Request
	ID       uint16
	Params   map[string]string
	Stdin    io.ReadCloser
	KeepConn bool
}

// client is the default implementation of Client
type client struct {
	conn   *conn
	chanID chan uint16
}

// AllocID implements Client.AllocID
func (c *client) AllocID() (reqID uint16) {
	reqID = <-c.chanID
	return
}

// ReleaseID implements Client.ReleaseID
func (c *client) ReleaseID(reqID uint16) {
	go func() {
		// release the ID back to channel for reuse
		// use goroutine to prevent blocking ReleaseID
		c.chanID <- reqID
	}()
}

// writeRequest writes params and stdin to the FastCGI application
func (c *client) writeRequest(resp *ResponsePipe, req *Request) (err error) {

	// FIXME: add other role implementation, add role field to Request
	err = c.conn.writeBeginRequest(req.ID, uint16(roleResponder), 0)
	if err != nil {
		resp.Close()
		return
	}
	err = c.conn.writePairs(typeParams, req.ID, req.Params)
	if err != nil {
		resp.Close()
		return
	}
	if req.Stdin == nil {
		err = c.conn.writeRecord(typeStdin, req.ID, []byte{})
	} else {
		defer req.Stdin.Close()
		p := make([]byte, 1024)
		var count int
		for {
			count, err = req.Stdin.Read(p)
			if err == io.EOF {
				err = nil
			} else if err != nil {
				break
			}
			if count == 0 {
				break
			}

			err = c.conn.writeRecord(typeStdin, req.ID, p[:count])
			if err != nil {
				break
			}
		}
	}

	if err != nil {
		resp.Close()
	}
	return
}

// readResponse read the FastCGI stdout and stderr, then write
// to the response pipe
func (c *client) readResponse(ctx context.Context, resp *ResponsePipe, req *Request) (err error) {

	var rec record
	readError := make(chan error)

	defer c.ReleaseID(req.ID)
	defer resp.Close()

	// readloop in goroutine
	go func() {
	readLoop:
		for {
			if err := rec.read(c.conn.rwc); err != nil {
				break
			}

			// different output type for different stream
			switch rec.h.Type {
			case typeStdout:
				resp.stdOutWriter.Write(rec.content())
			case typeStderr:
				resp.stdErrWriter.Write(rec.content())
			case typeEndRequest:
				break readLoop
			default:
				readError <- fmt.Errorf("unexpected type %#v in readLoop", rec.h.Type)
			}
		}
		close(readError)
	}()

	select {
	case <-ctx.Done():
		err = fmt.Errorf("timeout or canceled by context")
	case err = <-readError:
		// do nothing and return the error
	}
	return
}

// Do implements Client.Do
func (c *client) Do(req *Request) (resp *ResponsePipe, err error) {

	resp = NewResponsePipe()
	readError, writeError := make(chan error), make(chan error)

	// check if connection exists
	if c.conn == nil {
		err = fmt.Errorf("client connection has been closed")
		return
	}

	// if there is a raw request, use the context deadline
	var ctx context.Context
	if req.Raw != nil {
		ctx = req.Raw.Context()
	} else {
		ctx = context.TODO()
	}

	// Run read and write in parallel.
	// Note: Specification never said "write before read".
	go func() {
		readError <- c.writeRequest(resp, req)
		close(readError)
	}()

	// get response in a goroutine and send to response pipe
	go func() {
		writeError <- c.readResponse(ctx, resp, req)
		close(writeError)
	}()

	// wait until context deadline
	// or until writeError is not blocked.
	select {
	case <-ctx.Done():
		err = fmt.Errorf("timeout or canceled by context")
	case err = <-readError:
		// do nothing and return the error
	case err = <-writeError:
		// do nothing and return the error
	}
	return
}

// NewRequest implements Client.NewRequest
func (c *client) NewRequest(r *http.Request) (req *Request) {
	req = &Request{
		Raw:    r,
		ID:     c.AllocID(),
		Params: make(map[string]string),
	}

	// if no http request, return here
	if r == nil {
		return
	}

	// pass body (io.ReadCloser) to stdio
	req.Stdin = r.Body

	return
}

// Close implements Client.Close
// If the inner connection has been closed before,
// this method would do nothing and return nil
func (c *client) Close() (err error) {
	if c.conn == nil {
		return
	}
	err = c.conn.Close()
	c.conn = nil
	return
}

// Client is a client interface of FastCGI
// application process through given
// connection (net.Conn)
type Client interface {

	// Do takes care of a proper FastCGI request
	Do(req *Request) (resp *ResponsePipe, err error)

	// NewRequest returns a standard FastCGI request
	// with a unique request ID allocted by the client
	NewRequest(*http.Request) *Request

	// AllocID allocates a new reqID.
	// It blocks if all possible uint16 IDs are allocated.
	AllocID() uint16

	// ReleaseID releases a reqID.
	// It never blocks.
	ReleaseID(uint16)

	// Close the underlying connection
	Close() error
}

// ConnFactory creates new network connections
// to the FPM application
type ConnFactory func() (net.Conn, error)

// SimpleConnFactory creates the simplest ConnFactory implementation.
func SimpleConnFactory(network, address string) ConnFactory {
	return func() (net.Conn, error) {
		return net.Dial(network, address)
	}
}

// ClientFactory creates new FPM client with proper connection
// to the FPM application.
type ClientFactory func() (Client, error)

// SimpleClientFactory returns a ClientFactory implementation
// with the given ConnFactory.
//
// limit is the maximum number of request that the
// applcation support. 0 means the maximum number
// available for 16bit request id (65536).
// Default 0.
//
func SimpleClientFactory(connFactory ConnFactory, limit uint32) ClientFactory {
	return func() (c Client, err error) {
		// connect to given network address
		conn, err := connFactory()
		if err != nil {
			return
		}

		// sanatize limit
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
		requestID := make(chan uint16)
		go func(maxID uint16) {
			log.Printf("run requestID loop")
			for i := uint16(0); i < maxID; i++ {
				requestID <- i
			}
			requestID <- uint16(maxID)
		}(uint16(limit - 1))

		// create client
		c = &client{
			conn:   newConn(conn),
			chanID: requestID,
		}
		return
	}
}

// NewResponsePipe returns an initialized new ResponsePipe struct
func NewResponsePipe() (p *ResponsePipe) {
	p = new(ResponsePipe)
	p.stdOutReader, p.stdOutWriter = io.Pipe()
	p.stdErrReader, p.stdErrWriter = io.Pipe()
	return
}

// ResponsePipe contains readers and writers that handles
// all FastCGI output streams
type ResponsePipe struct {
	stdOutReader io.Reader
	stdOutWriter io.WriteCloser
	stdErrReader io.Reader
	stdErrWriter io.WriteCloser
}

// Close close all writers
func (pipes *ResponsePipe) Close() {
	pipes.stdOutWriter.Close()
	pipes.stdErrWriter.Close()
}

// WriteTo writes the given output into http.ResponseWriter
func (pipes *ResponsePipe) WriteTo(rw http.ResponseWriter, ew io.Writer) (err error) {
	chErr := make(chan error, 2)

	go func() {
		chErr <- pipes.writeResponse(rw)
	}()
	go func() {
		chErr <- pipes.writeError(ew)
	}()

	for i := 0; i < 2; i++ {
		if err = <-chErr; err != nil {
			close(chErr)
			return
		}
	}
	return
}

func (pipes *ResponsePipe) writeError(w io.Writer) (err error) {
	_, err = io.Copy(w, pipes.stdErrReader)
	if err != nil {
		err = fmt.Errorf("gofast: copy error: %v", err.Error())
	}
	return
}

// writeTo writes the given output into http.ResponseWriter
func (pipes *ResponsePipe) writeResponse(w http.ResponseWriter) (err error) {
	linebody := bufio.NewReaderSize(pipes.stdOutReader, 1024)
	headers := make(http.Header)
	statusCode := 0
	headerLines := 0
	sawBlankLine := false

	for {
		var line []byte
		var isPrefix bool
		line, isPrefix, err = linebody.ReadLine()
		if isPrefix {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("gofast: long header line from subprocess")
			return
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			err = fmt.Errorf("gofast: error reading headers: %v", err)
			return
		}
		if len(line) == 0 {
			sawBlankLine = true
			break
		}
		headerLines++
		parts := strings.SplitN(string(line), ":", 2)
		if len(parts) < 2 {
			err = fmt.Errorf("gofast: bogus header line: %s", string(line))
			return
		}
		header, val := parts[0], parts[1]
		header = strings.TrimSpace(header)
		val = strings.TrimSpace(val)
		switch {
		case header == "Status":
			if len(val) < 3 {
				err = fmt.Errorf("gofast: bogus status (short): %q", val)
				return
			}
			var code int
			code, err = strconv.Atoi(val[0:3])
			if err != nil {
				err = fmt.Errorf("gofast: bogus status: %q\nline was %q",
					val, line)
				return
			}
			statusCode = code
		default:
			headers.Add(header, val)
		}
	}
	if headerLines == 0 || !sawBlankLine {
		w.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("gofast: no headers")
		return
	}

	if loc := headers.Get("Location"); loc != "" {
		/*
			if strings.HasPrefix(loc, "/") && h.PathLocationHandler != nil {
				h.handleInternalRedirect(rw, req, loc)
				return
			}
		*/
		if statusCode == 0 {
			statusCode = http.StatusFound
		}
	}

	if statusCode == 0 && headers.Get("Content-Type") == "" {
		w.WriteHeader(http.StatusInternalServerError)
		err = fmt.Errorf("gofast: missing required Content-Type in headers")
		return
	}

	if statusCode == 0 {
		statusCode = http.StatusOK
	}

	// Copy headers to rw's headers, after we've decided not to
	// go into handleInternalRedirect, which won't want its rw
	// headers to have been touched.
	for k, vv := range headers {
		for _, v := range vv {
			w.Header().Add(k, v)
		}
	}

	w.WriteHeader(statusCode)

	_, err = io.Copy(w, linebody)
	if err != nil {
		err = fmt.Errorf("gofast: copy error: %v", err)
	}
	return
}
