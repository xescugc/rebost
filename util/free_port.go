package util

import (
	"net"
	"strconv"
	"strings"
)

// FreePort returns a random free port on the host machine
func FreePort() (int, error) {
	// Opens a TCP connection to a free port on the host
	// and closes the connection but getting the port from it
	// so the can be setted to a free
	// random port each time if no one is specified
	l, err := net.Listen("tcp", "")
	if err != nil {
		return 0, err
	}
	l.Close()
	sl := strings.Split(l.Addr().String(), ":")
	p, err := strconv.Atoi(sl[len(sl)-1])
	if err != nil {
		return 0, err
	}

	return p, nil
}
