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

package elasticsearch

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	coordclient "github.com/control-center/serviced/coordinator/client"
	coordzk "github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	userdomain "github.com/control-center/serviced/domain/user"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/utils"
	_ "github.com/control-center/serviced/volume"
	_ "github.com/control-center/serviced/volume/btrfs"
	_ "github.com/control-center/serviced/volume/rsync"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
	. "gopkg.in/check.v1"
)

const (
	HOSTID = "deadbeef"
)

var unused int
var unusedStr string
var id string
var addresses []string
var version datastore.VersionedEntity

var err error

// MockStorageDriver is an interface that mock the storage subsystem
type MockStorageDriver struct {
	exportPath string
}

func (m MockStorageDriver) ExportPath() string {
	return m.exportPath
}

func (m MockStorageDriver) SetClients(clients ...string) {
}

func (m MockStorageDriver) Sync() error {
	return nil
}

func (m MockStorageDriver) Restart() error {
	return nil
}

func (m MockStorageDriver) Stop() error {
	return nil
}

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

//Instantiate the gocheck suite. Initialize the DaoTest and the embedded FacadeTest
var _ = Suite(&DaoTest{})

//DaoTest gocheck test type for setting up isvcs and other resources needed by tests
type DaoTest struct {
	facade.FacadeTest
	Dao    *ControlPlaneDao
	zkConn coordclient.Connection
}

//SetUpSuite is run before the tests to ensure elastic, zookeeper etc. are running.
func (dt *DaoTest) SetUpSuite(c *C) {
	docker.SetUseRegistry(true)

	dt.Port = 9202
	isvcs.Init()
	isvcs.Mgr.SetVolumesDir("/tmp/serviced-test")
	esServicedClusterName, _ := utils.NewUUID36()
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-serviced", "cluster", esServicedClusterName); err != nil {
		c.Fatalf("Could not set elasticsearch-serviced clustername: %s", err)
	}
	esLogstashClusterName, _ := utils.NewUUID36()
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-logstash", "cluster", esLogstashClusterName); err != nil {
		c.Fatalf("Could not set elasticsearch-logstash clustername: %s", err)
	}
	isvcs.Mgr.Wipe()
	if err := isvcs.Mgr.Start(); err != nil {
		c.Fatalf("Could not start es container: %s", err)
	}
	dt.MappingsFile = "controlplane.json"
	dt.FacadeTest.SetUpSuite(c)

	dsn := coordzk.NewDSN([]string{"127.0.0.1:2181"}, time.Second*15).String()
	glog.Infof("zookeeper dsn: %s", dsn)
	zClient, err := coordclient.New("zookeeper", dsn, "", nil)
	if err != nil {
		glog.Fatalf("Could not start es container: %s", err)
	}

	zzk.InitializeLocalClient(zClient)

	dt.zkConn, err = zzk.GetLocalConnection("/")
	if err != nil {
		c.Fatalf("could not get zk connection %v", err)
	}

	dt.Dao, err = NewControlSvc("localhost", int(dt.Port), dt.Facade, "/tmp", "rsync", 4979, time.Minute*5, "localhost:5000", MockStorageDriver{})
	if err != nil {
		glog.Fatalf("Could not start es container: %s", err)
	} else {
		for i := 0; i < 10; i += 1 {
			id := strconv.Itoa(i)
			dt.Dao.RemoveService(id, &unused)
		}
		for i := 100; i < 110; i += 1 {
			id := strconv.Itoa(i)
			dt.Dao.RemoveService(id, &unused)
		}
	}
}

//SetUpTest run before each test.
func (dt *DaoTest) SetUpTest(c *C) {
	//Facade tests delete the contents of the database for every test
	dt.FacadeTest.SetUpTest(c)
	//DAO tests expect default pool and system user

	if err := dt.Facade.CreateDefaultPool(dt.CTX, "default"); err != nil {
		c.Fatalf("could not create default pool: %s", err)
	}

	// create the account credentials
	if err := createSystemUser(dt.Dao); err != nil {
		c.Fatalf("could not create systemuser: %s", err)
	}
}

//TearDownSuite stops all isvcs
func (dt *DaoTest) TearDownSuite(c *C) {
	isvcs.Mgr.Stop()
	dt.FacadeTest.TearDownSuite(c)
}

