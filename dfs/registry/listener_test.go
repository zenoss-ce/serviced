// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build integration

package registry

import (
	"errors"
	"path"
	"testing"
	"time"

	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/dfs/docker/mocks"
	"github.com/control-center/serviced/domain/registry"
	dockerclient "github.com/fsouza/go-dockerclient"

	. "gopkg.in/check.v1"
)

type testImage struct {
	Image *registry.Image
}

func (i *testImage) ID() string {
	return i.Image.ID()
}

func (i *testImage) Path() string {
	return path.Join(zkregistrypath, i.ID())
}

func (i *testImage) Address(host string) string {
	return path.Join(host, i.Image.String())
}

func (i *testImage) Create(c *C, conn coordclient.Connection) *RegistryImageNode {
	node := &RegistryImageNode{Image: i.Image, PushedAt: time.Unix(0, 0)}
	c.Logf("Creating node at %s: %v", i.Path(), *i.Image)
	err := conn.Create(i.Path(), node)
	c.Assert(err, IsNil)
	exists, err := conn.Exists(i.Path())
	c.Assert(exists, Equals, true)
	return node
}

func (i *testImage) Update(c *C, conn coordclient.Connection, node *RegistryImageNode) {
	err := conn.Set(i.Path(), node)
	c.Assert(err, IsNil)
}

func (i *testImage) GetW(c *C, conn coordclient.Connection) (<-chan coordclient.Event, *RegistryImageNode) {
	node := &RegistryImageNode{}
	evt, err := conn.GetW(i.Path(), node)
	c.Assert(err, IsNil)
	return evt, node
}

func TestRegistryListener(t *testing.T) { TestingT(t) }

type RegistryListenerSuite struct {
	dc       *dockerclient.Client
	conn     coordclient.Connection
	docker   *mocks.Docker
	listener *RegistryListener
	zkCtrID  string
}

var _ = Suite(&RegistryListenerSuite{})

func (s *RegistryListenerSuite) SetUpSuite(c *C) {
	var err error
	if s.dc, err = dockerclient.NewClient(docker.DefaultSocket); err != nil {
		c.Fatalf("Could not connect to docker client: %s", err)
	}
	if ctr, err := s.dc.InspectContainer("zktestserver"); err == nil {
		s.dc.KillContainer(dockerclient.KillContainerOptions{ID: ctr.ID})
		opts := dockerclient.RemoveContainerOptions{
			ID:            ctr.ID,
			RemoveVolumes: true,
			Force:         true,
		}
		s.dc.RemoveContainer(opts)
	} else {
		opts := dockerclient.PullImageOptions{
			Repository: "jplock/zookeeper",
			Tag:        "3.4.6",
		}
		auth := dockerclient.AuthConfiguration{}
		s.dc.PullImage(opts, auth)
	}
	// Start zookeeper
	opts := dockerclient.CreateContainerOptions{Name: "zktestserver"}
	opts.Config = &dockerclient.Config{Image: "jplock/zookeeper:3.4.6"}
	ctr, err := s.dc.CreateContainer(opts)
	if err != nil {
		c.Fatalf("Could not initialize zookeeper: %s", err)
	}
	s.zkCtrID = ctr.ID
	hconf := &dockerclient.HostConfig{
		PortBindings: map[dockerclient.Port][]dockerclient.PortBinding{
			"2181/tcp": []dockerclient.PortBinding{
				{HostIP: "localhost", HostPort: "2181"},
			},
		},
	}
	if err := s.dc.StartContainer(ctr.ID, hconf); err != nil {
		c.Fatalf("Could not start zookeeper: %s", err)
	}
	// Connect to the zookeeper client
	dsn := zookeeper.NewDSN([]string{"localhost:2181"}, 15*time.Second).String()
	zkclient, err := coordclient.New("zookeeper", dsn, "/", nil)
	if err != nil {
		c.Fatalf("Could not establish the zookeeper client: %s", err)
	}
	s.conn, err = zkclient.GetCustomConnection("/")
	if err != nil {
		c.Fatalf("Could not create a connection to the zookeeper client: %s", err)
	}
}

