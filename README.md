# gofast [![GoDoc][godoc-badge]][godoc] [![Go Report Card][goreport-badge]][goreport] [![Travis CI results][travis-badge]][travis] [![GitHub Action Test result][github-action-badge]][github-action]

**gofast** is a [FastCGI][fastcgi] "client" library written purely in
[golang][golang].

## Contents

* [What does it do, really?](#what-does-it-do-really)
* [Why?](#why)
* [How to Use?](#how-to-use)
  * [Simple Example](#simple-example)
  * [Advanced Examples](#advanced-examples)
    * [Normal PHP Application](#normal-php-application)
    * [Customizing Request Session with Middleware](#customizing-request-session-with-middleware)
    * [FastCGI Authorizer](#fastcgi-authorizer)
    * [FastCGI Filter](#fastcgi-filter)
    * [Pooling Clients](#pooling-clients)
  * [Full Examples](#full-examples)
* [Author](#author)
* [Contributing](#contributing)
* [Licence](#licence)

[fastcgi]: http://www.mit.edu/~yandros/doc/specs/fcgi-spec.html
[godoc]: https://godoc.org/github.com/yookoala/gofast
[godoc-badge]: https://godoc.org/github.com/yookoala/gofast?status.svg
[travis]: https://travis-ci.com/github/yookoala/gofast?branch=main
[travis-badge]: https://api.travis-ci.com/yookoala/gofast.svg?branch=main
[github-action]: https://github.com/yookoala/gofast/actions?query=workflow%3ATests+branch%3Amain
[github-action-badge]: https://github.com/yookoala/gofast/workflows/Tests/badge.svg?branch=main
[goreport]: https://goreportcard.com/report/github.com/yookoala/gofast
[goreport-badge]: https://goreportcard.com/badge/github.com/yookoala/gofast
[golang]: https://golang.org

## What does it do, really?

In FastCGI specification, a FastCGI system has 2 components: **(a) web
server**; and **(b) application server**. A web server should hand over
request information to the application server through socket. The
application server always listens to the socket and response to
socket request accordingly.

[![visitor → web server → application server → web server → visitor][fastcgi-illustration]][fastcgi-illustration]

[fastcgi-illustration]: docs/fastcgi-illustration.svg

**gofast** help you to write the code on the **web server** part of this
picture. It helps you to pass the request to application server and
receive response from it.

You may think of **gofast** as a "client library" to consume
any FastCGI application server.

## Why?

Many popular languages (e.g. [Python][python/webservers],
[PHP][php-fpm], [nodejs][node-fastcgi]) has FastCGI application
server implementations. With **gofast**, you may mix using the languages
in a simple way.

Also, this is fun to do :-)

[php-fpm]: http://php.net/manual/en/install.fpm.php
[python/webservers]: https://docs.python.org/3.1/howto/webservers.html
[node-fastcgi]: https://www.npmjs.com/package/node-fastcgi


## How to Use?

You basically would use the `Handler` as [http.Handler]. You can further mux it
with [default ServeMux][http.NewServeMux] or other compatible routers (e.g.
[gorilla][gorilla], [pat][pat]). You then serve your fastcgi within this
golang http server.

[http.Handler]: https://golang.org/pkg/net/http/#Handler
[mux]: https://golang.org/pkg/net/http/#ServeMux
[http.NewServeMux]: https://golang.org/pkg/net/http/#NewServeMux
[gorilla]: https://github.com/gorilla/mux
[pat]: https://github.com/gorilla/pat

### Simple Example

Please note that this is only the **web server** component. You need to start
your **application** component elsewhere.

```go
// this is a very simple fastcgi web server
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/yookoala/gofast"
)

func main() {
	// Get fastcgi application server tcp address
	// from env FASTCGI_ADDR. Then configure
	// connection factory for the address.
	address := os.Getenv("FASTCGI_ADDR")
	connFactory := gofast.SimpleConnFactory("tcp", address)

	// route all requests to a single php file
	http.Handle("/", gofast.NewHandler(
		gofast.NewFileEndpoint("/var/www/html/index.php")(gofast.BasicSession),
		gofast.SimpleClientFactory(connFactory),
	))

	// serve at 8080 port
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```

### Advanced Examples

#### Normal PHP Application

To serve normal PHP application, you'd need to:

1. Serve the static assets from file system; and
1. Serve only the path with relevant PHP file.

<details>
<summary>Code</summary>
<div>


```go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/yookoala/gofast"
)

func main() {
	// Get fastcgi application server tcp address
	// from env FASTCGI_ADDR. Then configure
	// connection factory for the address.
	address := os.Getenv("FASTCGI_ADDR")
	connFactory := gofast.SimpleConnFactory("tcp", address)

	// handles static assets in the assets folder
	http.Handle("/assets/",
		http.StripPrefix("/assets/",
			http.FileServer(http.FileSystem(http.Dir("/var/www/html/assets")))))

	// route all requests to relevant PHP file
	http.Handle("/", gofast.NewHandler(
		gofast.NewPHPFS("/var/www/html")(gofast.BasicSession),
		gofast.SimpleClientFactory(connFactory),
	))

	// serve at 8080 port
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```

</div>
</details>


#### Customizing Request Session with Middleware

Each web server request will result in a [gofast.Request][gofast-request].
And each [gofast.Request][gofast-request] will first run through SessionHandler
before handing to the `Do()` method of [gofast.Client][gofast-client].

The default [gofast.BasicSession][gofast-basicsession] implementation does
nothing. The library function like [gofast.NewPHPFS][gofast-phpfs],
[gofast.NewFileEndpoint][gofast-file-endpoint] are [gofast.Middleware][gofast-middleware]
implementations, which are lower level middleware chains.

So you may customize your own session by implemention [gofast.Middleware][gofast-middleware].

<details>
<summary>Code</summary>
<div>

```go

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/yookoala/gofast"
)

func main() {
	// Get fastcgi application server tcp address
	// from env FASTCGI_ADDR. Then configure
	// connection factory for the address.
	address := os.Getenv("FASTCGI_ADDR")
	connFactory := gofast.SimpleConnFactory("tcp", address)

	// a custom authentication handler
	customAuth := func(inner gofast.SessionHandler) gofast.SessionHandler {
		return func(client gofast.Client, req *gofast.Request) (*gofast.ResponsePipe, error) {
			user, err := someCustomAuth(
				req.Raw.Header.Get("Authorization"))
			if err != nil {
				// if login not success
				return nil, err
			}
			// set REMOTE_USER accordingly
			req.Params["REMOTE_USER"] = user
			// run inner session handler
			return inner(client, req)
		}
	}

	// session handler
	sess := gofast.Chain(
		customAuth,            // maps REMOTE_USER
		gofast.BasicParamsMap, // maps common CGI parameters
		gofast.MapHeader,      // maps header fields into HTTP_* parameters
		gofast.MapRemoteHost,  // maps REMOTE_HOST
	)(gofast.BasicSession)

	// route all requests to a single php file
	http.Handle("/", gofast.NewHandler(
		gofast.NewFileEndpoint("/var/www/html/index.php")(sess),
		gofast.SimpleClientFactory(connFactory),
	))

	// serve at 8080 port
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```
</div>
</details>

[gofast-basicsession]: https://godoc.org/github.com/yookoala/gofast#BasicSession
[gofast-request]: https://godoc.org/github.com/yookoala/gofast#Request
[gofast-client]: https://godoc.org/github.com/yookoala/gofast#Client
[gofast-phpfs]: https://godoc.org/github.com/yookoala/gofast#NewPHPFS
[gofast-file-endpoint]: https://godoc.org/github.com/yookoala/gofast#NewFileEndpoint
[gofast-middleware]: https://godoc.org/github.com/yookoala/gofast#Middleware

#### FastCGI Authorizer

FastCGI specified an [authorizer role][fastcgi-authorizer] for authorizing
an HTTP request with an "authorizer application". As different from a usual
FastCGI application (i.e. **responder**), it only does authorization check.

<details>
<summary>Summary of Spec</summary>
<div>

Before actually serving an HTTP request, a web server can format a normal
FastCGI request to the Authorizer application with only FastCGI parameters
(`FCGI_PARAMS` stream). This application is responsible to determine if the
request is properly authenticated and authorized for the request.

If valid,

* The authorizer application should response with HTTP status `200` (OK).

* It may add additional variables (e.g. `SOME-HEADER`) to the subsequence
  request by adding `Variable-SOME-HEADER` header field to its response to
  web server.

* The web server will create a new HTTP request from the old one, appending
  the additional header variables (e.g. `Some-Header`), then send the modified
  request to the subquence application.

If invalid,

* The authorizer application should response with HTTP status that is NOT
  `200`, and the content to display for failed login.

* The webserver will skip the responder and directly show the authorizer's
  response.

</div>
</details>

<details>
<summary>Code</summary>
<div>

```go

package main

import (
	"net/http"
	"time"

	"github.com/yookoala/gofast"
)

func myApp() http.Handler {
  // ... any normal http.Handler, using gofast or not
	return h
}

func main() {
	address := os.Getenv("FASTCGI_ADDR")
	connFactory := gofast.SimpleConnFactory("tcp", address)
	clientFactory := gofast.SimpleClientFactory(connFactory)

	// authorization with php
	authSess := gofast.Chain(
		gofast.NewAuthPrepare(),
		gofast.NewFileEndpoint("/var/www/html/authorization.php"),
	)(gofast.BasicSession)
	authorizer := gofast.NewAuthorizer(
		authSess,
		gofast.SimpleConnFactory(network, address)
	)

	// wrap the actual app
	http.Handle("/", authorizer.Wrap(myApp()))

	// serve at 8080 port
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```

</div>
</details>


[fastcgi-authorizer]: http://www.mit.edu/~yandros/doc/specs/fcgi-spec.html#S6.3


#### FastCGI Filter

FastCGI specified a [filter role][fastcgi-filter] for filtering web server
assets before sending out. As different from a usual FastCGI application
(i.e. **responder**), the requested data is on the web server side. So the
web server will pass those data to the application when requested.

<details>
<summary>Code</summary>
<div>

```go

package main

import (
	"net/http"
	"time"

	"github.com/yookoala/gofast"
)

func main() {
	address := os.Getenv("FASTCGI_ADDR")
	connFactory := gofast.SimpleConnFactory("tcp", address)
	clientFactory := gofast.SimpleClientFactory(connFactory)

	// Note: The local file system "/var/www/html/" only need to be
	// local to web server. No need for the FastCGI application to access
	// it directly.
	connFactory := gofast.SimpleConnFactory(network, address)
	http.Handle("/", gofast.NewHandler(
		gofast.NewFilterLocalFS("/var/www/html/")(gofast.BasicSession),
		clientFactory,
	))

	// serve at 8080 port
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```

</div>
</details>

[fastcgi-filter]: http://www.mit.edu/~yandros/doc/specs/fcgi-spec.html#S6.4


#### Pooling Clients

To have a better, more controlled, scaling property, you may
scale the clients with ClientPool.

<details>
<summary>Code</summary>
<div>


```go
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/yookoala/gofast"
)

func main() {
	// Get fastcgi application server tcp address
	// from env FASTCGI_ADDR. Then configure
	// connection factory for the address.
	address := os.Getenv("FASTCGI_ADDR")
	connFactory := gofast.SimpleConnFactory("tcp", address)

	// handles static assets in the assets folder
	http.Handle("/assets/",
		http.StripPrefix("/assets/",
			http.FileSystem(http.Dir("/var/www/html/assets"))))

	// handle all scripts in document root
	// extra pooling layer
	pool := gofast.NewClientPool(
		gofast.SimpleClientFactory(connFactory),
		10, // buffer size for pre-created client-connection
		30*time.Second, // life span of a client before expire
	)
	http.Handle("/", gofast.NewHandler(
		gofast.NewPHPFS("/var/www/html")(gofast.BasicSession),
		pool.CreateClient,
	))

	// serve at 8080 port
	log.Fatal(http.ListenAndServe(":8080", nil))
}

```

</div>
</details>

### Full Examples

Please see the example usages:

* [PHP]
* [Python3]
* [nodejs]

[PHP]: example/php
[Python3]: example/python3
[nodejs]: example/nodejs


## Author

This library is written by [Koala Yeung][author@github].

[author@github]: https://github.com/yookoala/


## Contributing

Your are welcome to contribute to this library.

To report bug, please use the [issue tracker][issue tracker].

To fix an existing bug or implement a new feature, please:

1. Check the [issue tracker][issue tracker] and [pull requests][pull requests] for existing discussion.
2. If not, please open a new issue for discussion.
3. Write tests.
4. Open a pull request referencing the issue.
5. Have fun :-)

[issue tracker]: https://github.com/yookoala/gofast/issues
[pull requests]: https://github.com/yookoala/gofast/pulls


## Licence

This library is release under a BSD-like licence. Please find the
[LICENCE][LICENCE] file in this repository

[LICENCE]: /LICENCE