func (dt *DaoTest) TestDao_ValidateEndpoints(t *C) {
	var id string

	// These tests were originally added wtih https://github.com/control-center/serviced/pull/1787,
	// but were skipped for CC 1.1 by https://github.com/control-center/serviced/pull/1811
	t.Skip("this test needs to be restored once we validate endpoints for service add, update, etc")

	// test 1: add tenant with dup ep
	svc := service.Service{
		ID:           "test_tenant",
		Name:         "test_tenant",
		PoolID:       "default",
		Launch:       "auto",
		DeploymentID: "deployment_id",
		Endpoints: []service.ServiceEndpoint{
			{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}},
			{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}},
		},
	}
	err := dt.Dao.AddService(svc, &id)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 2: success
	svc.Endpoints = []service.ServiceEndpoint{
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}},
	}
	err = dt.Dao.AddService(svc, &id)
	t.Assert(err, IsNil)

	// test 3: update service with dup ep
	svc.Endpoints = []service.ServiceEndpoint{
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}},
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}},
	}
	err = dt.Dao.UpdateService(svc, &unused)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 4: add child service with dup ep
	svc2 := service.Service{
		ID:              "test_service_1",
		Name:            "test_service_1",
		ParentServiceID: svc.ID,
		PoolID:          svc.PoolID,
		Launch:          svc.Launch,
		DeploymentID:    svc.DeploymentID,
		Endpoints: []service.ServiceEndpoint{
			{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}},
		},
	}
	err = dt.Dao.AddService(svc2, &id)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 5: add child success
	svc2.Endpoints = []service.ServiceEndpoint{
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}},
	}
	err = dt.Dao.AddService(svc2, &id)
	t.Assert(err, IsNil)

	// test 6: update parent service with dup id on child
	svc.Endpoints = []service.ServiceEndpoint{
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}},
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}},
	}
	err = dt.Dao.UpdateService(svc, &unused)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 7: update parent service success
	svc.Endpoints = []service.ServiceEndpoint{
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}},
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_3", Application: "test_ep_3", Purpose: "export"}},
	}
	err = dt.Dao.UpdateService(svc, &unused)
	t.Assert(err, IsNil)

	// test 8: update child service with dup id on parent
	svc2.Endpoints = []service.ServiceEndpoint{
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}},
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}},
	}
	err = dt.Dao.UpdateService(svc2, &unused)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 9: update child service success
	svc2.Endpoints = []service.ServiceEndpoint{
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}},
		{EndpointDefinition: servicedefinition.EndpointDefinition{Name: "test_ep_4", Application: "test_ep_4", Purpose: "export"}},
	}
	err = dt.Dao.UpdateService(svc2, &unused)
	t.Assert(err, IsNil)
}

func (dt *DaoTest) TestDao_NewService(t *C) {
	svc := service.Service{}
	err := dt.Dao.AddService(svc, &id)
	if err == nil {
		t.Errorf("Expected failure to create service %-v", svc)
		t.Fail()
	}

	svc.ID = "default"
	svc.Name = "default"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	err = dt.Dao.AddService(svc, &id)
	if err != nil {
		t.Errorf("Failure creating service %-v with error: %s", svc, err)
		t.Fail()
	}

	err = dt.Dao.AddService(svc, &id)
	if err == nil {
		t.Errorf("Expected error creating redundant service %-v", svc)
		t.Fail()
	}

	svc.ID = ""
	err = dt.Dao.AddService(svc, &id)
	if err == nil {
		t.Errorf("Expected error creating service with same name and parent: %-v", svc)
		t.Fail()
	}
}

func (dt *DaoTest) TestDao_UpdateService(t *C) {
	dt.Dao.RemoveService("default", &unused)

	svc, _ := service.NewService()
	svc.ID = "default0"
	svc.Name = "default0"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	svc.Name = "name"
	err = dt.Dao.UpdateService(*svc, &unused)
	if err != nil {
		t.Errorf("Failure updating service %-v with error: %s", svc, err)
		t.Fail()
	}

	result := service.Service{}
	dt.Dao.GetService("default0", &result)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result.UpdatedAt = svc.UpdatedAt
	result.CreatedAt = svc.CreatedAt
	if !svc.Equals(&result) {
		t.Errorf("Expected Service %+v, Actual Service %+v", result, *svc)
		t.Fail()
	}

	svc, _ = service.NewService()
	svc.ID = "default1"
	svc.Name = "default1"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	err = dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	svc.Name = "name"
	err = dt.Dao.UpdateService(*svc, &unused)
	if err == nil {
		t.Errorf("Expected error updating service with same name and parent: %-v", svc)
		t.Fail()
	}
}
func (dt *DaoTest) TestDao_UpdateServiceWithConfigFile(t *C) {
	svc, _ := service.NewService()
	svc.ID = "default"
	svc.Name = "default0"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"

	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)
	//Conf file update shouldn't occur because original service didn't have conf files
	confFile := servicedefinition.ConfigFile{Content: "Test content", Filename: "testname"}
	svc.ConfigFiles = map[string]servicedefinition.ConfigFile{"testname": confFile}
	err = dt.Dao.UpdateService(*svc, &unused)
	t.Assert(err, IsNil)

	result := service.Service{}
	dt.Dao.GetService("default", &result)
	t.Assert(0, Equals, len(result.ConfigFiles))

	//test update conf file works
	svc, _ = service.NewService()
	svc.ID = "default_conf"
	svc.Name = "default1"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	svc.OriginalConfigs = map[string]servicedefinition.ConfigFile{"testname": confFile}

	err = dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)
	dt.Dao.GetService("default_conf", &result)
	t.Assert(1, Equals, len(result.ConfigFiles))
	t.Assert(1, Equals, len(result.OriginalConfigs))
	t.Assert(result.ConfigFiles, DeepEquals, svc.OriginalConfigs)
	t.Assert(result.ConfigFiles, DeepEquals, result.OriginalConfigs)

	confFile2 := servicedefinition.ConfigFile{Content: "Test content 2", Filename: "testname"}
	svc.ConfigFiles = map[string]servicedefinition.ConfigFile{"testname": confFile2}
	err = dt.Dao.UpdateService(*svc, &unused)
	t.Assert(err, IsNil)
	dt.Dao.GetService("default_conf", &result)
	t.Assert(1, Equals, len(result.ConfigFiles))
	t.Assert(result.ConfigFiles["testname"], DeepEquals, confFile2)
	t.Assert(result.ConfigFiles, Not(DeepEquals), result.OriginalConfigs)

	//now delete service and re-add, it should have previous modified config file
	err = dt.Dao.RemoveService(svc.ID, &unused)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)
	dt.Dao.GetService("default_conf", &result)
	t.Assert(1, Equals, len(result.ConfigFiles))
	t.Assert(result.ConfigFiles["testname"], DeepEquals, confFile2)
	t.Assert(result.ConfigFiles, Not(DeepEquals), result.OriginalConfigs)

}

