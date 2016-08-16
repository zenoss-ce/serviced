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

	"github.com/control-center/serviced/web/statsapi"
)

// TODO - configurable defaults?
var defaultDuration, _ = time.ParseDuration("1h")
var defaultResolution, _ = time.ParseDuration("5m")

func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, msInt*int64(time.Millisecond)), nil
}

// newStatRequest creates a new stat request from
// a Values map, defaults values, and validates them
func newStatRequest(entity string, query url.Values) (sr *statsapi.StatRequest, err error) {
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

	sr = &statsapi.StatRequest{
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

	statInfo, err := statsapi.GetStatInfo(entity)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	glog.V(4).Infof("restGetStatsMeta: entity %s, details: %#v", entity, statInfo.Details)
	writeJSON(w, statInfo.Details, 200)
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

	statInfo, err := statsapi.GetStatInfo(entity)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	results, err := statInfo.Fetch(sr, statInfo)
	if err != nil {
		restBadRequest(w, fmt.Errorf("Error fetching stats for %s: %s", entity, err))
		return
	}

	// key results by EntityID
	response := make(map[string][]statsapi.StatResult)
	for _, result := range results {
		response[result.EntityID] = append(response[result.EntityID], result)
	}

	w.WriteJson(&response)
}
