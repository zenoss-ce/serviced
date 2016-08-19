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

// +build unit

package statsapi_test

import (
	"testing"

	. "gopkg.in/check.v1"

	. "github.com/control-center/serviced/statsapi"
)

func TestUtils(t *testing.T) { TestingT(t) }

type StatsAPISuite struct{}

var _ = Suite(&StatsAPISuite{})

func (s *StatsAPISuite) TestGetStatInfo(c *C) {
	info := StatInfo{
		Details: []StatDetails{
			{
				StatID:    "size",
				Label:     "Size",
				Unit:      "B",
				Threshold: "90%",
			},
		},
		Fetch: func(sr *StatRequest, info *StatInfo) (results []StatResult, err error) {
			return
		},
	}

	AddStatInfo("karatehorses", info)

	_, err := GetStatInfo("regularhorses")
	_, ok := err.(*MissingStatInfo)
	c.Assert(ok, Equals, true)

	info2, err := GetStatInfo("karatehorses")
	c.Assert(err, IsNil)
	c.Assert(info.Details, DeepEquals, info2.Details)
}

func (s *StatsAPISuite) TestGetStatDetail(c *C) {
	deets := []StatDetails{
		{
			StatID:    "size",
			Label:     "Size",
			Unit:      "B",
			Threshold: "90%",
		},
		{
			StatID:    "height",
			Label:     "Height",
			Unit:      "inch",
			Threshold: "70%",
		},
	}

	_, err := GetStatDetail(deets, "width")
	_, ok := err.(*MissingStatDetails)
	c.Assert(ok, Equals, true)

	deet, err := GetStatDetail(deets, "height")
	c.Assert(err, IsNil)
	c.Assert(deet, DeepEquals, &deets[1])
}

func (s *StatsAPISuite) TestApplyThreshold(c *C) {
	val, _ := ApplyThreshold("60%", 100)
	c.Assert(val, Equals, 60)

	val2, _ := ApplyThreshold("60", 100)
	c.Assert(val2, Equals, 60)

	_, err := ApplyThreshold("", 100)
	c.Assert(err, Equals, ErrMissingThreshold)

	_, err2 := ApplyThreshold("ABC%", 100)
	c.Assert(err2, NotNil)

	_, err3 := ApplyThreshold("ABC", 100)
	c.Assert(err3, NotNil)
}

//func NewStatRequest(entity string, opts map[string][]string) (*statsapi.StatRequest, error) {
func (s *StatsAPISuite) TestNewStatRequest(c *C) {
	/*
		req := map[string][]string{
			"stat":       {},
			"end":        {},
			"start":      {},
			"id":         {},
			"resolution": {},
		}

		sr, err := NewStatRequest("karatehorses", req)
	*/
}
