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

package service

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/elastigo/search"

	"errors"
	"strings"
	"time"
	"sync"
	"github.com/control-center/serviced/logging"
)

var (
	log = logging.PackageLogger()
	ErrInvalidDesiredState = errors.New("invalid DesiredState value")
)

// NewStore creates a Service store
func NewStore() Store {
	return &storeImpl{}
}

// Store type for interacting with Service persistent storage
type Store interface {
	// Put adds or updates a Service
	Put(ctx datastore.Context, svc *Service) error

	// Get a Service by id. Return ErrNoSuchEntity if not found
	Get(ctx datastore.Context, id string) (*Service, error)

	// Delete removes the a Service if it exists
	Delete(ctx datastore.Context, id string) error

	// Update the DesiredState for the service
	UpdateDesiredState(ctx datastore.Context, serviceID string, desiredState int) error

	// GetServices returns all services
	GetServices(ctx datastore.Context) ([]Service, error)

	// GetUpdatedServices returns all services updated since "since" time.Duration ago
	GetUpdatedServices(ctx datastore.Context, since time.Duration) ([]Service, error)

	// GetTaggedServices returns services with the given tags
	GetTaggedServices(ctx datastore.Context, tags ...string) ([]Service, error)

	// GetServicesByPool returns services with the given pool id
	GetServicesByPool(ctx datastore.Context, poolID string) ([]Service, error)

	// GetServicesByDeployment returns services with the given deployment id
	GetServicesByDeployment(ctx datastore.Context, deploymentID string) ([]Service, error)

	// GetChildServices returns services that are children of the given parent service id
	GetChildServices(ctx datastore.Context, parentID string) ([]Service, error)

	FindChildService(ctx datastore.Context, deploymentID, parentID, serviceName string) (*Service, error)

	// FindTenantByDeployment returns the tenant service for a given deployment id and service name
	FindTenantByDeploymentID(ctx datastore.Context, deploymentID, name string) (*Service, error)

	// GetAllServiceDetails returns all service details
	GetAllServiceDetails(ctx datastore.Context) ([]ServiceDetails, error)

	// GetServiceDetails returns the details for the given service
	GetServiceDetails(ctx datastore.Context, serviceID string) (*ServiceDetails, error)

	// GetChildServiceDetails returns the details for the child service of the given parent
	GetServiceDetailsByParentID(ctx datastore.Context, parentID string) ([]ServiceDetails, error)
}

type storeImpl struct {
	ds datastore.DataStore
}

// Put adds or updates a Service
func (s *storeImpl) Put(ctx datastore.Context, svc *Service) error {
	//No need to store ConfigFiles
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.Put"))
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)

	err := s.ds.Put(ctx, Key(svc.ID), svc)
	if err == nil {
		updateVolatileInfo(svc.ID, svc.DesiredState)
	}
	return err
}

// UpdateDesiredState updates the DesiredState for the service by saving the information in volatile storage.
func (s *storeImpl) UpdateDesiredState(ctx datastore.Context, serviceID string, desiredState int) error {
	log.Infof("Storing desiredState %d for service %s", desiredState, serviceID)
	updateVolatileInfo(serviceID, desiredState)
	return nil
}

// Get a Service by id. Return ErrNoSuchEntity if not found
func (s *storeImpl) Get(ctx datastore.Context, id string) (*Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.Get"))
	svc := &Service{}
	if err := s.ds.Get(ctx, Key(id), svc); err != nil {
		return nil, err
	}

	fillAdditionalInfo(svc)
	return svc, nil
}

// Delete removes the a Service if it exists
func (s *storeImpl) Delete(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.Delete"))
	err := s.ds.Delete(ctx, Key(id))
	if err == nil {
		removeVolatileInfo(id)
	}
	return err
}

// GetServices returns all services
func (s *storeImpl) GetServices(ctx datastore.Context) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.GetServices"))
	return query(ctx, "_exists_:ID")
}

