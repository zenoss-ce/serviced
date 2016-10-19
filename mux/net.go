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
<<<<<<< Updated upstream
=======
	"crypto/tls"
>>>>>>> Stashed changes
	"io"
	"net"
)

// Dial dials a muxed connection with the provided header and auth
<<<<<<< Updated upstream
func Dial(net, address string, header *Header, auth Signer) (conn net.Conn, err error) {
=======
func Dial(net, address string, header *Header, auth Signer, useTLS bool) (conn net.Conn, err error) {
>>>>>>> Stashed changes
	conn, err = net.Dial(net, address)
	if err != nil {
		return
	}

<<<<<<< Updated upstream
=======
	if useTLS {
		conn = tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
	}

>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
func Listen(net, address, auth Verifier) (*Listener, err) {
=======
func Listen(net, address string, auth Verifier) (*Listener, error) {
>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
		remote, err := l.listener.Accept()
=======
		conn, err = l.listener.Accept()
>>>>>>> Stashed changes
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
<<<<<<< Updated upstream
		conn, err = net.Dial(l.listener.Addr().Network(), header.Address())
=======
		local, err := net.Dial(l.listener.Addr().Network(), header.Address())
>>>>>>> Stashed changes
		if err != nil {
			// TODO: write dialer error to the connection
			remote.Close()
			continue
		}

<<<<<<< Updated upstream
		// proxy the request and return the local connection
		proxy(remote, conn)
=======
		// proxy the request and return the connection
		proxy(conn, local)
>>>>>>> Stashed changes
		return
	}
}

<<<<<<< Updated upstream
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

=======
>>>>>>> Stashed changes
// Addr returns the listener address
func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}

// Close closes the listener
func (l *Listener) Close() error {
	return l.listener.Close()
}
<<<<<<< Updated upstream
=======

func proxy(a, b net.Conn) {
	go func() {
		io.Copy(a, b)
		b.Close()
		a.Close()
	}()

	go func() {
		io.Copy(b, a)
		a.Close()
		b.Close()
	}()
}
>>>>>>> Stashed changes
