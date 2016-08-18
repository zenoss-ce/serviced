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

package statsapi

import (
	"testing"

	. "gopkg.in/check.v1"
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

	addStatInfo("karatehorses", info)

	_, err := GetStatInfo("regularhorses")
	_, ok := err.(*MissingStatInfo)
	c.Assert(ok, Equals, true)

	info2, err := GetStatInfo("karatehorses")
	c.Assert(err, IsNil)
	c.Assert(info.Details, DeepEquals, info2.Details)
}

func (s *StatsAPISuite) TestgetStatDetail(c *C) {
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

	_, err := getStatDetail(deets, "width")
	_, ok := err.(*MissingStatDetails)
	c.Assert(ok, Equals, true)

	deet, err := getStatDetail(deets, "height")
	c.Assert(err, IsNil)
	c.Assert(deet, DeepEquals, &deets[1])
}

func (s *StatsAPISuite) TestapplyThreshold(c *C) {
	val, _ := applyThreshold("60%", 100)
	c.Assert(val, Equals, 60)

	val2, _ := applyThreshold("60", 100)
	c.Assert(val2, Equals, 60)

	_, err := applyThreshold("", 100)
	c.Assert(err, Equals, ErrMissingThreshold)

	_, err2 := applyThreshold("ABC%", 100)
	c.Assert(err2, NotNil)

	_, err3 := applyThreshold("ABC", 100)
	c.Assert(err3, NotNil)
}
