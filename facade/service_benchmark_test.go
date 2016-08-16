// Copyright 2016 The Serviced Authors.
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

package facade

import (
	"testing"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/datastore"
)

var FBT = NewFBT()

type FacadeBenchmarkTest struct {
	Ctx datastore.Context
	BFacade *Facade
	Bzzk    ZZK
}

// NewFBT creates an initialized FacadeBenchmarkTest instance
func NewFBT() *FacadeBenchmarkTest {
	newFacade := New()
	return &FacadeBenchmarkTest{
		BFacade: newFacade,
		Bzzk: GetFacadeZZK(newFacade),
	}
}

func BenchmarkServiceGet(b *testing.B) {
	// get context for test
	ctx := datastore.Get()
	if ctx == nil {
		b.Errorf("Cannot proceed - nil context.")
	}
	// load services into zookeeper
	testServices := createTestServices(1000)
	// reset timer
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		name := "a service"
		if service, err := FBT.BFacade.GetService(ctx, name); err != nil {
			b.Errorf("Error getting service %s: %s\n", name, err)
		} else if service == nil {
			b.Errorf("Error: GetService %s returned nil", name)
		}

	}
	removeTestServices(testServices)
}

func createTestServices(n int) []service.Service{
	var services []service.Service
	for i := 0; i < n; i++ {
		services = append(services,createSimpleService())
	}
	return services
}

func createSimpleService() service.Service{
	return service.Service {
		ID: fakeID(),
		Name: fakeServiceName(),
		Instances: 1,
		Description: fakeServiceDescription(),
		ParentServiceID: "FAKEPARENTSERVICEID",
	}
}

func fakeID() string {
	return "FAKE SERVICE ID"
}

func fakeServiceName() string {
	return "Fake Service Name"
}

func fakeServiceDescription() string {
	return "This is a description of a fake service"
}

func removeTestServices([]service.Service) {
}