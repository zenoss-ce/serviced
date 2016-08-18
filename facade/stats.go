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

package facade

import (
	"errors"
	"fmt"
	"github.com/control-center/serviced/statsapi"
)

// Retrieve Stats metadata for a specific entity
func (f *Facade) GetStatsMetadata(sr *statsapi.StatRequest) (result *statsapi.StatInfo, err error) {
	return statsapi.GetStatInfo(sr.EntityType)
}

// Get Stats for entity defined in StatRequest
func (f *Facade) GetStats(sr *statsapi.StatRequest) (results []statsapi.StatResult, err error) {

	statInfo, err := statsapi.GetStatInfo(sr.EntityType)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Unknown Entity: %s", sr.EntityType))
	}

	sr.QueryServiceClient = f.GetQueryServiceClient() // TEMP For now we stick it in the statRequest
	switch sr.EntityType {
	case "masters":
		sr.EntityIDs = []string{f.ccMasterHost.ID}
		results, err = statInfo.Fetch(sr, statInfo)

	// NOTE - this is a demo stat getter and does
	// not actually fetch hosts
	case "hosts":
		results, err = statInfo.Fetch(sr, statInfo)

	default:
		return nil, errors.New(fmt.Sprintf("Entity Stat not implemented: %s", sr.EntityType))
	}

	return results, err
}
