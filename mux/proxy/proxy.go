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

package proxy

import (
	"crypto/tls"
	"errors"
	"io"
	"net"

	"github.com/control-center/serviced/net/mux"
)

type Proxy interface {
	Serve(l net.Listener)
}

func ServeProxy(p Proxy, l net.Listener) {

type TCPProxy struct {

}

type HTTPProxy struct {
}




// Proxy proxies addresses through an exposed listener
type Proxy struct {
	addrs     Addrs
	tlsConfig *tls.Config
	auth      mux.Signer
	forceMux  bool
}

// NewProxy instantiates a new proxy
func NewProxy(addrs Addrs, tlsConfig *tls.Config, auth mux.Signer, forceMux bool) *Proxy {
	return &Proxy{
		addrs:     addrs,
		tlsConfig: tlsConfig,
		auth:      auth,
		forceMux:  forceMux,
	}
}

// ServeTCP proxies tcp connections
func (p *Proxy) ServeTCP(l net.Listener) {
	for {
		local, err := l.Accept()
		if err != nil {
			// TODO: log error
			return
		}

		remote, err := p.dial()
		if err != nil {
			// TODO: log error
			local.Close()
			continue
		}

		go func() {
			io.Copy(local, remote)
			remote.Close()
			local.Close()
		}()

		go func() {
			io.Copy(remote, local)
			local.Close()
			remote.Close()
		}()
	}
}

// ServeHTTP handles http connections
func (p *Proxy) ServeHTTP() {
}

func (p *Proxy) Handler(cancel <-chan struct{}) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Requ
}

// dial receieves an address and dials the remote
func (p *Proxy) dial() (net.Conn, error) {
	addr := p.addrs.Next()
	if addr == nil {
		return nil, errors.New("no addresses to proxy")
	}

	// look up the network of the listener
	network := p.listener.Addr().Network()

	// dial the mux or dial the local address
	if p.forceMux || !IsLocalAddress(addr.MuxIP()) {
		header, err := mux.NewHeader(addr, l.tlsConfig)
		if err != nil {
			// TODO: log error
			return nil, err
		}

		return mux.Dial(network, addr.MuxAddr(), header, l.auth)
	}

	return net.Dial(network, addr.LocalAddr())
}

// SetAddresses updates the list of available addresses to proxy
func (p *Proxy) SetAddresses(addrs []Addr) {
	p.addrs.Set(addrs)
}
