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

package user

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/logging"
)

// User for the system???
type User struct {
	Name     string // the unique identifier for a user
	Password string // no requirements on passwords yet
	datastore.VersionedEntity
}

// initialize the package logger
var plog = logging.PackageLogger()
