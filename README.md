gofast
======

*gofast* is a [FastCGI](http://www.fastcgi.com/devkit/doc/fcgi-spec.html)
client written purely in go.


Why?
----
Many popular languages (e.g. Python, PHP, Ruby) has FastCGI server
implementations. Developer used to proxy Nginx or Apache requests
to these FastCGI backend. What if go developers can use these
languages through the same protocol?

Some quick and dirty RPC could be handy with the help of all the
other languages support FastCGI. The only limit is your imagination.

Also, this is fun to do :-)

