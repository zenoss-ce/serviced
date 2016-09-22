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

package master

import (
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/facade"
)

// NewServer creates a new serviced master rpc server
func NewServer(f *facade.Facade, tokenExpiration time.Duration) *Server {
	return &Server{f, tokenExpiration}
}

// Server is the RPC type for the master(s)
type Server struct {
	f          *facade.Facade
	expiration time.Duration
}

func (s *Server) context() datastore.Context {
	//here in case we ever need to create a per request context
	return datastore.Get()
}
