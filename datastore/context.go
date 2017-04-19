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

package datastore

import (
	"github.com/control-center/serviced/metrics"
	"os/user"
	"fmt"
	"os"
)

// Context is the context of the application or request being made
type Context interface {
	// Get a connection to the datastore
	Connection() (Connection, error)

	// Get the Metrics object from the context
	Metrics() *metrics.Metrics
	GetUser() string
	SetUser(string)
	GetIntention() string
	SetIntention(string)
	GetOrigin() string
	SetOrigin(string)
}

var savedDriver Driver

//Register a driver to use for the context
func Register(driver Driver) {
	savedDriver = driver
	ctx = newCtx(driver)
}

//Get returns the global Context
func Get() Context {
	return ctx
}

// GetNew() returns a new global context.
// This function is not intended for production use, but is for the purpose
// of getting fresh contexts for performance testing with metrics for troubleshooting.
func GetNew() Context {
	return newCtx(savedDriver)
}

func GetNewPlus(user string, origin string) {
	ctx = newCtx(savedDriver)
	ctx.SetUser(user)
	ctx.SetOrigin(origin)
}

var ctx Context

//new Creates a new context with a Driver to a datastore
func newCtx(driver Driver) Context {
	return &context{driver, metrics.NewMetrics(), "CONTEXT INTENTION", "Internal",""}
}

type context struct {
	driver  Driver
	metrics *metrics.Metrics
	intention string
	origin string
	user string
	//origin dao.OriginType
}

func (c *context) Connection() (Connection, error) {
	return c.driver.GetConnection()
}

func (c *context) Metrics() *metrics.Metrics {
	return c.metrics
}

func (c *context) SetUser(user string) {
	c.user = user
}

func (c *context) GetUser() string {
	if c.user != "" {
		return c.user
	}
	cu, err := user.Current()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting current user: %s\n", err)
		return "UNKNOWN_USER"
	}
	return cu.Name
}

func (c * context) SetIntention(intention string) {
	c.intention = intention
}

func (c * context) GetIntention() string {
	return c.intention
}

func (c * context) SetOrigin(origin string) {
	c.origin = origin
}

func (c * context) GetOrigin() string {
	return c.origin
}