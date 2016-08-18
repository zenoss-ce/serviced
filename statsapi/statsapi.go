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
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/metrics"
)

// ErrMissingThreshold occurs when a
// threshold is necessary by absent
var ErrMissingThreshold = errors.New("threshold cannot be empty")

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

// availableStats is a registry of StatInfo, keyed by
// entity name
var availableStats = map[string]StatInfo{}

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

// addStatInfo allows new StatInfo objects
// to be added to availableStats, keyed by
// entity name (eg: hosts, masters)
func addStatInfo(entity string, s StatInfo) error {
	availableStats[entity] = s
	return nil
}

// getStatDetail searches through a StatInfo for
// the StatDetails object that matches the provided
// stat id
func getStatDetail(details []StatDetails, statID string) (*StatDetails, error) {
	for _, i := range details {
		if i.StatID == statID {
			return &i, nil
		}
	}
	return nil, &MissingStatDetails{
		Message: fmt.Sprintf("No stat detail for stat %s", statID),
	}
}

// applyThreshold takes a threshold and a value to apply to.
// If the threshold is a percent, it is applied to the value
// and the result returned. If the threshold is a number, it is
// parsed to int and returned. Eg: 100% or 872891
func applyThreshold(threshold string, val int) (int, error) {
	if threshold == "" {
		return 0, ErrMissingThreshold
	}

	// apply threshold percentage to total val
	if strings.HasSuffix(threshold, "%") {
		trimmed := strings.TrimSuffix(threshold, "%")
		percent, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, err
		}
		// TODO - is int sufficient precision?
		result := int(float64(percent) * 0.01 * float64(val))
		return result, nil
	}

	// just return threshold as int
	result, err := strconv.Atoi(threshold)
	if err != nil {
		return 0, err
	}
	return result, nil

}
