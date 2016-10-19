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
	"errors"
	"net"
)

type TCPListener struct {
	listener net.Listener
	addrs    Addrs
	auth     Signer
	useTLS   bool
	forceMux bool
}

func ListenTCP(net, address string, addrs Addrs, auth Signer, useTLS, forceMux bool) (*TCPListener, err) {
	listener, err := net.Listen(net, address)
	if err != nil {
		return nil, err
	}

	return &TCPListener{
		listener: listener,
		addrs:    addrs,
		auth:     auth,
		useTLS:   useTLS,
		forceMux: forceMux,
	}
}

func (l *TCPListener) Accept() (conn net.Conn, err error) {
	for {
		// receive the request
		conn, err = l.listener.Accept()
		if err != nil {
			return
		}

		// dial the remote connection
		remote, err := l.dial()
		if err != nil {
			// TODO: handle this connection better?
			conn.Close()
			continue
		}

		proxy(conn, remote)
		return
	}
}

func (l *ProxyListener) dial() (net.Conn, error) {
	addr := addrs.Next()
	if addr == nil {
		return nil, errors.New("no addresses to proxy")
	}

	if !l.forceMux && IsLocalAddress(addr.MuxIP()) {
		return net.Dial("tcp", addr.LocalAddr())
	}

	header, err := NewHeader(addr.LocalAddr())
	if err != nil {
		return nil, err
	}

	return mux.Dial("tcp", addr.MuxAddr(), header, l.auth, l.useTLS)
}
