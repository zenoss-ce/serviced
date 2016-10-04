// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mux

import (
	"io"
	"net"
)

// Dial dials a muxed connection with the provided header and auth
func Dial(net, address string, header *Header, auth Signer) (conn net.Conn, err error) {
	conn, err = net.Dial(net, address)
	if err != nil {
		return
	}

	_, err = WriteHeader(conn, header, auth)
	if err != nil {
		conn.Close()
		return nil, err
	}

	return
}

// Listener is the mux listener
type Listener struct {
	listener net.Listener
	auth     Verifier
}

// Listen creates a listener with an auth verifier
func Listen(net, address, auth Verifier) (*Listener, err) {
	listener, err := net.Listen(net, address)
	if err != nil {
		return nil, err
	}

	return &Listener{listener: listener, auth: auth}, nil
}

// Accept accepts incoming connections to proxy to local connections
func (l *Listener) Accept() (conn net.Conn, err error) {
	for {
		// receive the request
		remote, err := l.listener.Accept()
		if err != nil {
			return
		}

		// read the header
		header, _, err := ReadHeader(remote, l.verifier)
		if err != nil {
			// TODO: write auth error to the connection
			remote.Close()
			continue
		}

		// dial the local connection
		conn, err = net.Dial(l.listener.Addr().Network(), header.Address())
		if err != nil {
			// TODO: write dialer error to the connection
			remote.Close()
			continue
		}

		// proxy the request and return the local connection
		proxy(remote, conn)
		return
	}
}

func proxy(remote, local net.Conn) {
	go func() {
		io.Copy(remote, local)
		local.Close()
		remote.Close()
	}()

	go func() {
		io.Copy(local, remote)
		remote.Close()
		local.Close()
	}()
}

// Addr returns the listener address
func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}

// Close closes the listener
func (l *Listener) Close() error {
	return l.listener.Close()
}
