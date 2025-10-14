package utils

import (
	"fmt"
	"net"
)

type RemoteLogger struct {
	Port     int
	Listener net.Listener
	clients  []net.Conn
}

// NewRemoteLogger starts a TCP listener on the given port.
func NewRemoteLogger(port int) (*RemoteLogger, error) {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		rl := &RemoteLogger{}
		return rl, err
	}
	rl := &RemoteLogger{
		Port:     port,
		Listener: ln,
	}
	go rl.acceptClients()
	return rl, nil
}

// acceptClients accepts incoming TCP connections.
func (rl *RemoteLogger) acceptClients() {
	for {
		conn, err := rl.Listener.Accept()
		if err != nil {
			continue
		}
		rl.clients = append(rl.clients, conn)
	}
}

// Logf sends a formatted log message to all connected clients.
func (rl *RemoteLogger) Logf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	for _, conn := range rl.clients {
		fmt.Fprintln(conn, msg)
	}
}
