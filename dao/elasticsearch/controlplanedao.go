// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package elasticsearch

import (
	"github.com/mattbaird/elastigo/api"
	"github.com/zenoss/glog"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/dfs"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/zzk"
	zkdocker "github.com/zenoss/serviced/zzk/docker"

	"fmt"
	"strconv"
)

const (
	DOCKER_ENDPOINT string = "unix:///var/run/docker.sock"
)

//assert interface
var _ dao.ControlPlane = &ControlPlaneDao{}

type ControlPlaneDao struct {
	hostName string
	port     int
	varpath  string
	vfs      string
	zclient  *coordclient.Client
	zkDao    *zzk.ZkDao
	dfs      *dfs.DistributedFileSystem
	//needed while we move things over
	facade         *facade.Facade
	dockerRegistry string
}

func (this *ControlPlaneDao) Action(request dao.AttachRequest, unused *int) error {
	// Get the service and update the request
	var svc service.Service
	if err := this.GetService(request.Running.ServiceID, &svc); err != nil {
		return err
	}

	var command []string
	if request.Command == "" {
		return fmt.Errorf("missing command")
	}

	action, ok := svc.Actions[request.Command]
	if !ok {
		return fmt.Errorf("action not found for service %s: %s", svc.Id, request.Command)
	}

	command = append([]string{action}, request.Args...)
	req := zkdocker.Action{
		HostID:   request.Running.HostID,
		DockerID: request.Running.DockerID,
		Command:  command,
	}

	conn, err := this.zclient.GetConnection()
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = zkdocker.SendAction(conn, &req)
	return err
}

func (this *ControlPlaneDao) RestartService(serviceID string, unused *int) error {
	return dao.ControlPlaneError{"unimplemented"}
}

// Create a elastic search control plane data access object
func NewControlPlaneDao(hostName string, port int, facade *facade.Facade, dockerRegistry string) (*ControlPlaneDao, error) {
	glog.V(0).Infof("Opening ElasticSearch ControlPlane Dao: hostName=%s, port=%d", hostName, port)
	api.Domain = hostName
	api.Port = strconv.Itoa(port)

	dao := &ControlPlaneDao{
		hostName:       hostName,
		port:           port,
		dockerRegistry: dockerRegistry,
	}
	if dfs, err := dfs.NewDistributedFileSystem(dao, facade); err != nil {
		return nil, err
	} else {
		dao.dfs = dfs
	}

	return dao, nil
}

func NewControlSvc(hostName string, port int, facade *facade.Facade, zclient *coordclient.Client, varpath, vfs string, dockerRegistry string) (*ControlPlaneDao, error) {
	glog.V(2).Info("calling NewControlSvc()")
	defer glog.V(2).Info("leaving NewControlSvc()")

	s, err := NewControlPlaneDao(hostName, port, facade, dockerRegistry)
	if err != nil {
		return nil, err
	}

	//Used to bridge old to new
	s.facade = facade

	s.varpath = varpath
	s.vfs = vfs

	s.zclient = zclient
	s.zkDao = zzk.NewZkDao(zclient)

	// create the account credentials
	if err = createSystemUser(s); err != nil {
		return nil, err
	}

	return s, nil
}
