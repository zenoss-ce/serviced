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

package statsapi

import (
	"fmt"
	"time"

	"github.com/control-center/serviced/metrics"
)

var (
	// TODO - configurable defaults?
	defaultDuration, _   = time.ParseDuration("1h")
	defaultResolution, _ = time.ParseDuration("5m")

	// availableStats is a registry of StatInfo, keyed by
	// entity name
	availableStats = map[string]StatInfo{}
)

// StatRequestError is created when there
// is an error creating a StatRequest
type StatRequestError struct {
	Message string
}

func (err StatRequestError) Error() string {
	return err.Message
}

// MissingStatInfo occurs when a StatInfo
// is requested but the entity is not in
// the list
type MissingStatInfo struct {
	Message string
}

func (err MissingStatInfo) Error() string {
	return err.Message
}

// MissingStatDetails occurs when a StatDetails
// is requested but the entity is not in
// the list
type MissingStatDetails struct {
	Message string
}

func (err MissingStatDetails) Error() string {
	return err.Message
}

// StatRequest is a validated and defaulted
// request for stats
type StatRequest struct {
	EntityType         string
	EntityIDs          []string
	Stats              []string
	Start              time.Time
	End                time.Time
	Resolution         time.Duration
	QueryServiceClient *metrics.Client // Temporary until we have a more solid design
}

// StatResult contains stat values as well
// as other supporting information about a
// specific stat
type StatResult struct {
	EntityID  string `json:"-"`
	Stat      string `json:"stat"`
	Values    []int  `json:"values"`
	Threshold int    `json:"threshold"`
	Capacity  int    `json:"capacity"`
	Error     string `json:"error,omitempty"`
}

// StatInfo provides a way to describe available
// stats and how to get them. This serves as the
// interface for adding more stats and endpoints
type StatInfo struct {
	Details []StatDetails
	Fetch   StatFetcher
}

// StatDetails provides details about an available stat
type StatDetails struct {
	StatID    string
	Label     string
	Unit      string
	Threshold string `json:"-"`
}

// StatFetcher is function that, given a StatRequest,
// can produce an array of StatResults
type StatFetcher func(*StatRequest, *StatInfo) ([]StatResult, error)

// GetStatInfo looksup StatInfo object in
// availableStats for the given entity
func GetStatInfo(entity string) (*StatInfo, error) {
	statInfo, ok := availableStats[entity]
	if !ok {
		return nil, &MissingStatInfo{
			Message: fmt.Sprintf("No stat info for entity %s", entity),
		}
	}
	return &statInfo, nil
}

// AddStatInfo allows new StatInfo objects
// to be added to availableStats, keyed by
// entity name (eg: hosts, masters)
func AddStatInfo(entity string, s StatInfo) error {
	availableStats[entity] = s
	return nil
}

// GetStatDetail searches through a StatInfo for
// the StatDetails object that matches the provided
// stat id
func GetStatDetail(details []StatDetails, statID string) (*StatDetails, error) {
	for _, i := range details {
		if i.StatID == statID {
			return &i, nil
		}
	}
	return nil, &MissingStatDetails{
		Message: fmt.Sprintf("No stat detail for stat %s", statID),
	}
}

// NewStatRequest creates a new stat request from
// a map options, defaults values as needed, and validates them
func NewStatRequest(entity string, opts map[string][]string) (*StatRequest, error) {
	// required fields
	stats, ok := opts["stat"]
	if !ok || len(stats) == 0 {
		return nil, &StatRequestError{
			Message: "at least one stat is required",
		}
	}

	// optional fields
	var end time.Time
	if endArr, ok := opts["end"]; ok && (len(endArr) == 1) {
		endStr := endArr[0]
		var err error
		end, err = MSToTime(endStr)
		if err != nil {
			return nil, &StatRequestError{
				Message: fmt.Sprintf("invalid end time %s", endStr),
			}
		}
	}

	var start time.Time
	if startArr, ok := opts["start"]; ok && (len(startArr) == 1) {
		startStr := startArr[0]
		var err error
		start, err = MSToTime(startStr)
		if err != nil {
			return nil, &StatRequestError{
				Message: fmt.Sprintf("invalid start time %s", startStr),
			}
		}
	}

	// if start, end, or both are missing, create them
	if end.IsZero() && start.IsZero() {
		end = time.Now()
		start = end.Add(-defaultDuration)
	} else if end.IsZero() {
		// NOTE - this can produce an end time
		// in the future
		end = start.Add(defaultDuration)
	} else if start.IsZero() {
		start = end.Add(-defaultDuration)
	}

	if !end.After(start) {
		return nil, &StatRequestError{
			Message: fmt.Sprintf("end time must be after start time"),
		}
	}

	ids, _ := opts["id"]

	var res time.Duration
	if resArr, ok := opts["resolution"]; !ok || (len(resArr) == 0) {
		res = defaultResolution
	} else {
		resStr := resArr[0]
		var err error
		res, err = time.ParseDuration(resStr)
		if err != nil {
			return nil, &StatRequestError{
				Message: fmt.Sprintf("invalid resolution %s", resStr),
			}
		}
	}

	sr := &StatRequest{
		EntityType: entity,
		Stats:      stats,
		EntityIDs:  ids,
		Start:      start,
		End:        end,
		Resolution: res,
	}

	return sr, nil
}
