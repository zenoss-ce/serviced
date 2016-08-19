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
	"strconv"
	"strings"
	"time"
)

// ErrMissingThreshold occurs when a
// threshold is necessary by absent
var ErrMissingThreshold = errors.New("threshold cannot be empty")

// ApplyThreshold takes a threshold and a value to apply to.
// If the threshold is a percent, it is applied to the value
// and the result returned. If the threshold is a number, it is
// parsed to int and returned. Eg: 100% or 872891
func ApplyThreshold(threshold string, val int) (int, error) {
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

// MSToTime takes milliseconds since epoch and
// returns a time object
func MSToTime(ms string) (time.Time, error) {
	msInt, err := strconv.ParseInt(ms, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(0, msInt*int64(time.Millisecond)), nil
}

// TimeToMS takes a time object and returns
// milliseconds since epoch
func TimeToMS(t time.Time) int {
	return int(t.UnixNano() / 1000000)
}
