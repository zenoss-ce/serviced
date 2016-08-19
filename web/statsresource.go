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
	"time"

	"github.com/zenoss/go-json-rest"

	"github.com/control-center/serviced/statsapi"
)

// ErrMissingEntityName is created when
// an entity name is required but is empty
var ErrMissingEntityName = errors.New("entity name cannot be empty")

// statsResponse is the response struct that is
// serialized and sent to the client
type statsResponse struct {
	Start      int                              `json:"start"`
	End        int                              `json:"end"`
	Resolution time.Duration                    `json:"resolution"`
	Results    map[string][]statsapi.StatResult `json:"results"`
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

	sr, err := statsapi.NewStatRequest(entity, query)
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
	byID := make(map[string][]statsapi.StatResult)
	for _, result := range results {
		byID[result.EntityID] = append(byID[result.EntityID], result)
	}

	w.WriteJson(&statsResponse{
		Start:      statsapi.TimeToMS(sr.Start),
		End:        statsapi.TimeToMS(sr.End),
		Resolution: sr.Resolution,
		Results:    byID,
	})
}