// GetUpdatedServices returns all services updated since "since" time.Duration ago
func (s *storeImpl) GetUpdatedServices(ctx datastore.Context, since time.Duration) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.GetUpdatedServices"))
	q := datastore.NewQuery(ctx)
	t0 := time.Now().Add(-since)
	t0s := t0.Format(time.RFC3339)
	elasticQuery := search.Query().Range(search.Range().Field("UpdatedAt").From(t0s)).Search("_exists_:ID")
	search := search.Search("controlplane").Type(kind).Size("50000").Query(elasticQuery)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	// First get the list of updated services from Elastic.
	svcs, err := convert(results)
	// Then add updated services from the cache
	return s.addUpdatedServicesFromCache(ctx, svcs, t0)
}

// GetTaggedServices returns services with the given tags
func (s *storeImpl) GetTaggedServices(ctx datastore.Context, tags ...string) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.GetTaggedServices"))
	if len(tags) == 0 {
		return nil, errors.New("empty tags not allowed")
	}
	qs := strings.Join(tags, " AND ")
	return query(ctx, qs)
}

// GetServicesByPool returns services with the given pool id
func (s *storeImpl) GetServicesByPool(ctx datastore.Context, poolID string) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.GetServicesByPool"))
	id := strings.TrimSpace(poolID)
	if id == "" {
		return nil, errors.New("empty poolID not allowed")
	}
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("PoolID", id)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetServicesByDeployment returns services with the given deployment id
func (s *storeImpl) GetServicesByDeployment(ctx datastore.Context, deploymentID string) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.GetServicesByDeployment"))
	id := strings.TrimSpace(deploymentID)
	if id == "" {
		return nil, errors.New("empty deploymentID not allowed")
	}
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("DeploymentID", id)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// GetChildServices returns services that are children of the given parent service id
func (s *storeImpl) GetChildServices(ctx datastore.Context, parentID string) ([]Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.GetChildServices"))
	id := strings.TrimSpace(parentID)
	if id == "" {
		return nil, errors.New("empty parent service id not allowed")
	}
	q := datastore.NewQuery(ctx)
	query := search.Query().Term("ParentServiceID", id)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(query)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

func (s *storeImpl) FindChildService(ctx datastore.Context, deploymentID, parentID, serviceName string) (*Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.FindChildService"))
	parentID = strings.TrimSpace(parentID)

	if deploymentID = strings.TrimSpace(deploymentID); deploymentID == "" {
		return nil, errors.New("empty deployment ID not allowed")
	} else if serviceName = strings.TrimSpace(serviceName); serviceName == "" {
		return nil, errors.New("empty service name not allowed")
	}

	search := search.Search("controlplane").Type(kind).Filter(
		"and",
		search.Filter().Terms("DeploymentID", deploymentID),
		search.Filter().Terms("ParentServiceID", parentID),
		search.Filter().Terms("Name", serviceName),
	)

	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}

	if results.Len() == 0 {
		return nil, nil
	} else if svcs, err := convert(results); err != nil {
		return nil, err
	} else {
		return &svcs[0], nil
	}
}

// FindTenantByDeployment returns the tenant service for a given deployment id and service name
func (s *storeImpl) FindTenantByDeploymentID(ctx datastore.Context, deploymentID, name string) (*Service, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("storeImpl.FindTenantByDeploymentID"))
	if deploymentID = strings.TrimSpace(deploymentID); deploymentID == "" {
		return nil, errors.New("empty deployment ID not allowed")
	} else if name = strings.TrimSpace(name); name == "" {
		return nil, errors.New("empty service name not allowed")
	}

	search := search.Search("controlplane").Type(kind).Filter(
		"and",
		search.Filter().Terms("DeploymentID", deploymentID),
		search.Filter().Terms("Name", name),
		search.Filter().Terms("ParentServiceID", ""),
	)

	q := datastore.NewQuery(ctx)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}

	if results.Len() == 0 {
		return nil, nil
	} else if svcs, err := convert(results); err != nil {
		return nil, err
	} else {
		return &svcs[0], nil
	}
}

