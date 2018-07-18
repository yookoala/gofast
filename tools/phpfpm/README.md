# phpfpm [![GoDoc](https://godoc.org/github.com/yookoala/gofast/tools/phpfpm?status.svg)][godoc]

**phpfpm** is a minimalistic php-fpm process manager written
in [go][golang].

It generates config file for a simple php-fpm process with 1 pool
and listen to 1 address only.

This is a fringe case, I know. Just hope it might be useful for
someone else.

[godoc]: https://godoc.org/github.com/yookoala/gofast/tools/phpfpm
[golang]: https://golang.org

Usage
-----

```go
package main

import "github.com/yookoala/gofast/tools/phpfpm"

func main() {

  fpm := phpfpm.NewProcess("/usr/sbin/php5-fpm")

  // config to save pidfile, log to "/home/foobar/var"
  // also have the socket file "/home/foobar/var/php-fpm.sock"
  fpm.SetDatadir("/home/foobar/var")

  // save the config file to basepath + "/etc/php-fpm.conf"
  fpm.SaveConfig(basepath + "/etc/php-fpm.conf")
  fpm.Start()

  go func() {

    // do something that needs fpm
    // ...
    fpm.Stop()

  }()

  // will wait for phpfpm to exit
  fpm.Wait()

}

```

License
-------

This software is license under [MIT License][mit-license]. You
may find [a copy of the license][license] in this repository.

[mit-license]: https://opensource.org/licenses/MIT
[license]: /LICENSE