func (dt *DaoTest) TestDao_MigrateService(t *C) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	newDescription := "New Description"
	scriptExitCode := 0
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: dt.getMigrationScript(newDescription, scriptExitCode),
		DryRun:     false,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err != nil {
		t.Errorf("Failure migrating service %-v with error: %s", svc, err)
		t.Fail()
	}

	dt.assertServiceMigrated(t, svc, newDescription)
}

func (dt *DaoTest) TestDao_MigrateServiceClone(t *C) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	scriptBody := `
import json, os, string, sys, copy

inputFile = os.environ["MIGRATE_INPUTFILE"]
outputFile = os.environ["MIGRATE_OUTPUTFILE"]
svcs = json.loads(open(inputFile, 'r').read())
clone = copy.deepcopy(svcs[0])
clone["Name"] = "clone name"
export = filter(lambda x: x["Purpose"] == "export", clone["Endpoints"])[0]
export["Application"] = "clone-application"
wrapper = {
	"Modified": svcs,
	"Added": [clone],
	"Deploy": []
}

f = open(outputFile, 'w')
f.write(json.dumps(wrapper, indent=4, sort_keys=True))
f.close()
exit(0)
`
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: scriptBody,
		DryRun:     false,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err != nil {
		t.Errorf("Failure migrating service %-v with error: %s", svc, err)
		t.Fail()
	}

	var svcs []service.Service
	var serviceRequest dao.ServiceRequest
	dt.Dao.GetServices(serviceRequest, &svcs)
	for _, svc := range svcs {
		if svc.Name == "clone name" {
			if svc.Endpoints[0].Application != "clone-application" {
				t.Error("Service endpoint application name was not altered as intended.")
				t.Fail()
			}
			return
		}
	}
	t.Error("Failed to clone service during migration.")
	t.Fail()
}

func (dt *DaoTest) TestDao_MigrateServiceCloneAndMigrate(t *C) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	scriptBody := `
import json, os, string, sys, copy

inputFile = os.environ["MIGRATE_INPUTFILE"]
outputFile = os.environ["MIGRATE_OUTPUTFILE"]
svcs = json.loads(open(inputFile, 'r').read())
clone = copy.deepcopy(svcs[0])
clone["Name"] = "clone name"
export = filter(lambda x: x["Purpose"] == "export", clone["Endpoints"])[0]
export["Application"] = "clone-application"
svcs[0]["Name"] = "migrated name"
wrapper = {
	"Modified": svcs,
	"Added": [clone],
	"Deploy": []
}

f = open(outputFile, 'w')
f.write(json.dumps(wrapper, indent=4, sort_keys=True))
f.close()
exit(0)
`
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: scriptBody,
		DryRun:     false,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err != nil {
		t.Errorf("Failure migrating service %-v with error: %s", svc, err)
		t.Fail()
	}

	var svcs []service.Service
	var serviceRequest dao.ServiceRequest
	dt.Dao.GetServices(serviceRequest, &svcs)
	clonePassed := false
	migratePassed := false
	for _, svc := range svcs {
		if svc.Name == "clone name" {
			if svc.Endpoints[0].Application != "clone-application" {
				t.Error("Service endpoint application name was not altered as intended.")
				t.Fail()
			}
			clonePassed = true
		} else if svc.Name == "migrated name" {
			migratePassed = true
		}
	}
	if !clonePassed || !migratePassed {
		t.Error("Failed to add and migrate service.")
		t.Fail()
	}
}

func (dt *DaoTest) TestDao_MigrateServiceDeploy(t *C) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	scriptBody := `
import json, os, string, sys, copy

inputFile = os.environ["MIGRATE_INPUTFILE"]
outputFile = os.environ["MIGRATE_OUTPUTFILE"]
svcs = json.loads(open(inputFile, 'r').read())
wrapper = {
	"Modified": svcs,
	"Added": [],
	"Deploy": [{
		"ParentID": "migrationTest",
		"Service": {
			"Name": "added-service-name",
	        "ImageID": "ubuntu:latest",
	        "Launch": "auto"
		}
	}]
}

f = open(outputFile, 'w')
f.write(json.dumps(wrapper, indent=4, sort_keys=True))
f.close()
exit(0)
`
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: scriptBody,
		DryRun:     false,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err != nil {
		t.Errorf("Failure migrating service %-v with error: %s", svc, err)
		t.Fail()
	}

	var svcs []service.Service
	var serviceRequest dao.ServiceRequest
	dt.Dao.GetServices(serviceRequest, &svcs)
	for _, svc := range svcs {
		if svc.Name == "added-service-name" {
			if svc.ParentServiceID != "migrationTest" {
				t.Error("Failed to correctly set parent service id when deploying service during migration.")
				t.Fail()
			}
			return
		}
	}
	t.Error("Failed to deploy service during migration.")
	t.Fail()
}