func query(ctx datastore.Context, query string) ([]Service, error) {
	q := datastore.NewQuery(ctx)
	elasticQuery := search.Query().Search(query)
	search := search.Search("controlplane").Type(kind).Size("50000").Query(elasticQuery)
	results, err := q.Execute(search)
	if err != nil {
		return nil, err
	}
	return convert(results)
}

// fillAdditionalInfo fills the service object with additional information
// that amends or overrides what was retrieved from elastic
func fillAdditionalInfo(svc *Service) {
	fillConfig(svc)
	fillVolatileInfo(svc)
}

// fillConfig fills in the ConfgiFiles values
func fillConfig(svc *Service) {
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)
	for key, val := range svc.OriginalConfigs {
		svc.ConfigFiles[key] = val
	}
}

func convert(results datastore.Results) ([]Service, error) {
	svcs := make([]Service, results.Len())
	for idx := range svcs {
		var svc Service
		err := results.Get(idx, &svc)
		if err != nil {
			return nil, err
		}
		fillAdditionalInfo(&svc)
		svcs[idx] = svc
	}
	return svcs, nil
}

//Key creates a Key suitable for getting, putting and deleting Services
func Key(id string) datastore.Key {
	return datastore.NewKey(kind, id)
}

//confFileKey creates a Key suitable for getting, putting and deleting svcConfigFile
func confFileKey(id string) datastore.Key {
	return datastore.NewKey(confKind, id)
}

var (
	kind     = "service"
	confKind = "serviceconfig"
)

type volatileService struct {
	ID             string
	DesiredState   int
	UpdatedAt      time.Time	// Time when the cached entry was changed, not when elastic was changed
}

var serviceCacheLock = &sync.RWMutex{}
var serviceCache = map[string]volatileService{}

// take the list of services and append services updated since 'since'
func (s *storeImpl) addUpdatedServicesFromCache(ctx datastore.Context, svcs []Service, since time.Time) ([]Service, error) {
	// If getting these one at a time turns out to be hard on elastic, we can
	// later try batching the elastic queries for sets of N ids until we go
	// through the whole list with a new elastic search.
	for _, id := range getUpdatedServicesFromCache(since) {
		if svc, err := s.Get(ctx, id); err != nil {
			return svcs, err
		} else {
			svcs = append(svcs, *svc)
		}
	}
	return svcs, nil
}

// Returns the list of ids from the cache updated since the given time.
func getUpdatedServicesFromCache(since time.Time) []string {
	serviceCacheLock.RLock()
	defer serviceCacheLock.RUnlock()

	ids := []string{}
	for _, cacheEntry := range serviceCache {
		if since.After(cacheEntry.UpdatedAt) {
			ids = append(ids, cacheEntry.ID)
		}
	}
	return ids
}

// fillVolatileInfo fills volatile information into the service
func fillVolatileInfo(svc *Service) bool {
	serviceCacheLock.RLock()
	defer serviceCacheLock.RUnlock()
	cacheEntry, ok := serviceCache[svc.ID];
	if ok {
		svc.DesiredState = cacheEntry.DesiredState
	}
	return ok
}

// updateVolatileInfo updates the local cache for volatile information
func updateVolatileInfo(serviceID string, desiredState int) error {

	// Validate desired state
	switch desiredState {
	case int(SVCRun), int(SVCStop), int(SVCPause):
	default:
		return ErrInvalidDesiredState
	}

	serviceCacheLock.Lock()
	defer serviceCacheLock.Unlock()
	cacheEntry := volatileService{
		ID:           serviceID,
		DesiredState: desiredState,
		UpdatedAt:    time.Now(),
	}
	serviceCache[serviceID] = cacheEntry
	return nil
}

// removeVolatileInfo removes the service's information from the local cache
func removeVolatileInfo(serviceID string) {
	serviceCacheLock.Lock()
	defer serviceCacheLock.Unlock()
	delete(serviceCache, serviceID)
}