// Copyright 2014 The Serviced Authors.
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

package container

import (
	"crypto/tls"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/mux"
	"github.com/control-center/serviced/web"
	"github.com/control-center/serviced/zzk/registry"
)

/*
The 'prxy' service implemented here provides both a prxy for outbound
service requests and a multiplexer for inbound requests. The diagram below
illustrates one way proxies interoperate.

      proxy A                   proxy B
      +-----------+             +-----------+
    22250         |     +---->22250 ---------------+
      |           |     |       |           |      |
 +-->3306 --------------+       |           |      |
 +-->4369 --------------+       |           |      |
 |    |           |             |           |      |
 |    +-----------+             +-----------+      |
 |                                                 |
 +----zensvc                    mysql/3306 <-------+
                                rabbitmq/4369 <----+

proxy A exposes MySQL and RabbitMQ ports, 3306 and 4369 respectively, to its
zensvc. When zensvc connects to those ports proxy A forwards the resulting
traffic to the appropriate remote services via the TCPMux port exposed by
proxy B.

Start the service from the command line by typing

prxy [OPTIONS] SERVICE_ID

  -certfile="": path to public certificate file (defaults to compiled in public cert)
  -endpoint="127.0.0.1:4979": serviced endpoint address
  -keyfile="": path to private key file (defaults to compiled in private key)
  -mux=true: enable port multiplexing
  -muxport=22250: multiplexing port to use
  tls is always enabled

To terminate the prxy service connect to it via port 4321 and it will exit.
The netcat (nc) command is particularly useful for this:

    nc 127.0.0.1 4321
*/

// proxyKey is the key mapping for finding the container proxy within
// the cache.
type proxyKey struct {
	Application string
	PortNumber  uint16
}

// ContainerProxyCache stores a mapping of container proxies
type proxyCache struct {
	mu            *sync.Mutex
	cache         map[ContainerProxyKey]*ContainerProxy
	cancel        <-chan struct{}
	useTLS        bool
	useDirectConn bool
}

// NewContainerProxyCache instantiates a new container proxy cache.
func newProxyCache(cancel <-chan struct{}, useTLS, useDirectConn bool) *ContainerProxyCache {
	return &ContainerProxyCache{
		mu:            &sync.Mutex{},
		cache:         make(map[ContainerProxyKey]*ContainerProxy),
		cancel:        cancel,
		useTLS:        useTLS,
		useDirectConn: useDirectConn,
	}
}

// Set updates exports for a particular import, creating a new proxy if it does
// not already exist in the cache.
func (c *proxyCache) Set(application string, portNumber uint16, exports ...registry.ExportDetails) (bool, error) {
	logger := plog.WithFields(logrus.Fields{
		"application": application,
		"portnumber":  portNumber,
	})

	c.mu.Lock()
	defer c.mu.Unlock()

	key := ContainerProxyKey{Application: application, PortNumber: portNumber}
	proxy, ok := c.cache[key]
	if !ok {
		proxy, err := NewContainerProxy(c.cancel, application, portNumber, c.useTLS, c.useDirectConn, exports)
		if err != nil {
			logger.WithError(err).Debug("Could not set up new proxy")
			return false, err
		}
		logger.Debug("Adding a new proxy to the cache")
		c.cache[key] = proxy
		return true, nil
	}

	proxy.SetExports(exports)
	return false, nil
}

// Proxy proxies a collection of exports to a specific port binding.
type Proxy struct {
	exports       web.Exports
	useTLS        bool
	useDirectConn bool
}

// NewProxy opens a port to proxy a group of incoming connections.
func NewProxy(cancel <-chan struct{}, application string, portNumber uint16, useTLS, useDirectConn bool, exports []registry.ExportDetails) (*ContainerProxy, error) {
	logger := plog.WithFields(logrus.Fields{
		"application": application,
		"portnumber":  portNumber,
	})

	logger.Debug("Opening port for proxy")
	listener, err := net.Listen("tcp4", fmt.Sprintf(":%d", portNumber))
	if err != nil {
		logger.WithError(err).Debug("Could not open port")
		return nil, err
	}

	p := &ContainerProxy{
		exports:       web.NewRoundRobinExports(exports),
		useTLS:        useTLS,
		useDirectConn: useDirectConn,
	}

	go p.serve(cancel, application, portNumber, listener)
	return p, nil
}

// SetExports updates the list of exports to proxy.
func (p *Proxy) SetExports(exports []registry.ExportDetails) {
	p.exports.Set(exports)
}

// serve manages requests sent on a given port
func (p *Proxy) serve(cancel <-chan struct{}, application string, portNumber uint16, listener net.Listener) {
	logger := plog.WithFields(logrus.Fields{
		"application": application,
		"portnumber":  portNumber,
	})

	// Pass along incoming connections
	ch := make(chan net.Conn)
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// Check if the reason the listener failed was because we are
				// shutting down.  If so, then just exit nicely.
				select {
				case <-cancel:
					logger.WithError(err).Debug("Could not accept incoming connection")
					return
				case <-time.After(time.Second):
					logger.WithError(err).Fatal("Could not accept incoming connection")
				}
			}
			select {
			case ch <- conn:
			case <-cancel:
				return
			}
		}
	}()

	// Proxy local connections to remote endpoints
	for {
		select {
		case local := <-ch:
			if export := p.exports.Next(); export != nil {
				remote, err := p.getRemoteConnection(cancel, export)
				if err != nil {
					logger.WithError(err).Error("Could not establish remote connection")
					local.Close()
					continue
				}
				proxy(remote, local)
				select {
				case <-cancel:
					return
				default:
				}
			} else {
				logger.Warn("No remote services available for proxying")
				local.Close()
			}
		case <-cancel:
			return
		}
	}
}

// getRemoteConnection opens an outbound connection on the given export.
func (p *Proxy) getRemoteConnection(cancel <-chan struct{}, export *registry.ExportDetails) (net.Conn, error) {
	logger := plog.WithFields(logrus.Fields{
		"application": export.Application,
		"hostip":      export.HostIP,
		"muxport":     export.MuxPort,
		"privateip":   export.PrivateIP,
		"privateport": export.PortNumber,
	})

	// establish the local address
	localAddress := fmt.Sprintf("%s:%d", export.PrivateIP, export.PortNumber)

	if p.useDirectConn {
		// check if the host for the container is running on the same host
		if isLocalAddress(export.HostIP) {

			// don't proxy localhost address, we'll end up in a loop
			if !strings.HasPrefix(export.HostIP, "127") && export.HostIP != "localhost" {

				// return the connection if the target is local
				logger.WithField("address", localAddress).Debug("Dialing a local connection")
				return net.Dial("tcp4", localAddress)
			}
		}
	}

	// establish the remote address
	if export.MuxPort == 0 {
		logger.Warn("Mux port is unspecified, using default of 22250")
		export.MuxPort = 22250
	}
	remoteAddress := fmt.Sprintf("%s:%d", export.HostIP, export.MuxPort)

	logger = logger.WithFields(logrus.Fields{
		"address":    remoteAddress,
		"tlsenabled": p.useTLS,
	})

	// build the header
	header, err = mux.NewHeader(localAddress)
	if err != nil {
		logger.WithError(err).Debug("Could not build header")
		return nil, err
	}

	// TODO: send auth to connection
	logger.Debug("Dialing a remote connection")
	conn, err := mux.Dial("tcp4", remoteAddress, header, nil)
	if err != nil {
		return nil, err
	}
	if p.useTLS {
		conn = tls.Client(conn, &tls.Config{InsecureSkipVerify: true})
	}
	return conn, err
}