func (dt *DaoTest) TestDao_MigrateServiceCloneFailDupeName(t *C) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	scriptBody := `
import json, os, string, sys, copy

inputFile = os.environ["MIGRATE_INPUTFILE"]
outputFile = os.environ["MIGRATE_OUTPUTFILE"]
svcs = json.loads(open(inputFile, 'r').read())
clone = copy.deepcopy(svcs[0])
export = filter(lambda x: x["Purpose"] == "export", clone["Endpoints"])[0]
export["Application"] = "clone-application"
wrapper = {
	"Modified": svcs,
	"Added": [clone],
	"Deploy": []
}

f = open(outputFile, 'w')
f.write(json.dumps(wrapper, indent=4, sort_keys=True))
f.close()
exit(0)
`
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: scriptBody,
		DryRun:     false,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err == nil {
		t.Errorf("Expected error cloning service during migration with unchanged name.")
		t.Fail()
	}

	t.Assert(err, ErrorMatches, "ValidationError: Duplicate name detected.*")

}

func (dt *DaoTest) TestDao_MigrateServiceCloneFailDupeEndpoint(t *C) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	scriptBody := `
import json, os, string, sys, copy

inputFile = os.environ["MIGRATE_INPUTFILE"]
outputFile = os.environ["MIGRATE_OUTPUTFILE"]
svcs = json.loads(open(inputFile, 'r').read())
clone = copy.deepcopy(svcs[0])
clone["Name"] = "clone name"
export = filter(lambda x: x["Purpose"] == "export", clone["Endpoints"])[0]
wrapper = {
	"Modified": svcs,
	"Added": [clone],
	"Deploy": []
}

f = open(outputFile, 'w')
f.write(json.dumps(wrapper, indent=4, sort_keys=True))
f.close()
exit(0)
`
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: scriptBody,
		DryRun:     false,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err == nil {
		t.Errorf("Expected error cloning service during migration with unchanged name.")
		t.Fail()
	}

	t.Assert(err, ErrorMatches, "ValidationError: Duplicate Application detected for endpoint.*")

}

func (dt *DaoTest) TestDao_MigrateServiceCloneFailAddService(t *C) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	scriptBody := `
import json, os, string, sys, copy

inputFile = os.environ["MIGRATE_INPUTFILE"]
outputFile = os.environ["MIGRATE_OUTPUTFILE"]
svcs = json.loads(open(inputFile, 'r').read())
clone = copy.deepcopy(svcs[0])
clone["Name"] = "clone name"
export = filter(lambda x: x["Purpose"] == "export", clone["Endpoints"])[0]
export["Application"] = "clone-application"
clone["Launch"] = "bogusState"
wrapper = {
	"Modified": svcs,
	"Added": [clone],
	"Deploy": []
}

f = open(outputFile, 'w')
f.write(json.dumps(wrapper, indent=4, sort_keys=True))
f.close()
exit(0)
`
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: scriptBody,
		DryRun:     false,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err == nil {
		t.Errorf("Expected error cloning service during migration with incorrect desired state.")
		t.Fail()
	} else {
		t.Assert(err.Error(), Equals, "ValidationError: \n   0 -  string bogusState not in [auto manual]")
	}
}

func (dt *DaoTest) TestDao_MigrateServiceWithDryRun(t *C) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	newDescription := "New Description"
	scriptExitCode := 0
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: dt.getMigrationScript(newDescription, scriptExitCode),
		DryRun:     true,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err != nil {
		t.Errorf("Failure migrating service %-v with error: %s", svc, err)
		t.Fail()
	}

	// Dry-run should not update the service Description
	dt.assertServiceNotMigrated(t, svc)
}

func (dt *DaoTest) TestDao_MigrateServiceFailsForInvalidID(t *C) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	request := dao.RunMigrationScriptRequest{
		ServiceID:  "Some Undefined Service",
		ScriptBody: dt.getMigrationScript("unused", 0),
		DryRun:     false,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err == nil {
		t.Errorf("Expected error migrating service with invalid ID")
		t.Fail()
	}

	t.Assert(err, ErrorMatches, "No such entity.*")
	dt.assertServiceNotMigrated(t, svc)
}

func (dt *DaoTest) TestDao_MigrateServiceScriptFails(t *C) {
	dryRun := false
	dt.testMigrationScriptFails(t, dryRun)
}

func (dt *DaoTest) TestDao_MigrateServiceWithDryRunScriptFails(t *C) {
	dryRun := true
	dt.testMigrationScriptFails(t, dryRun)
}

func (dt *DaoTest) testMigrationScriptFails(t *C, dryRun bool) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)

	newDescription := "New Description"
	scriptExitCode := 1
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: dt.getMigrationScript(newDescription, scriptExitCode),
		DryRun:     dryRun,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err == nil {
		t.Errorf("Expected error migrating service with script failure")
		t.Fail()
	}

	t.Assert(err, ErrorMatches, "migration script failed: exit status 1")
	dt.assertServiceNotMigrated(t, svc)
}

func (dt *DaoTest) TestDao_MigrateServiceValidationFails(t *C) {
	dryRun := false
	dt.testMigrationScriptFails(t, dryRun)
}

