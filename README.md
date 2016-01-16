gofast
======

**gofast** is a [FastCGI](http://www.fastcgi.com/devkit/doc/fcgi-spec.html)
client library written purely in go.


Why?
----
Many popular languages (e.g. [Python][python/webservers],
[PHP][php-fpm], [Ruby][rubygem/fcgi]) has FastCGI server
implementations. Developer used to proxy Nginx or Apache requests
to these FastCGI backend. What if go developers can use these
languages through the same protocol?

Some quick and dirty RPC could be handy with the help of all the
other languages support FastCGI. The only limit is your imagination.

Also, this is fun to do :-)

[php-fpm]: http://php.net/manual/en/install.fpm.php
[rubygem/fcgi]: https://rubygems.org/gems/fcgi/versions/0.9.2.1
[python/webservers]: https://docs.python.org/2/howto/webservers.html

Author
------

This library is written by [Koala Yeung][author@github].

[author@github]: https://github.com/yookoala/


Contirbuting
------------

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

Licence
-------

This library is release under a BSD-like licence. Please find the
[LICENCE][LICENCE] file in this repository

[LICENCE]: /LICENCE
