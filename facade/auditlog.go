// Copyright 2017 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"fmt"
	"os"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/audit"
	"github.com/zenoss/logri"
)

//brute-force writer to file
func (f *Facade) setupAuditLogger() {
	auditlogloc := "/tmp/cc_bruteforce_audit.log"
	fileopt := map[string]string{"file": auditlogloc}
	w, err := logri.GetOutputWriter(logri.FileOutput, fileopt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting output writer for %s: %s\n", auditlogloc, err)
	}
	alog.SetOutput(w)
}

func (f *Facade) AuditLog(request audit.AuditLogRequest) error {
	f.setupAuditLogger()
	alog.WithFields(
		log.Fields{
			"audit":  "AUDIT",
			"user":   request.User,
			"origin": request.Origin,
			"intent": request.Intent,
			"auditlogfunction": "facade.AuditLog()",
		}).Info(request.Message)
	//
	//plog.WithFields(
	//	log.Fields{
	//		"audit":  "AUDIT-PLOG",
	//		"user":   request.User,
	//		"origin": request.Origin,
	//		"intent": request.Intent,
	//	}).Info(request.Message)
	return nil
}

func (f *Facade) LogAudit(ctx datastore.Context, msg string) {
	fmt.Fprintf(os.Stderr, "Facade.LogAudit(%s)\n", msg)
	//f.setupAuditLogger()
	alog.WithFields(
		log.Fields{
			"audit": "AUDIT",
			"user":  ctx.GetUser(),
			"origin": ctx.GetOrigin(),
			"intent": ctx.GetIntention(),
			"auditlogfunction": "facade.LogAudit()",
		}).Info(msg)
}