func (dt *DaoTest) TestDao_MigrateServiceWithDryRunFailsValidation(t *C) {
	dryRun := true
	dt.testMigrationScriptFailsValidation(t, dryRun)
}

func (dt *DaoTest) testMigrationScriptFailsValidation(t *C, dryRun bool) {
	svc, err := dt.setupMigrationTest()
	t.Assert(err, IsNil)
	request := dao.RunMigrationScriptRequest{
		ServiceID:  svc.ID,
		ScriptBody: dt.getInvalidMigrationScript(),
		DryRun:     true,
	}

	err = dt.Dao.RunMigrationScript(request, &unused)
	if err == nil {
		t.Errorf("Expected error migrating service with invalid content")
		t.Fail()
		return
	}

	t.Assert(strings.Contains(err.Error(), "ValidationError"), Equals, true)
	dt.assertServiceNotMigrated(t, svc)
}

func (dt *DaoTest) TestDao_GetService(t *C) {
	svc, _ := service.NewService()
	svc.Name = "testname"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	var result service.Service
	err = dt.Dao.GetService(svc.ID, &result)
	t.Assert(err, IsNil)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result.UpdatedAt = svc.UpdatedAt
	result.CreatedAt = svc.CreatedAt
	if !svc.Equals(&result) {
		t.Errorf("GetService Failed: expected=%+v, actual=%+v", svc, result)
	}
}

func (dt *DaoTest) TestDao_GetServices(t *C) {
	svc, _ := service.NewService()
	svc.ID = "default"
	svc.Name = "name"
	svc.PoolID = "default"
	svc.DeploymentID = "deployment_id"
	svc.Launch = "auto"
	svc.Description = "description"
	svc.Instances = 0

	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	var result []service.Service
	var serviceRequest dao.ServiceRequest
	err = dt.Dao.GetServices(serviceRequest, &result)
	t.Assert(err, IsNil)
	t.Assert(len(result), Equals, 1)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result[0].UpdatedAt = svc.UpdatedAt
	result[0].CreatedAt = svc.CreatedAt
	if !result[0].Equals(svc) {
		t.Errorf("expected [%+v] actual=%+v", *svc, result)
		t.Fail()
	}
}

