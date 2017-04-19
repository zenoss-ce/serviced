// Copyright 2017 The Serviced Authors.
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

package audit

import "github.com/control-center/serviced/logging"

// initialize the package logger
var plog = logging.PackageLogger()
var alog = logging.AuditLogger()

type OriginType string

const (
	UI       OriginType = "UI"
	CLI                 = "CLI"
	Internal            = "Internal Call"
)

type AuditLogRequest struct {
	User    string
	Message string
	Origin  OriginType
	Intent  string
}
