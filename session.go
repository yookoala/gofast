package gofast

import (
	"net"
	"path"
	"path/filepath"
	"regexp"
	"strings"
)

// SessionHandler handles the gofast *Reqeust with the provided given Client.
// The Client should properly handle the transport to the fastcgi application.
// Should do proper routing or other parameter mapping here.
type SessionHandler func(client Client, req *Request) (resp *ResponsePipe, err error)

// Handle implements SessionHandler
func (handler SessionHandler) Handle(client Client, req *Request) (*ResponsePipe, error) {
	return handler(client, req)
}

// Middleware transform a SessionHandler as another SessionHandler
type Middleware func(SessionHandler) SessionHandler

// Chain chains middlewares into a single middleware
func Chain(middlewares ...Middleware) Middleware {
	if len(middlewares) == 0 {
		return nil
	}
	return func(inner SessionHandler) (out SessionHandler) {
		out = inner
		for i := len(middlewares) - 1; i >= 0; i-- {
			out = middlewares[i](out)
		}
		return
	}
}

// BasicSession implements SessionHandlerFunc
func BasicSession(client Client, req *Request) (*ResponsePipe, error) {
	return client.Do(req)
}

// BasicParamsMap maps basic parameters
func BasicParamsMap(inner SessionHandler) SessionHandler {
	return func(client Client, req *Request) (*ResponsePipe, error) {

		r := req.Raw

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

		// the basic information here
		req.Params["CONTENT_TYPE"] = r.Header.Get("Content-Type")
		req.Params["CONTENT_LENGTH"] = r.Header.Get("Content-Length")
		req.Params["HTTPS"] = isHTTPS
		req.Params["GATEWAY_INTERFACE"] = "CGI/1.1"
		req.Params["REMOTE_ADDR"] = remoteAddr
		req.Params["REMOTE_PORT"] = remotePort
		req.Params["SERVER_PORT"] = serverPort
		req.Params["SERVER_NAME"] = r.Host
		req.Params["SERVER_PROTOCOL"] = r.Proto
		req.Params["SERVER_SOFTWARE"] = "gofast"
		req.Params["REDIRECT_STATUS"] = "200"
		req.Params["REQUEST_METHOD"] = r.Method
		req.Params["REQUEST_URI"] = r.RequestURI
		req.Params["QUERY_STRING"] = r.URL.RawQuery

		return inner.Handle(client, req)
	}
}

// FileSystemRouter handles ordinary filesystem based router
// like an old fashion PHP hosting environment
type FileSystemRouter struct {

	// DocRoot stores the ordinary Apache DocumentRoot parameter
	DocRoot string

	// Exts stores accepted extensions
	Exts []string

	// DirIndex stores ordinary Apache DirectoryIndex parameter
	// for to identify file to show in directory
	DirIndex []string
}