func (dt *DaoTest) TestStoppingParentStopsChildren(t *C) {
	svc := service.Service{
		ID:             "ParentServiceID",
		Name:           "ParentService",
		Startup:        "/usr/bin/ping -c localhost",
		Description:    "Ping a remote host a fixed number of times",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1, 1},
		ImageID:        "test/pinger",
		PoolID:         "default",
		DeploymentID:   "deployment_id",
		DesiredState:   int(service.SVCRun),
		Launch:         "auto",
		Endpoints:      []service.ServiceEndpoint{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	childService1 := service.Service{
		ID:              "childService1",
		Name:            "childservice1",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
	}
	childService2 := service.Service{
		ID:              "childService2",
		Name:            "childservice2",
		Launch:          "auto",
		PoolID:          "default",
		DeploymentID:    "deployment_id",
		Startup:         "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
	}
	// add a service with a subservice
	id := "ParentServiceID"
	var err error
	if err = dt.Dao.AddService(svc, &id); err != nil {
		glog.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	childService1Id := "childService1"
	childService2Id := "childService2"
	if err = dt.Dao.AddService(childService1, &childService1Id); err != nil {
		glog.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}
	if err = dt.Dao.AddService(childService2, &childService2Id); err != nil {
		glog.Fatalf("Failed Loading Child Service 2: %+v, %s", childService2, err)
	}

	// start the service
	var affected int
	if err = dt.Dao.StartService(dao.ScheduleServiceRequest{id, true}, &affected); err != nil {
		glog.Fatalf("Unable to stop parent service: %+v, %s", svc, err)
	}
	// stop the parent
	if err = dt.Dao.StopService(dao.ScheduleServiceRequest{id, true}, &affected); err != nil {
		glog.Fatalf("Unable to stop parent service: %+v, %s", svc, err)
	}
	// verify the children have all stopped
	var services []service.Service
	var serviceRequest dao.ServiceRequest
	err = dt.Dao.GetServices(serviceRequest, &services)
	for _, subService := range services {
		if subService.DesiredState == int(service.SVCRun) && subService.ParentServiceID == id {
			t.Errorf("Was expecting child services to be stopped %v", subService)
		}
	}
}

func (dt *DaoTest) TestDao_StartService(t *C) {

	s0, _ := service.NewService()
	s0.ID = "0"
	s0.Name = "name0"
	s0.PoolID = "default"
	s0.Launch = "auto"
	s0.DesiredState = int(service.SVCStop)
	s0.DeploymentID = "deployment_id"

	s01, _ := service.NewService()
	s01.ID = "01"
	s01.Name = "name1"
	s01.PoolID = "default"
	s01.Launch = "auto"
	s01.ParentServiceID = "0"
	s01.DesiredState = int(service.SVCStop)
	s01.DeploymentID = "deployment_id"

	s011, _ := service.NewService()
	s011.ID = "011"
	s011.Name = "name2"
	s011.PoolID = "default"
	s011.Launch = "auto"
	s011.ParentServiceID = "01"
	s011.DesiredState = int(service.SVCStop)
	s011.DeploymentID = "deployment_id"

	s02, _ := service.NewService()
	s02.ID = "02"
	s02.Name = "name3"
	s02.DeploymentID = "deployment_id2"

	s02.PoolID = "default"
	s02.Launch = "auto"
	s02.ParentServiceID = "0"
	s02.DesiredState = int(service.SVCStop)

	err := dt.Dao.AddService(*s0, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s01, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s011, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s02, &id)
	t.Assert(err, IsNil)

	var affected int
	if err := dt.Dao.StartService(dao.ScheduleServiceRequest{"0", true}, &affected); err != nil {
		t.Fatalf("could not start services: %v", err)
	}

	svc := service.Service{}
	dt.Dao.GetService("0", &svc)
	if svc.DesiredState != int(service.SVCRun) {
		t.Errorf("Service: 0 not requested to run: %+v", svc)
		t.Fail()
	}

	dt.Dao.GetService("01", &svc)
	if svc.DesiredState != int(service.SVCRun) {
		t.Errorf("Service: 01 not requested to run: %+v", svc)
		t.Fail()
	}

	dt.Dao.GetService("011", &svc)
	if svc.DesiredState != int(service.SVCRun) {
		t.Errorf("Service: 011 not requested to run: %+v", svc)
		t.Fail()
	}

	dt.Dao.GetService("02", &svc)
	if svc.DesiredState != int(service.SVCRun) {
		t.Errorf("Service: 02 not requested to run: %+v", svc)
		t.Fail()
	}
}

func (dt *DaoTest) TestDao_GetTenantId(t *C) {
	var err error
	var tenantId string
	err = dt.Dao.GetTenantId("0", &tenantId)
	if err == nil {
		t.Errorf("Expected failure for getting tenantId for 0")
		t.Fail()
	}

	s0, _ := service.NewService()
	s0.Name = "name0"
	s0.PoolID = "default"
	s0.Launch = "auto"
	s0.ID = "0"
	s0.DeploymentID = "deployment_id"

	s01, _ := service.NewService()
	s01.ID = "01"
	s01.ParentServiceID = "0"
	s01.Name = "name1"
	s01.PoolID = "default"
	s01.Launch = "auto"
	s01.DeploymentID = "deployment_id"

	s011, _ := service.NewService()
	s011.ID = "011"
	s011.ParentServiceID = "01"
	s011.Name = "name2"
	s011.PoolID = "default"
	s011.Launch = "auto"
	s011.DeploymentID = "deployment_id"

	err = dt.Dao.AddService(*s0, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s01, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s011, &id)
	t.Assert(err, IsNil)

	tenantId = ""
	err = dt.Dao.GetTenantId("0", &tenantId)
	if err != nil || tenantId != "0" {
		t.Errorf("Failure getting tenantId for 0, err=%s, tenantId=%s", err, tenantId)
		t.Fail()
	}

	tenantId = ""
	err = dt.Dao.GetTenantId("01", &tenantId)
	if err != nil || tenantId != "0" {
		t.Errorf("Failure getting tenantId for 0, err=%s, tenantId=%s", err, tenantId)
		t.Fail()
	}

	tenantId = ""
	err = dt.Dao.GetTenantId("011", &tenantId)
	if err != nil || tenantId != "0" {
		t.Errorf("Failure getting tenantId for 0, err=%s, tenantId=%s", err, tenantId)
		t.Fail()
	}
}

func (dt *DaoTest) TestDaoAutoAssignIPs(t *C) {
	assignIPsPool := pool.New("assignIPsPoolID")
	fmt.Printf("%s\n", assignIPsPool.ID)
	err := dt.Facade.AddResourcePool(dt.CTX, assignIPsPool)
	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", assignIPsPool, err)
		t.Fail()
	}

	ipAddress1 := "192.168.100.10"
	ipAddress2 := "10.50.9.1"

	assignIPsHostIPResources := []host.HostIPResource{}
	oneHostIPResource := host.HostIPResource{}
	oneHostIPResource.HostID = HOSTID
	oneHostIPResource.IPAddress = ipAddress1
	oneHostIPResource.InterfaceName = "eth0"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)
	oneHostIPResource.HostID = HOSTID
	oneHostIPResource.IPAddress = ipAddress2
	oneHostIPResource.InterfaceName = "eth1"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)

	assignIPsHost, err := host.Build("", "65535", assignIPsPool.ID, "", []string{}...)
	if err != nil {
		t.Fatalf("Error creating host: %v", err)
	}
	assignIPsHost.ID = HOSTID
	assignIPsHost.IPs = assignIPsHostIPResources
	err = dt.Facade.AddHost(dt.CTX, assignIPsHost)
	if err != nil {
		t.Fatalf("Failure creating resource host %-v with error: %s", assignIPsHost, err)
	}

	testService := service.Service{
		ID:           "assignIPsServiceID",
		Name:         "testsvc",
		Launch:       "auto",
		PoolID:       assignIPsPool.ID,
		DeploymentID: "deployment_id",
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Name:        "AssignIPsEndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
					AddressConfig: servicedefinition.AddressResourceConfig{
						Port:     8081,
						Protocol: commons.TCP,
					},
				},
			},
		},
	}

	err = dt.Dao.AddService(testService, &id)
	if err != nil {
		t.Fatalf("Failure creating service %-v with error: %s", testService, err)
	}
	assignmentRequest := dao.AssignmentRequest{testService.ID, "", true}
	err = dt.Dao.AssignIPs(assignmentRequest, nil)
	if err != nil {
		t.Errorf("AssignIPs failed: %v", err)
	}

	assignments := []addressassignment.AddressAssignment{}
	err = dt.Dao.GetServiceAddressAssignments(testService.ID, &assignments)
	if err != nil {
		t.Errorf("GetServiceAddressAssignments failed: %v", err)
	}
	if len(assignments) != 1 {
		t.Errorf("Expected 1 AddressAssignment but found %d", len(assignments))
	}
}