func (s *RegistryListenerSuite) TearDownSuite(c *C) {
	if s.conn != nil {
		s.conn.Close()
	}
	s.dc.StopContainer(s.zkCtrID, 10)
	opts := dockerclient.RemoveContainerOptions{
		ID:            s.zkCtrID,
		RemoveVolumes: true,
		Force:         true,
	}
	s.dc.RemoveContainer(opts)
}

func (s *RegistryListenerSuite) SetUpTest(c *C) {
	// Initialize the mock docker object
	s.docker = &mocks.Docker{}
	// Initialize the listener
	s.listener = NewRegistryListener(s.docker, "test-server:5000", "test-host", 15*time.Second)
	s.listener.conn = s.conn
	// Create the base path
	s.conn.CreateDir(zkregistrypath)
}

func (s *RegistryListenerSuite) TearDownTest(c *C) {
	s.conn.Delete(zkregistrypath)
}

func (s *RegistryListenerSuite) TestRegistryListener_NoNode(c *C) {
	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, "keynotexists")
	}()
	select {
	case <-time.After(5 * time.Second):
		close(shutdown)
		select {
		case <-time.After(5 * time.Second):
			close(shutdown)
			select {
			case <-time.After(5 * time.Second):
				c.Fatalf("listener did not shutdown within timeout!")
			case <-done:
				c.Errorf("listener timed out waiting to shutdown")
			}
		case <-done:
		}
	}
}

func (s *RegistryListenerSuite) TestRegistryListener_ImagePushed(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	node := rImage.Create(c, s.conn)
	node.PushedAt = time.Now().UTC()
	rImage.Update(c, s.conn, node)
	evt, _ := rImage.GetW(c, s.conn)

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Errorf("listener exited prematurely")
	case <-evt:
		c.Errorf("listener updated node")
	}
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within timeout!")
	case <-done:
	}
}

func (s *RegistryListenerSuite) TestRegistryListener_NoLocalImage(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	_ = rImage.Create(c, s.conn)
	evt, _ := rImage.GetW(c, s.conn)
	s.docker.On("FindImage", rImage.Image.UUID).Return(nil, errors.New("image not found")).Once()

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Fatalf("listener exited prematurely")
	case <-evt:
		c.Errorf("listener updated node")
	}
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_AnotherNodePush(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	node := rImage.Create(c, s.conn)
	s.docker.On("FindImage", rImage.Image.UUID).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil).Once()

	// take lead of the node
	leader := s.conn.NewLeader(rImage.Path(), &RegistryImageLeader{HostID: "master"})
	_, err := leader.TakeLead()
	c.Assert(err, IsNil)
	leaders, cvt, err := s.conn.ChildrenW(rImage.Path())
	c.Assert(err, IsNil)
	c.Assert(leaders, HasLen, 1)

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()

	// verify lead was attempted
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not try to lead")
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-cvt:
		// assert the listener is NOT the leader
		c.Logf("listener is acquiring lead")
		lnode := &RegistryImageLeader{}
		err = leader.Current(lnode)
		c.Assert(err, IsNil)
		c.Assert(lnode.HostID, Equals, "master")
	}

	// "push" the image
	c.Logf("updating push")
	node.PushedAt = time.Now().UTC()
	rImage.Update(c, s.conn, node)
	evt, _ := rImage.GetW(c, s.conn)
	err = leader.ReleaseLead()
	c.Assert(err, IsNil)

	// verify the node was NOT updated
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-evt:
		c.Errorf("listener updated node")
	}

	// verify shutdown
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within the timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_PushFails(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	_ = rImage.Create(c, s.conn)
	evt, _ := rImage.GetW(c, s.conn)
	leader := s.conn.NewLeader(rImage.Path(), &RegistryImageLeader{HostID: "master"})
	_, cvt, err := s.conn.ChildrenW(rImage.Path())
	c.Assert(err, IsNil)
	timeoutC := make(chan time.Time)
	s.docker.On("FindImage", rImage.Image.UUID).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil).Once()
	s.docker.On("TagImage", rImage.Image.UUID, rImage.Address(s.listener.address)).Return(nil).Once()
	s.docker.On("PushImage", rImage.Address(s.listener.address)).Return(errors.New("could not push image")).WaitUntil(timeoutC).Once()

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()

	// verify lead was attempted
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not try to lead")
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-cvt:
		// assert the listener IS the leader
		c.Logf("listener is acquiring lead")
		lnode := &RegistryImageLeader{}
		err = leader.Current(lnode)
		c.Assert(err, IsNil)
		c.Assert(lnode.HostID, Equals, s.listener.hostid)
	case <-evt:
		c.Errorf("listener updated node")
	}

	// verify the node did NOT update
	close(timeoutC)
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Fatalf("listener exited prematurely")
	case <-evt:
		c.Errorf("listener updated node")
	}

	// verify shutdown
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within the timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_LeadDisconnect(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	_ = rImage.Create(c, s.conn)
	evt, _ := rImage.GetW(c, s.conn)
	leader := s.conn.NewLeader(rImage.Path(), &RegistryImageLeader{HostID: "master"})
	_, cvt, err := s.conn.ChildrenW(rImage.Path())
	c.Assert(err, IsNil)
	timeoutC := make(chan time.Time)
	s.docker.On("FindImage", rImage.Image.UUID).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil).Once()
	s.docker.On("TagImage", rImage.Image.UUID, rImage.Address(s.listener.address)).Return(nil).Once()
	s.docker.On("PushImage", rImage.Address(s.listener.address)).Return(nil).WaitUntil(timeoutC).Once()

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()

	// verify lead was attempted
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not try to lead")
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-cvt:
		// assert the listener IS the leader
		c.Logf("listener is acquiring lead")
		lnode := &RegistryImageLeader{}
		err = leader.Current(lnode)
		c.Assert(err, IsNil)
		c.Assert(lnode.HostID, Equals, s.listener.hostid)
	case <-evt:
		c.Errorf("listener updated node")
	}

	// delete the leader
	children, err := s.conn.Children(rImage.Path())
	c.Assert(err, IsNil)
	c.Assert(children, HasLen, 1)
	err = s.conn.Delete(path.Join(rImage.Path(), children[0]))
	c.Assert(err, IsNil)

	// verify the node did NOT update
	close(timeoutC)
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Fatalf("listener exited prematurely")
	case <-evt:
		c.Errorf("listener updated node")
	}

	// verify shutdown
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within the timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}