// Router returns Middleware
func (fs *FileSystemRouter) Router() Middleware {
	return func(inner SessionHandler) SessionHandler {
		return func(client Client, req *Request) (*ResponsePipe, error) {

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

			r := req.Raw
			fastcgiScriptName := r.URL.Path

			var fastcgiPathInfo string
			pathinfoRe := regexp.MustCompile(`^(.+\.php)(/?.+)$`)
			if matches := pathinfoRe.FindStringSubmatch(fastcgiScriptName); len(matches) > 0 {
				fastcgiScriptName, fastcgiPathInfo = matches[1], matches[2]
			}

			// should be handled by fs.Inner
			//req.Params["CONTENT_TYPE"] = r.Header.Get("Content-Type")
			//req.Params["CONTENT_LENGTH"] = r.Header.Get("Content-Length")
			//req.Params["HTTPS"] = isHTTPS
			//req.Params["GATEWAY_INTERFACE"] = "CGI/1.1"
			//req.Params["SERVER_SOFTWARE"] = "gofast"
			//req.Params["REMOTE_ADDR"] = remoteAddr
			//req.Params["REMOTE_PORT"] = remotePort
			//req.Params["SERVER_PORT"] = serverPort
			//req.Params["SERVER_NAME"] = r.Host
			//req.Params["SERVER_PROTOCOL"] = r.Proto
			//req.Params["REDIRECT_STATUS"] = "200"
			//req.Params["REQUEST_METHOD"] = r.Method
			//req.Params["REQUEST_URI"] = r.RequestURI
			//req.Params["QUERY_STRING"] = r.URL.RawQuery

			req.Params["PATH_INFO"] = fastcgiPathInfo
			req.Params["PATH_TRANSLATED"] = filepath.Join(fs.DocRoot, fastcgiPathInfo)
			req.Params["SCRIPT_NAME"] = fastcgiScriptName
			req.Params["SCRIPT_FILENAME"] = filepath.Join(fs.DocRoot, fastcgiScriptName)
			req.Params["DOCUMENT_URI"] = r.URL.Path
			req.Params["DOCUMENT_ROOT"] = fs.DocRoot

			// handle directory index
			urlPath := r.URL.Path
			if strings.HasSuffix(urlPath, "/") {
				urlPath = path.Join(urlPath, "index.php")
			}
			req.Params["SCRIPT_FILENAME"] = path.Join(fs.DocRoot, urlPath)

			return inner.Handle(client, req)
		}
	}
}

// MapHeader implement Middleware to map header field SomeRandomField to
// HTTP_SOME_RANDOM_FIELD
func MapHeader(inner SessionHandler) SessionHandler {
	return func(client Client, req *Request) (*ResponsePipe, error) {
		r := req.Raw

		// http header
		for k, v := range r.Header {
			formattedKey := strings.Replace(strings.ToUpper(k), "-", "_", -1)
			if formattedKey == "CONTENT_TYPE" || formattedKey == "CONTENT_LENGTH" {
				continue
			}

			key := "HTTP_" + formattedKey
			var value string
			if len(v) > 0 {
				//   refer to https://tools.ietf.org/html/rfc7230#section-3.2.2
				//
				//   A recipient MAY combine multiple header fields with the same field
				//   name into one "field-name: field-value" pair, without changing the
				//   semantics of the message, by appending each subsequent field value to
				//   the combined field value in order, separated by a comma.  The order
				//   in which header fields with the same field name are received is
				//   therefore significant to the interpretation of the combined field
				//   value; a proxy MUST NOT change the order of these field values when
				//   forwarding a message.
				value = strings.Join(v, ",")
			}
			req.Params[key] = value
		}

		return inner.Handle(client, req)
	}
}

// NewPHPFS returns a Session Middleware that prepare a classic PHP hosting
// environment for a HTTP session
func NewPHPFS(root string) Middleware {
	fs := &FileSystemRouter{
		DocRoot:  root,
		Exts:     []string{"php"},
		DirIndex: []string{"index.php"},
	}
	return Chain(
		BasicParamsMap,
		MapHeader,
		fs.Router(),
	)
}

// NewFileEndpoint returns a Session Middleware that prepares a PHP hosting
// environment for single entry point applications (e.g. Laravel)
func NewFileEndpoint(endpointFile string) Middleware {
	dir, webpath := filepath.Dir(endpointFile), "/"+filepath.Base(endpointFile)
	endpointRoutes := func(inner SessionHandler) SessionHandler {
		return func(client Client, req *Request) (*ResponsePipe, error) {
			r := req.Raw
			req.Params["REQUEST_URI"] = webpath + r.URL.RequestURI()
			req.Params["SCRIPT_NAME"] = webpath
			req.Params["SCRIPT_FILENAME"] = endpointFile
			req.Params["DOCUMENT_URI"] = r.URL.Path
			req.Params["DOCUMENT_ROOT"] = dir
			return inner.Handle(client, req)
		}
	}
	return Chain(
		BasicParamsMap,
		MapHeader,
		endpointRoutes,
	)
}
