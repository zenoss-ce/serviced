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

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/domain/host"

	"errors"
)

// GetHost gets the host
func (s *Server) GetHost(hostID string, reply *host.Host) error {
	response, err := s.f.GetHost(s.context(), hostID)
	if err != nil {
		return err
	}
	if response == nil {
		return errors.New("hosts_server.go host not found")
	}
	*reply = *response
	return nil
}

// GetHosts returns all Hosts
func (s *Server) GetHosts(empty struct{}, hostReply *[]host.Host) error {
	hosts, err := s.f.GetHosts(s.context())
	if err != nil {
		return err
	}
	*hostReply = hosts
	return nil
}

// GetActiveHosts returns all active host ids
func (s *Server) GetActiveHostIDs(empty struct{}, hostReply *[]string) error {
	hosts, err := s.f.GetActiveHostIDs(s.context())
	if err != nil {
		return err
	}
	*hostReply = hosts
	return nil
}

// AddHost adds the host
func (s *Server) AddHost(host host.Host, hostReply *[]byte) error {
	privateKey, err := s.f.AddHost(s.context(), &host)
	if err != nil {
		return err
	}
	*hostReply = privateKey
	return nil
}

// UpdateHost updates the host
func (s *Server) UpdateHost(host host.Host, _ *struct{}) error {
	return s.f.UpdateHost(s.context(), &host)
}

// RemoveHost removes the host
func (s *Server) RemoveHost(hostID string, _ *struct{}) error {
	return s.f.RemoveHost(s.context(), hostID)
}

// FindHostsInPool  Returns all Hosts in a pool
func (s *Server) FindHostsInPool(poolID string, hostReply *[]host.Host) error {
	hosts, err := s.f.FindHostsInPool(s.context(), poolID)
	if err != nil {
		return err
	}
	*hostReply = hosts
	return nil
}

type HostAuthenticationRequest struct {
	HostID    string
	Timestamp int64
	Signature []byte
}

type HostAuthenticationResponse struct {
	Token   string
	Expires int64
}

func (s *Server) AuthenticateHost(req *HostAuthenticationRequest, token *string) error {
	// TODO: Verify the request
	host, err := s.f.GetHost(s.context(), req.HostID)
	if err != nil {
		return err
	}
	signed, err := auth.CreateJWTIdentity(host.ID, host.PoolID, true, true,
		auth.DevPubKeyPEM, time.Duration(10*time.Minute), auth.DevPrivKeyPEM)
	if err != nil {
		return err
	}
	*token = signed
	return nil
}
