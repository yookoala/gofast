// Copyright 2016 Yeung Shu Hung and The Go Authors.
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// This file implements the web server side for FastCGI
// as specified in http://www.fastcgi.com/drupal/node/22

// A part of this file is from golang package net/http/cgi,
// in particular https://golang.org/src/net/http/cgi/host.go

package gofast

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

// Request hold information of a standard
// FastCGI request
type Request struct {
	ID       uint16
	Params   map[string]string
	Stdin    io.ReadCloser
	KeepConn bool
}

// client is the default implementation of Client
type client struct {
	root   string
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
func (c *client) readResponse(resp *ResponsePipe, req *Request) {
	var rec record

	defer c.ReleaseID(req.ID)
	defer resp.Close()
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
			panic(fmt.Sprintf("unexpected type %#v in readLoop", rec.h.Type))
		}
	}
}

// Do implements Client.Do
func (c *client) Do(req *Request) (resp *ResponsePipe, err error) {

	resp = NewResponsePipe()

	// FIXME: Should run read and write in parallel.
	//        Specification never said "write before read".
	//        Current workflow may block.

	if err = c.writeRequest(resp, req); err != nil {
		return
	}

	// NOTE: all errors return before readResponse
	go c.readResponse(resp, req)
	return
}

// NewRequest implements Client.NewRequest
func (c *client) NewRequest(r *http.Request) (req *Request) {
	req = &Request{
		ID:     c.AllocID(),
		Params: make(map[string]string),
	}

	// if no http request, return here
	if r == nil {
		return
	}

	// define some required cgi parameters
	// with the given http request

	// refer from nginx fastcgi_params
	// fastcgi_param  SCRIPT_FILENAME  $document_root$fastcgi_script_name;
	// fastcgi_split_path_info ^(.+\.php)(/?.+)$;
	// fastcgi_param PATH_INFO $fastcgi_path_info;
	// fastcgi_param PATH_TRANSLATED $document_root$fastcgi_path_info;
	// fastcgi_param  QUERY_STRING       $query_string;
	// fastcgi_param  REQUEST_METHOD     $request_method;
	// fastcgi_param  CONTENT_TYPE       $content_type;
	// fastcgi_param  CONTENT_LENGTH     $content_length;
	// fastcgi_param  SCRIPT_NAME        $fastcgi_script_name;
	// fastcgi_param  REQUEST_URI        $request_uri;
	// fastcgi_param  DOCUMENT_URI       $document_uri;
	// fastcgi_param  DOCUMENT_ROOT      $document_root;
	// fastcgi_param  SERVER_PROTOCOL    $server_protocol;
	// fastcgi_param  HTTPS              $https if_not_empty;
	// fastcgi_param  GATEWAY_INTERFACE  CGI/1.1;
	// fastcgi_param  SERVER_SOFTWARE    nginx/$nginx_version;
	// fastcgi_param  REMOTE_ADDR        $remote_addr;
	// fastcgi_param  REMOTE_PORT        $remote_port;
	// fastcgi_param  SERVER_ADDR        $server_addr;
	// fastcgi_param  SERVER_PORT        $server_port;
	// fastcgi_param  SERVER_NAME        $server_name;
	// # PHP only, required if PHP was built with --enable-force-cgi-redirect
	// fastcgi_param  REDIRECT_STATUS    200;

	fastcgiScriptName := r.URL.Path

	var fastcgiPathInfo string
	pathinfoRe := regexp.MustCompile(`^(.+\.php)(/?.+)$`)
	if matches := pathinfoRe.FindStringSubmatch(fastcgiScriptName); len(matches) > 0 {
		fastcgiScriptName, fastcgiPathInfo = matches[1], matches[2]
	}
	var isHTTPS string
	if r.URL.Scheme == "https" || r.URL.Scheme == "wss" {
		isHTTPS = "on"
	}

	remoteAddr, remotePort, _ := net.SplitHostPort(r.RemoteAddr)
	_, serverPort, err := net.SplitHostPort(r.URL.Host)
	if err != nil {
		if r.URL.Scheme == "https" || r.URL.Scheme == "wss" {
			serverPort = "443"
		} else {
			serverPort = "80"
		}
	}

	req.Params["SCRIPT_FILENAME"] = filepath.Join(c.root, fastcgiScriptName)
	req.Params["PATH_INFO"] = fastcgiPathInfo
	req.Params["PATH_TRANSLATED"] = filepath.Join(c.root, fastcgiPathInfo)
	req.Params["QUERY_STRING"] = r.URL.RawQuery
	req.Params["REQUEST_METHOD"] = r.Method
	req.Params["CONTENT_TYPE"] = r.Header.Get("Content-Type")
	req.Params["CONTENT_LENGTH"] = r.Header.Get("Content-Length")
	req.Params["SCRIPT_NAME"] = fastcgiScriptName
	req.Params["REQUEST_URI"] = r.RequestURI
	req.Params["DOCUMENT_URI"] = r.URL.Path
	req.Params["DOCUMENT_ROOT"] = c.root
	req.Params["SERVER_PROTOCOL"] = r.Proto
	req.Params["HTTPS"] = isHTTPS
	req.Params["GATEWAY_INTERFACE"] = "CGI/1.1"
	req.Params["SERVER_SOFTWARE"] = "appnode"
	req.Params["REMOTE_ADDR"] = remoteAddr
	req.Params["REMOTE_PORT"] = remotePort
	// req.Params["SERVER_ADDR"] = ""
	req.Params["SERVER_PORT"] = serverPort
	req.Params["SERVER_NAME"] = r.Host
	req.Params["REDIRECT_STATUS"] = "200"

	// http header
	for k, v := range r.Header {
		formattedKey := strings.Replace(strings.ToUpper(k), "-", "_", -1)
		if formattedKey == "CONTENT_TYPE" || formattedKey == "CONTENT_LENGTH" {
			continue
		}

		key := "HTTP_" + formattedKey
		var value string
		if len(v) > 0 {
			value = v[0]
		}
		req.Params[key] = value
	}

	// pass body (io.ReadCloser) to stdio
	req.Stdin = r.Body

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
}

// NewClient returns a Client of the given
// connection (net.Conn).
//
// limit is the maximum number of request that the
// applcation support. 0 means the maximum number
// available for 16bit request id (65536).
// Default 0.
//
func NewClient(root string, conn net.Conn, limit uint32) Client {
	cid := make(chan uint16)

	if limit == 0 || limit > 65536 {
		limit = 65536
	}
	go func(maxID uint16) {
		for i := uint16(0); i < maxID; i++ {
			cid <- i
		}
		cid <- uint16(maxID)
	}(uint16(limit - 1))

	return &client{
		root:   root,
		conn:   newConn(conn),
		chanID: cid,
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
	wg := new(sync.WaitGroup)
	wg.Add(2)

	go func() {
		defer wg.Done()
		err = pipes.writeResponse(rw)
	}()

	go func() {
		defer wg.Done()
		err = pipes.writeError(ew)
	}()

	// blocks until all reads and writes are done
	wg.Wait()
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
