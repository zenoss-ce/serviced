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

package web

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, msInt*int64(time.Millisecond)), nil
}

// statRequest is a validated and defaulted
// request for stats
type statRequest struct {
	EntityIDs  []string
	Stats      []string
	Start      time.Time
	End        time.Time
	Resolution time.Duration
}

// statResult contains stat values as well
// as other supporting information about a
// specific stat
type statResult struct {
	entityID  string
	Stat      string
	Values    []int
	Threshold int
	Capacity  int
	Error     string
}

// statFetcher is function that, given a statRequest,
// can produce an array of statResults
type statFetcher func(*statRequest) ([]statResult, error)

// StatDetails provides details about an available stat
type StatDetails struct {
	StatID string
	Label  string
	Unit   string
}

// StatInfo provides a way to describe available
// stats and how to get them
type StatInfo struct {
	details []StatDetails
	fetch   statFetcher
}

var availableStats = map[string]StatInfo{
	"hosts": StatInfo{
		[]StatDetails{
			{"mem", "Memory", "B"},
			{"cpu", "CPU", "pct"},
			{"docker_storage", "Docker Storage", "B"},
		},
		demoStatFetcher,
	},
	"masters": StatInfo{
		[]StatDetails{
			{"mem", "Memory", "B"},
			{"cpu", "CPU", "pct"},
			{"docker_storage", "Docker Storage", "B"},
			{"dfs_storage", "DFS Storage", "B"},
		},
		demoStatFetcher,
	},
	"backups": StatInfo{
		[]StatDetails{
			{"size", "Size", "B"},
		},
		demoStatFetcher,
	},
	"pools": StatInfo{
		[]StatDetails{
			{"mem", "Memory", "B"},
			{"cpu", "CPU", "pct"},
		},
		demoStatFetcher,
	},
	"isvcs": StatInfo{
		[]StatDetails{
			{"mem", "Memory", "B"},
			{"cpu", "CPU", "pct"},
			{"size", "Size", "B"},
		},
		demoStatFetcher,
	},
}

func demoStatFetcher(sr *statRequest) (results []statResult, err error) {
	for _, stat := range sr.Stats {
		for _, id := range sr.EntityIDs {
			// TODO - get threshold from somewhere
			threshold := 90
			results = append(results, statResult{
				entityID:  id,
				Stat:      stat,
				Values:    []int{40, 27, 27, 34, 40, 90, 89, 50, 40, 30},
				Capacity:  100,
				Threshold: threshold,
				Error:     "",
			})
		}
	}
	return results, nil
}

// TODO - configurable defaults?
var defaultDuration, _ = time.ParseDuration("1h")
var defaultResolution, _ = time.ParseDuration("5m")

// creates a new stat request from
// a Values map, defaults values, and validates them
func newStatRequest(entity string, query url.Values) (sr *statRequest, err error) {
	// required fields
	stats, ok := query["stat"]
	if !ok || len(stats) == 0 {
		return nil, fmt.Errorf("at least one stat is required")
	}

	// optional fields
	endStr := query.Get("end")
	startStr := query.Get("start")
	var end time.Time
	var start time.Time

	// parse and validate start and end if present
	if endStr != "" {
		end, err = msToTime(endStr)
		if err != nil {
			return nil, fmt.Errorf("invalid end time %s", endStr)
		}
	}
	if startStr != "" {
		start, err = msToTime(startStr)
		if err != nil {
			return nil, fmt.Errorf("invalid start time %s", startStr)
		}
	}

	// if start, end, or both are missing, create them
	if endStr == "" && startStr == "" {
		end = time.Now()
		start = end.Add(-defaultDuration)
	} else if endStr == "" {
		// NOTE - this can produce an end time
		// in the future
		end = start.Add(defaultDuration)
	} else if startStr == "" {
		start = end.Add(-defaultDuration)
	}

	if !end.After(start) {
		return nil, fmt.Errorf("end time must be after start time")
	}

	ids, ok := query["id"]
	if !ok {
		ids = []string{}
	}

	resStr := query.Get("resolution")
	var res time.Duration
	if resStr == "" {
		res = defaultResolution
	} else {
		res, err = time.ParseDuration(resStr)
		if err != nil {
			return nil, fmt.Errorf("invalid resolution %s", resStr)
		}
	}

	sr = &statRequest{
		Stats:      stats,
		EntityIDs:  ids,
		Start:      start,
		End:        end,
		Resolution: res,
	}

	return sr, nil
}

func restGetStatsMeta(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	entity, err := url.QueryUnescape(r.PathParam("entity"))
	if err != nil {
		restBadRequest(w, fmt.Errorf("Missing entity name"))
		return
	}

	statInfo, ok := availableStats[entity]
	if !ok {
		restBadRequest(w, fmt.Errorf("No stat info for entity %s", entity))
		return
	}

	glog.V(4).Infof("restGetStatsMeta: entity %s, details: %#v", entity, statInfo.details)
	writeJSON(w, statInfo.details, 200)
}

func restGetStats(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	entity, err := url.QueryUnescape(r.PathParam("entity"))
	if err != nil {
		restBadRequest(w, fmt.Errorf("Missing entity name"))
		return
	}

	// TODO - err handle
	query := r.URL.Query()

	sr, err := newStatRequest(entity, query)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	statInfo, ok := availableStats[entity]
	if !ok {
		restBadRequest(w, fmt.Errorf("No stat info for entity %s", entity))
		return
	}

	results, err := statInfo.fetch(sr)
	if err != nil {
		restBadRequest(w, fmt.Errorf("Error fetching stats for %s: %s", entity, err))
		return
	}

	// key results by entityID
	response := make(map[string][]statResult)
	for _, result := range results {
		response[result.entityID] = append(response[result.entityID], result)
	}

	w.WriteJson(&response)
}
