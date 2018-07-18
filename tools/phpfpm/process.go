package phpfpm

import (
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path"
	"regexp"
	"strconv"
	"syscall"
	"time"

	"github.com/go-ini/ini"
)

// Process describes a minimalistic php-fpm config
// that runs only 1 pool
type Process struct {

	// path to php-fpm executable
	Exec string

	// path to the config file
	ConfigFile string

	// username of the FastCGI process
	User string

	// The address on which to accept FastCGI requests.
	// Valid syntaxes are: 'ip.add.re.ss:port', 'port',
	// '/path/to/unix/socket'. This option is mandatory for each pool.
	Listen string

	// path of the PID file
	PidFile string

	// path of the error log
	ErrorLog string

	// cmd stores the command of the running process
	cmd *exec.Cmd
}

// NewProcess creates a new process descriptor
func NewProcess(phpFpm string) *Process {
	return &Process{
		Exec: phpFpm,
	}
}

// SaveConfig generates config file according to the
// process attributes
func (proc *Process) SaveConfig(path string) {
	proc.ConfigFile = path
	proc.Config().SaveTo(proc.ConfigFile)
}

// Config generates an minimalistic config ini file
// in *ini.File format. You may then use SaveTo(path)
// to save it
func (proc *Process) Config() (f *ini.File) {
	f = ini.Empty()
	f.NewSection("global")
	f.Section("global").NewKey("pid", proc.PidFile)
	f.Section("global").NewKey("error_log", proc.ErrorLog)
	f.NewSection("www")
	f.Section("www").NewKey("listen", proc.Listen)
	f.Section("www").NewKey("pm", "dynamic")
	f.Section("www").NewKey("pm.max_children", "5")
	f.Section("www").NewKey("pm.start_servers", "2")
	f.Section("www").NewKey("pm.min_spare_servers", "1")
	f.Section("www").NewKey("pm.max_spare_servers", "3")
	if proc.User != "" {
		f.Section("www").NewKey("user", proc.User)
	}
	return
}

// SetDatadir sets default config values according
// with reference to the folder prefix
//
// Equals to running these 3 statements:
//   process.PidFile  = basepath + "/phpfpm.pid"
//   process.ErrorLog = basepath + "/phpfpm.error_log"
//   process.Listen   = basepath + "/phpfpm.sock"
func (proc *Process) SetDatadir(prefix string) {
	// FIXME: add error if the prefix folder doesn't exists
	// or is not a folder
	proc.PidFile = path.Join(prefix, "phpfpm.pid")
	proc.ErrorLog = path.Join(prefix, "phpfpm.error_log")
	proc.Listen = path.Join(prefix, "phpfpm.sock")
}

// Start starts the php-fpm process
// in foreground mode instead of daemonize
func (proc *Process) Start() (err error) {
	proc.cmd = &exec.Cmd{
		Path: proc.Exec,
		Args: append([]string{proc.Exec},
			"--fpm-config", proc.ConfigFile,
			"-e"), // extended information
	}

	if cmbOut, err := proc.cmd.CombinedOutput(); err != nil {
		var ok bool
		var exitErr *exec.ExitError
		if exitErr, ok = err.(*exec.ExitError); !ok {
			// no an exit error
			return err
		}
		if !exitErr.ProcessState.Success() {
			// unsuccessful exitErr
			return fmt.Errorf("unsuccessful exit. error %s\noutput:\n%s",
				exitErr.ProcessState, cmbOut)
		}
	}

	pid := <-proc.waitPid()
	spawned, err := os.FindProcess(pid)
	if err != nil {
		return
	}
	proc.cmd.Process = spawned

	// wait until the service is connectable
	// or time out
	select {
	case <-proc.waitConn():
		// do nothing
	case <-time.After(time.Second * 10):
		// wait 10 seconds or timeout
		err = fmt.Errorf("time out")
	}

	return
}

// read pid from pid
func (proc *Process) pid() (pid int, err error) {
	f, err := os.Open(proc.PidFile)
	if err != nil {
		return
	}

	b, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	pid64, err := strconv.ParseInt(string(b), 10, 64)
	pid = int(pid64)
	return
}

// wait until pid file readable
func (proc *Process) waitPid() <-chan int {
	cout := make(chan int)
	go func() {
		for {
			if pid, err := proc.pid(); err != nil {
				time.Sleep(time.Millisecond * 2)
			} else {
				cout <- pid
				break
			}
		}
	}()
	return cout
}

func (proc *Process) waitConn() <-chan net.Conn {
	chanConn := make(chan net.Conn)
	go func() {
		for {
			if conn, err := net.Dial(proc.Address()); err != nil {
				time.Sleep(time.Millisecond * 2)
			} else {
				chanConn <- conn
				break
			}
		}
	}()
	return chanConn
}

// Address returns networkk and address that fits
// the use of either net.Dial or net.Listen
func (proc *Process) Address() (network, address string) {
	reIP := regexp.MustCompile("^(\\d{1,3}\\.\\d{1,3}\\.\\d{1,3}\\.\\d{1,3})\\:(\\d{2,5}$)")
	rePort := regexp.MustCompile("^(\\d+)$")
	switch {
	case reIP.MatchString(proc.Listen):
		network = "tcp"
		address = proc.Listen
	case rePort.MatchString(proc.Listen):
		network = "tcp"
		address = ":" + proc.Listen
	default:
		network = "unix"
		address = proc.Listen
	}
	return
}

// Stop stops the php-fpm process with SIGINT
// instead of killing
func (proc *Process) Stop() error {
	return proc.cmd.Process.Signal(os.Interrupt)
}

// Wait wait for the process to finish
func (proc *Process) Wait() (err error) {
	for {
		if err = proc.cmd.Process.Signal(syscall.Signal(0)); err != nil {
			switch err.Error() {
			case "os: process already finished":
				fallthrough
			case "no such process":
				err = nil
			}
			break
		} else {
			time.Sleep(time.Millisecond * 2)
		}
	}
	return
}
