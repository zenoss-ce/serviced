// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/rpc/rpcutils"
)

// Initializer for serviced server
func (c *ServicedCli) initServer() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "server",
		Usage:       "Starts serviced",
		Description: "serviced server",
		Action:      c.cmdServer,
	})
}

// serviced server
func (c *ServicedCli) cmdServer(ctx *cli.Context) {
	master := api.GetOptionsMaster()
	agent := api.GetOptionsAgent()

	// Make sure one of the configurations was specified
	if !master && !agent {
		fmt.Fprintf(os.Stderr, "serviced cannot be started: no mode (master or agent) was specified\n")
		return
	}

	if master {
		fmt.Println("This master has been configured to be in pool: " + api.GetOptionsMasterPoolID())
	}

	// Start server mode
	rpcutils.RPC_CLIENT_SIZE = api.GetOptionsMaxRPCClients()
	c.driver.StartServer()
}