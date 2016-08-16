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

package statsapi

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// StatRequest is a validated and defaulted
// request for stats
type StatRequest struct {
	EntityIDs  []string
	Stats      []string
	Start      time.Time
	End        time.Time
	Resolution time.Duration
}

// StatResult contains stat values as well
// as other supporting information about a
// specific stat
type StatResult struct {
	EntityID  string `json:"-"`
	Stat      string
	Values    []int
	Threshold int
	Capacity  int
	Error     string
}

// StatFetcher is function that, given a StatRequest,
// can produce an array of StatResults
type StatFetcher func(*StatRequest, StatInfo) ([]StatResult, error)

// StatDetails provides details about an available stat
type StatDetails struct {
	StatID    string
	Label     string
	Unit      string
	Threshold string
}

// StatInfo provides a way to describe available
// stats and how to get them. This serves as the
// interface for adding more stats and endpoints
type StatInfo struct {
	Details []StatDetails
	Fetch   StatFetcher
}

var hostsInfo = StatInfo{
	[]StatDetails{
		{"mem", "Memory", "B", "90%"},
		{"cpu", "CPU", "pct", "90%"},
		{"docker_storage", "Docker Storage", "B", "90%"},
	},
	demoStatFetcher,
}

var mastersInfo = StatInfo{
	[]StatDetails{
		{"mem", "Memory", "B", "90%"},
		{"cpu", "CPU", "pct", "90%"},
		{"docker_storage", "Docker Storage", "B", "90%"},
		{"dfs_storage", "DFS Storage", "B", "90%"},
	},
	demoStatFetcher,
}

var backupsInfo = StatInfo{
	[]StatDetails{
		{"size", "Size", "B", "90%"},
	},
	demoStatFetcher,
}

var poolsInfo = StatInfo{
	[]StatDetails{
		{"mem", "Memory", "B", "90%"},
		{"cpu", "CPU", "pct", "90%"},
	},
	demoStatFetcher,
}

var isvcsInfo = StatInfo{
	[]StatDetails{
		{"mem", "Memory", "B", "90%"},
		{"cpu", "CPU", "pct", "90%"},
		{"size", "Size", "B", "90%"},
	},
	demoStatFetcher,
}

var availableStats = map[string]StatInfo{
	"hosts":   hostsInfo,
	"masters": mastersInfo,
	"backups": backupsInfo,
	"pools":   poolsInfo,
	"isvcs":   isvcsInfo,
}

// getStatDetail searches through a StatInfo for
// the StatDetails object that matches the provided
// stat id
func getStatDetail(details []StatDetails, statID string) (StatDetails, error) {
	for _, i := range details {
		if i.StatID == statID {
			return i, nil
		}
	}
	return StatDetails{}, fmt.Errorf("Could not find stat %s", statID)
}

func applyThreshold(threshold string, val int) (int, error) {
	if threshold == "" {
		return 0, fmt.Errorf("Threshold is empty")
	}

	// apply threshold percentage to total val
	if strings.HasSuffix(threshold, "%") {
		trimmed := strings.TrimSuffix(threshold, "%")
		percent, err := strconv.Atoi(trimmed)
		if err != nil {
			return 0, err
		}
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

func demoStatFetcher(sr *StatRequest, info StatInfo) (results []StatResult, err error) {
	// NOTE - pretend request is for hosts
	entity := "hosts"
	details := info.Details

	for _, stat := range sr.Stats {
		// if detailErr, create results for each
		// EntityID anyway, just make it an "error" result
		detail, detailErr := getStatDetail(details, stat)

		for _, id := range sr.EntityIDs {

			// TODO - go somewhere and fetch values, capacity
			values := []int{40, 27, 27, 34, 40, 90, 89, 50, 40, 30}
			capacity := 100
			threshold, thresholdErr := applyThreshold(detail.Threshold, capacity)

			if detailErr != nil {
				results = append(results, StatResult{
					EntityID: id,
					Stat:     stat,
					Error:    fmt.Sprintf("Invalid stat %s for entity %s", stat, entity),
				})
			} else if thresholdErr != nil {
				results = append(results, StatResult{
					EntityID: id,
					Stat:     stat,
					Error:    fmt.Sprintf("Could not apply threshold %s", detail.Threshold),
				})
			} else {
				results = append(results, StatResult{
					EntityID:  id,
					Stat:      stat,
					Values:    values,
					Capacity:  capacity,
					Threshold: threshold,
				})
			}
		}
	}
	return results, nil
}

// GetStatInfo returns a StatInfo object
// for the given entity
func GetStatInfo(entity string) (StatInfo, error) {
	statInfo, ok := availableStats[entity]
	if !ok {
		return StatInfo{}, fmt.Errorf("No stat info for entity %s", entity)
	}
	return statInfo, nil
}
