package phpfpm_test

import (
	"os"
	"path"
	"testing"
	"time"

	"github.com/yookoala/gofast/tools/phpfpm"
)

var username, basepath, pathToPhpFpm string

func init() {
	var err error
	basepath, err = os.Getwd()
	if err != nil {
		panic(err)
	}
	basepath = path.Join(basepath, "_test")

	// defined in environment
	if pathToPhpFpm = os.Getenv("TEST_PHPFPM_PATH"); pathToPhpFpm != "" {
		// do nothing
	} else if pathToPhpFpm, err = phpfpm.FindBinary(phpfpm.ReadPaths(os.Getenv("PATH"))...); err != nil {
		panic(err)
	}

	username = os.Getenv("USER")
}

func TestNew(t *testing.T) {
	path := pathToPhpFpm
	process := phpfpm.NewProcess(path)
	if want, have := path, process.Exec; want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}

func TestProcess_SetPrefix(t *testing.T) {
	path := pathToPhpFpm
	process := phpfpm.NewProcess(path)
	process.SetDatadir(basepath + "/var")
	if want, have := basepath+"/var/phpfpm.pid", process.PidFile; want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
	if want, have := basepath+"/var/phpfpm.error_log", process.ErrorLog; want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
	if want, have := basepath+"/var/phpfpm.sock", process.Listen; want != have {
		t.Errorf("expected %#v, got %#v", want, have)
	}
}

func TestProcess_Address(t *testing.T) {
	var network, address string
	process := &phpfpm.Process{}

	process.Listen = "192.168.123.456:12345"
	network, address = process.Address()
	if want, have := "tcp", network; want != have {
		t.Errorf("expected %#v; got %#v", want, have)
	}
	if want, have := "192.168.123.456:12345", address; want != have {
		t.Errorf("expected %#v; got %#v", want, have)
	}

	process.Listen = "12345"
	network, address = process.Address()
	if want, have := "tcp", network; want != have {
		t.Errorf("expected %#v; got %#v", want, have)
	}
	if want, have := ":12345", address; want != have {
		t.Errorf("expected %#v; got %#v", want, have)
	}

	process.Listen = "hello.sock"
	network, address = process.Address()
	if want, have := "unix", network; want != have {
		t.Errorf("expected %#v; got %#v", want, have)
	}
	if want, have := "hello.sock", address; want != have {
		t.Errorf("expected %#v; got %#v", want, have)
	}

	process.Listen = "/path/to/hello.sock"
	network, address = process.Address()
	if want, have := "unix", network; want != have {
		t.Errorf("expected %#v; got %#v", want, have)
	}
	if want, have := "/path/to/hello.sock", address; want != have {
		t.Errorf("expected %#v; got %#v", want, have)
	}

}

func TestProcess_StartStop(t *testing.T) {
	path := pathToPhpFpm
	process := phpfpm.NewProcess(path)
	process.SetDatadir(basepath + "/var")
	process.User = username
	process.SaveConfig(basepath + "/etc/test.startstop.conf")

	if err := process.Start(); err != nil {
		t.Errorf("unexpected error: %s", err.Error())
		return
	}

	go func() {
		// do something that needs phpfpm
		// ...
		time.Sleep(time.Millisecond * 50)
		if err := process.Stop(); err != nil {
			panic(err)
		}
	}()

	if err := process.Wait(); err != nil {
		t.Errorf("unexpected error: %#v", err.Error())
	}
}

func ExampleProcess() {

	process := phpfpm.NewProcess(pathToPhpFpm)

	// SetDatadir equals to running these 3 settings:
	// process.PidFile  = basepath + "/phpfpm.pid"
	// process.ErrorLog = basepath + "/phpfpm.error_log"
	// process.Listen   = basepath + "/phpfpm.sock"
	process.SetDatadir(basepath + "/var")
	process.User = username

	// save the config file to basepath + "/etc/php-fpm.conf"
	process.SaveConfig(basepath + "/etc/example.conf")
	process.Start()

	go func() {
		// do something that needs phpfpm
		// ...
		time.Sleep(time.Millisecond * 50)
		process.Stop()
	}()

	process.Wait()

	// Output:
}
