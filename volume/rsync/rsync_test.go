// Copyright 2015 The Serviced Authors.
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

package rsync_test

import (
	"testing"

	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/volume/drivertest"
	// Register the rsync driver
	_ "github.com/control-center/serviced/volume/rsync"
)

// Wire in gocheck
func Test(t *testing.T) { TestingT(t) }

type RsyncSuite struct{}

var _ = Suite(&RsyncSuite{})

func (s *RsyncSuite) TestRsyncCreateEmpty(c *C) {
	drivertest.DriverTestCreateEmpty(c, "rsync", "")
}

func (s *RsyncSuite) TestRsyncCreateBase(c *C) {
	drivertest.DriverTestCreateBase(c, "rsync", "")
}

func (s *RsyncSuite) TestRsyncSnapshots(c *C) {
	drivertest.DriverTestSnapshots(c, "rsync", "")
}

func (s *RsyncSuite) TestRsyncExportImport(c *C) {
	drivertest.DriverTestExportImport(c, "rsync", "", "")
}
