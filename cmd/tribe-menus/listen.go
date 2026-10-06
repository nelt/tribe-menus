package main

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/nelt/tribe-menus/internal/config"
)

// systemdFirstFD is the first file descriptor passed by systemd (SD_LISTEN_FDS_START). A
// variable for the tests.
var systemdFirstFD = 3

// systemdVariables are the variables of the socket activation protocol of systemd.
var systemdVariables = []string{"LISTEN_PID", "LISTEN_FDS", "LISTEN_FDNAMES"}

// systemdSocketCount returns the number of sockets systemd passes to the process pid, read
// from the environment by getenv: 0 when none is passed to it (ADR 0015, point 11).
func systemdSocketCount(getenv func(string) string, pid int) (int, error) {
	listenPID := getenv("LISTEN_PID")
	if listenPID == "" {
		return 0, nil
	}
	p, err := strconv.Atoi(listenPID)
	if err != nil {
		return 0, errors.New("systemd socket: LISTEN_PID is not a number")
	}
	if p != pid {
		return 0, nil
	}
	n, err := strconv.Atoi(getenv("LISTEN_FDS"))
	if err != nil || n < 0 {
		return 0, errors.New("systemd socket: LISTEN_FDS is not a count")
	}
	return n, nil
}

// listen returns the listener of the server: the socket passed by systemd for
// config.ListenSystemd, or a TCP socket on the address otherwise. A socket passed by
// systemd that the address does not expect is refused, as well as a count other than one.
func listen(addr string, getenv func(string) string, pid int) (net.Listener, error) {
	n, err := systemdSocketCount(getenv, pid)
	if err != nil {
		return nil, err
	}
	if addr != config.ListenSystemd {
		if n > 0 {
			return nil, errors.New("a socket is passed by systemd, but the config file listens on TCP")
		}
		return net.Listen("tcp", addr)
	}
	if n != 1 {
		return nil, fmt.Errorf("systemd socket: %d passed, want 1", n)
	}
	// Not inherited by a child process: the protocol is for this process only.
	for _, v := range systemdVariables {
		if err := os.Unsetenv(v); err != nil {
			return nil, fmt.Errorf("systemd socket: %w", err)
		}
	}
	f := os.NewFile(uintptr(systemdFirstFD), "systemd socket")
	listener, err := net.FileListener(f)
	// FileListener works on a copy of the descriptor.
	if closeErr := f.Close(); err == nil && closeErr != nil {
		_ = listener.Close()
		err = closeErr
	}
	if err != nil {
		return nil, fmt.Errorf("systemd socket: %w", err)
	}
	// The file of the socket belongs to systemd, which keeps it across restarts.
	if unix, ok := listener.(*net.UnixListener); ok {
		unix.SetUnlinkOnClose(false)
	}
	return listener, nil
}
