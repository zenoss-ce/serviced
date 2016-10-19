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
	"math/rand"
	"sync"
	"time"

	"github.com/control-center/serviced/utils"
)

var ipmap = make(map[string]struct{})

func init() {
	rand.Seed(time.Now().Unix())

	ips, err := utils.GetIPv4Addresses()
	if err != nil {
		panic(err)
	}
	for _, ip := range ips {
		ipmap[ip] = struct{}{}
	}
}

// IsLocalAddress returns true if the ip address is available on the host
func IsLocalAddress(ip string) (ok bool) {
	_, ok = ipmap[ip]
	return
}

// Addr returns the local and remote addresses to connect
type Addr interface {
	MuxIP() string
	MuxAddr() string
	LocalAddr() string
}

// Addrs manages a list of available exports
type Addrs interface {
	Set(addrs []Address)
	Next() Address
}

// RoundRobinAddrs returns the next export using a round-robin strategy
type RoundRobinAddrs struct {
	mu       *sync.Mutex
	xid      int
	addrs    []Addr
	forceMux bool
}

// NewRoundRobinAddrs instantiates a new round-robin export strategy
func NewRoundRobinAddrs(addrs []Addr, forceMux bool) *RoundRobinAddrs {
	e := &RoundRobinAddrs{
		mu: &sync.Mutex{},
	}
	e.set(addr)
	return e
}

// Set updates the list of available addresses
func (e *RoundRobinAddrs) Set(addrs []Addr) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.set(addrs)
}

// set updates the export list, but first randomizes the order and resets the
// counter.
func (e *RoundRobinAddrs) set(addrs []Addr) {
	// reset the counter
	e.xid = 0

	// randomize the exports
	e.addrs = make([]Addr, len(addrs))
	for i, j := range rand.Perm(len(addrs)) {
		e.addrs[i] = addrs[j]
	}
}

// Next returns the next available export
func (e *RoundRobinExport) Next() (addr Addr) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// make sure there is data to submit
	if size := len(e.addrs); size > 0 {
		addr = e.addrs[e.xid]
		e.xid = (e.xid + 1) % size
	}
	return
}