func (dt *DaoTest) TestRemoveAddressAssignment(t *C) {
	//test removing address when not present
	err := dt.Dao.RemoveAddressAssignment("fake", nil)
	if err == nil {
		t.Errorf("Expected error removing address %v", err)
	}
}

func (dt *DaoTest) TestDao_ServiceTemplate(t *C) {
	glog.V(0).Infof("TestDao_AddServiceTemplate started")
	defer glog.V(0).Infof("TestDao_AddServiceTemplate finished")

	var (
		unused     int
		templateId string
		templates  map[string]servicetemplate.ServiceTemplate
	)

	// Clean up old templates...
	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	for id, _ := range templates {
		if e := dt.Dao.RemoveServiceTemplate(id, &unused); e != nil {
			t.Fatalf("Failure removing service template %s with error: %s", id, e)
		}
	}

	template := servicetemplate.ServiceTemplate{
		ID:          "",
		Name:        "test_template",
		Description: "test template",
	}

	if e := dt.Dao.AddServiceTemplate(template, &templateId); e != nil {
		t.Fatalf("Failure adding service template %+v with error: %s", template, e)
	}

	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template. Found %d", len(templates))
	}
	if _, ok := templates[templateId]; !ok {
		t.Fatalf("Expected to find template that was added (%s), but did not.", templateId)
	}
	if templates[templateId].Name != "test_template" {
		t.Fatalf("Expected to find test_template. Found %s", templates[templateId].Name)
	}
	template.ID = templateId
	template.Description = "test_template_modified"
	if e := dt.Dao.UpdateServiceTemplate(template, &unused); e != nil {
		t.Fatalf("Failure updating service template %+v with error: %s", template, e)
	}
	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template. Found %d", len(templates))
	}
	if _, ok := templates[templateId]; !ok {
		t.Fatalf("Expected to find template that was updated (%s), but did not.", templateId)
	}
	if templates[templateId].Name != "test_template" {
		t.Fatalf("Expected to find test_template. Found %s", templates[templateId].Name)
	}
	if templates[templateId].Description != "test_template_modified" {
		t.Fatalf("Expected template to be modified. It hasn't changed!")
	}
	if e := dt.Dao.RemoveServiceTemplate(templateId, &unused); e != nil {
		t.Fatalf("Failure removing service template with error: %s", e)
	}
	time.Sleep(1 * time.Second) // race condition. :(
	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 0 {
		t.Fatalf("Expected zero templates. Found %d", len(templates))
	}
	if e := dt.Dao.UpdateServiceTemplate(template, &unused); e != nil {
		t.Fatalf("Failure updating service template %+v with error: %s", template, e)
	}
	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template. Found %d", len(templates))
	}
	if _, ok := templates[templateId]; !ok {
		t.Fatalf("Expected to find template that was updated (%s), but did not.", templateId)
	}
	if templates[templateId].Name != "test_template" {
		t.Fatalf("Expected to find test_template. Found %s", templates[templateId].Name)
	}
}

func (dt *DaoTest) TestDao_NewSnapshot(t *C) {
	// this is technically not a unit test since it depends on the leader
	// starting a watch for snapshot requests and the code here is time
	// dependent waiting for that leader to start the watch
	t.Skip("TODO: fix this test")

	glog.V(0).Infof("TestDao_NewSnapshot started")
	defer glog.V(0).Infof("TestDao_NewSnapshot finished")

	time.Sleep(2 * time.Second) // wait for Leader to start watching for snapshot requests

	service := service.Service{}
	service.ID = "service-without-quiesce"
	dt.Dao.RemoveService(service.ID, &unused)
	// snapshot should work for services without Snapshot Pause/Resume
	err := dt.Dao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	service.ID = "service1-quiesce"
	dt.Dao.RemoveService(service.ID, &unused)
	service.Snapshot.Pause = fmt.Sprintf("STATE=paused echo %s quiesce $STATE", service.ID)
	service.Snapshot.Resume = fmt.Sprintf("STATE=resumed echo %s quiesce $STATE", service.ID)
	err = dt.Dao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	service.ID = "service2-quiesce"
	dt.Dao.RemoveService(service.ID, &unused)
	service.Snapshot.Pause = fmt.Sprintf("STATE=paused echo %s quiesce $STATE", service.ID)
	service.Snapshot.Resume = fmt.Sprintf("STATE=resumed echo %s quiesce $STATE", service.ID)
	err = dt.Dao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	err = dt.Dao.Snapshot(dao.SnapshotRequest{service.ID, ""}, &id)
	if err != nil {
		t.Fatalf("Failure creating snapshot for service %+v with error: %s", service, err)
	}
	if id == "" {
		t.Fatalf("Failure creating snapshot for service %+v - label is empty", service)
	}
	glog.V(0).Infof("successfully created 1st snapshot with label:%s", id)

	err = dt.Dao.Snapshot(dao.SnapshotRequest{service.ID, ""}, &id)
	if err != nil {
		t.Fatalf("Failure creating snapshot for service %+v with error: %s", service, err)
	}
	if id == "" {
		t.Fatalf("Failure creating snapshot for service %+v - label is empty", service)
	}
	glog.V(0).Infof("successfully created 2nd snapshot with label:%s", id)

	time.Sleep(10 * time.Second)
}