func (s *RegistryListenerSuite) TestRegistryListener_Success(c *C) {
	rImage := &testImage{
		Image: &registry.Image{
			Library: "libraryname",
			Repo:    "reponame",
			Tag:     "tagname",
			UUID:    "uuidvalue",
		},
	}
	_ = rImage.Create(c, s.conn)
	evt, _ := rImage.GetW(c, s.conn)
	leader := s.conn.NewLeader(rImage.Path(), &RegistryImageLeader{HostID: "master"})
	_, cvt, err := s.conn.ChildrenW(rImage.Path())
	c.Assert(err, IsNil)
	timeoutC := make(chan time.Time)
	s.docker.On("FindImage", rImage.Image.UUID).Return(&dockerclient.Image{ID: rImage.Image.UUID}, nil).Once()
	s.docker.On("TagImage", rImage.Image.UUID, rImage.Address(s.listener.address)).Return(nil).Once()
	s.docker.On("PushImage", rImage.Address(s.listener.address)).Return(nil).WaitUntil(timeoutC).Once()

	shutdown := make(chan interface{})
	done := make(chan struct{})
	go func() {
		defer close(done)
		s.listener.Spawn(shutdown, rImage.ID())
	}()

	// verify lead was attempted
	select {
	case <-time.After(5 * time.Second):
		c.Errorf("listener did not try to lead")
	case <-done:
		c.Fatalf("listener exited prematurely!")
	case <-cvt:
		// assert the listener IS the leader
		c.Logf("listener is acquiring lead")
		lnode := &RegistryImageLeader{}
		err = leader.Current(lnode)
		c.Assert(err, IsNil)
		c.Assert(lnode.HostID, Equals, s.listener.hostid)
	case <-evt:
		c.Errorf("listener updated node")
	}

	// verify the node DID update
	close(timeoutC)
	select {
	case <-time.After(5 * time.Second):
	case <-done:
		c.Fatalf("listener exited prematurely")
	case <-evt:
	}

	// verify shutdown
	close(shutdown)
	select {
	case <-time.After(5 * time.Second):
		c.Fatalf("listener did not shutdown within the timeout!")
	case <-done:
	}
	s.docker.AssertExpectations(c)
}
