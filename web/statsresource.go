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

package web

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/zenoss/go-json-rest"

	"github.com/control-center/serviced/statsapi"
)

// StatRequestError is created when there
// is an error creating a StatRequest
type StatRequestError struct {
	Message string
}

// ErrMissingEntityName is created when
// an entity name is required but is empty
var ErrMissingEntityName = errors.New("entity name cannot be empty")

func (err StatRequestError) Error() string {
	return err.Message
}

// TODO - configurable defaults?
var defaultDuration, _ = time.ParseDuration("1h")
var defaultResolution, _ = time.ParseDuration("5m")

// msToTime takes milliseconds since epoch and
// returns a time object
func msToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, msInt*int64(time.Millisecond)), nil
}

// newStatRequest creates a new stat request from
// a Values map, defaults values, and validates them
func newStatRequest(entity string, query url.Values) (*statsapi.StatRequest, error) {
	// required fields
	stats, ok := query["stat"]
	if !ok || len(stats) == 0 {
		return nil, &StatRequestError{
			Message: "at least one stat is required",
		}
	}

	// optional fields
	var end time.Time
	if endStr := query.Get("end"); endStr != "" {
		var err error
		end, err = msToTime(endStr)
		if err != nil {
			return nil, &StatRequestError{
				Message: fmt.Sprintf("invalid end time %s", endStr),
			}
		}
	}

	var start time.Time
	if startStr := query.Get("start"); startStr != "" {
		var err error
		start, err = msToTime(startStr)
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

	ids, _ := query["id"]

	var res time.Duration
	if resStr := query.Get("resolution"); resStr == "" {
		res = defaultResolution
	} else {
		var err error
		res, err = time.ParseDuration(resStr)
		if err != nil {
			return nil, &StatRequestError{
				Message: fmt.Sprintf("invalid resolution %s", resStr),
			}
		}
	}

	sr = &statsapi.StatRequest{
		EntityType: entity,
		Stats:      stats,
		EntityIDs:  ids,
		Start:      start,
		End:        end,
		Resolution: res,
	}

	return &sr, nil
}

func restGetStatsMeta(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	entity, err := url.QueryUnescape(r.PathParam("entity"))
	if err != nil {
		restBadRequest(w, ErrMissingEntityName)
		return
	}

	sr := &statsapi.StatRequest{EntityType: entity}
	statInfo, err := ctx.getFacade().GetStatsMetadata(sr)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	writeJSON(w, statInfo.Details, 200)
}

func restGetStats(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	entity, err := url.QueryUnescape(r.PathParam("entity"))
	if err != nil {
		restBadRequest(w, ErrMissingEntityName)
		return
	}

	// TODO - err handle
	query := r.URL.Query()

	sr, err := newStatRequest(entity, query)
	if err != nil {
		restBadRequest(w, err)
		return
	}

	results, err := ctx.getFacade().GetStats(sr)

	if err != nil {
		restBadRequest(w, fmt.Errorf("error fetching stats for %s: %s", entity, err))
		return
	}

	// key results by EntityID
	response := make(map[string][]statsapi.StatResult)
	for _, result := range results {
		response[result.EntityID] = append(response[result.EntityID], result)
	}

	w.WriteJson(&response)
}
