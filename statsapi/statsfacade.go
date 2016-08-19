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

// This package provides a facade to retrieve stats from Query Service

package statsapi

import (
	"github.com/control-center/serviced/metrics"
)

type statsFacade struct {
	QueryServiceClient *metrics.Client
	// CACHE
}

var (
	timeFormat             = "2006/01/02-15:04:00-MST"
	emptyResult            = []StatResult{}
	statToMetricTranslator = map[string]string{
		"cpu": "cpu.user",
		"mem": "memory.actualused",
	}
	metricToStatTranslator = map[string]string{}
)

func init() {
	for k, v := range statToMetricTranslator {
		metricToStatTranslator[v] = k
	}
}

func (sf *statsFacade) getHostStats(sr *StatRequest) (results []StatResult, err error) {
	// 1 - Build metrics.performance.PerformanceOptions from StatRequest
	query := metrics.PerformanceOptions{
		Start:     sr.Start.Format(timeFormat),
		End:       sr.End.Format(timeFormat),
		Returnset: "exact",
		Tags: map[string][]string{
			"controlplane_host_id": sr.EntityIDs,
		},
		Metrics: []metrics.MetricOptions{},
	}

	for _, stat := range sr.Stats {
		query.Metrics = append(query.Metrics, metrics.MetricOptions{
			Metric:     statToMetricTranslator[stat],
			Aggregator: "avg",
		})
	}

	// 2 - Make request to Query Service
	queryResult, err := sf.QueryServiceClient.PerformanceQuery(query)
	if err != nil {
		return emptyResult, err
	}

	// 3 - Transform result from metrics.performance.PerformanceData to StatResult
	results = []StatResult{}
	for _, metricResult := range queryResult.Results {
		nDatapoints := len(metricResult.Datapoints)
		statRes := &StatResult{
			EntityID: sr.EntityType,
			Stat:     metricToStatTranslator[metricResult.Metric],
			Values:   make([]float64, nDatapoints),
			//Threshold int    `json:"threshold"`
			//Capacity  int    `json:"capacity"`
			//Error     string `json:"error,omitempty"`
		}
		for _, dp := range metricResult.Datapoints {
			var statDataPoint float64 = -1.0
			if !dp.Value.IsNaN {
				statDataPoint = dp.Value.Value
			}
			statRes.Values = append(statRes.Values, statDataPoint)
		}
		results = append(results, *statRes)
	}

	return results, nil
}