func (dt *DaoTest) TestUser_UserOperations(t *C) {
	user := userdomain.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	id := "Pepe"
	err := dt.Dao.AddUser(user, &id)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}

	newUser := userdomain.User{}
	err = dt.Dao.GetUser("Pepe", &newUser)
	if err != nil {
		t.Fatalf("Failure getting user %s", err)
	}

	// make sure they are the same user
	if newUser.Name != user.Name {
		t.Fatalf("Retrieved an unexpected user %v", newUser)
	}

	// make sure the password was hashed
	if newUser.Password == "Pepe" {
		t.Fatalf("Did not hash the password %+v", user)
	}

	unused := 0
	err = dt.Dao.RemoveUser("Pepe", &unused)
	if err != nil {
		t.Fatalf("Failure removing user %s", err)
	}
}

func (dt *DaoTest) TestUser_ValidateCredentials(t *C) {
	user := userdomain.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	id := "Pepe"
	err := dt.Dao.AddUser(user, &id)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}
	var isValid bool
	attemptUser := userdomain.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	err = dt.Dao.ValidateCredentials(attemptUser, &isValid)

	if err != nil {
		t.Fatalf("Failure authenticating credentials %s", err)
	}

	if !isValid {
		t.Fatalf("Unable to authenticate user credentials")
	}

	unused := 0
	err = dt.Dao.RemoveUser("Pepe", &unused)
	if err != nil {
		t.Fatalf("Failure removing user %s", err)
	}

	// update the user
	user.Password = "pepe2"
	err = dt.Dao.UpdateUser(user, &unused)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}
	attemptUser.Password = "Pepe2"
	// make sure we can validate against the updated credentials
	err = dt.Dao.ValidateCredentials(attemptUser, &isValid)

	if err != nil {
		t.Fatalf("Failure authenticating credentials %s", err)
	}
}

func (dt *DaoTest) setupMigrationTest() (*service.Service, error) {
	svc, _ := service.NewService()
	svc.ID = "migrationTest"
	svc.Name = "migrationTest"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	svc.Endpoints = []service.ServiceEndpoint{
		service.ServiceEndpoint{
			EndpointDefinition: servicedefinition.EndpointDefinition{
				Name:        "endpoint-name",
				Application: "application-name",
				Purpose:     "export",
			},
		},
	}
	return svc, dt.Dao.AddService(*svc, &id)
}

func (dt *DaoTest) getMigrationScript(newDescription string, scriptExitCode int) string {
	scriptTemplate := `
import json, os, string, sys

inputFile = os.environ["MIGRATE_INPUTFILE"]
outputFile = os.environ["MIGRATE_OUTPUTFILE"]
svcs = json.loads(open(inputFile, 'r').read())

svcs[0]["Description"] = "%s"

wrapper = {
	"Modified": svcs,
	"Cloned": []
}

f = open(outputFile, 'w')
f.write(json.dumps(wrapper, indent=4, sort_keys=True))
f.close()
exit(%d)
`
	return fmt.Sprintf(scriptTemplate, newDescription, scriptExitCode)
}

// Returns a migration script that produces an invalid service definition (empyt PoolID)
func (dt *DaoTest) getInvalidMigrationScript() string {
	return `
import json, os, string, sys

inputFile = os.environ["MIGRATE_INPUTFILE"]
outputFile = os.environ["MIGRATE_OUTPUTFILE"]
svcs = json.loads(open(inputFile, 'r').read())

svcs[0]["PoolID"] = ""

wrapper = {
	"Modified": svcs,
	"Cloned": []
}

f = open(outputFile, 'w')
f.write(json.dumps(wrapper, indent=4, sort_keys=True))
f.close()
`
}

func (dt *DaoTest) assertServiceMigrated(t *C, svc *service.Service, newDescription string) bool {
	result := service.Service{}
	err := dt.Dao.GetService(svc.ID, &result)
	if err != nil {
		t.Errorf("Unable to read service %s: %s", svc.ID, err)
		t.Fail()
	}

	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result.UpdatedAt = svc.UpdatedAt
	result.CreatedAt = svc.CreatedAt

	// For these unit tests, only desription is changed by migration
	svc.Description = newDescription
	if !svc.Equals(&result) {
		t.Errorf("Expected Service %+v, Actual Service %+v", result, *svc)
		t.Fail()
		return false
	}

	return true
}

func (dt *DaoTest) assertServiceNotMigrated(t *C, svc *service.Service) bool {
	result := service.Service{}
	err := dt.Dao.GetService(svc.ID, &result)
	if err != nil {
		t.Errorf("Unable to read service %s: %s", svc.ID, err)
		t.Fail()
	}

	if !svc.Equals(&result) {
		t.Errorf("Expected Service %+v, Actual Service %+v", result, *svc)
		t.Fail()
		return false
	}

	return true
}
