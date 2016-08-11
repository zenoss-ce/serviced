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

package web

import (
	"fmt"
	"net/url"

	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

type statMetadata struct {
	StatID string
	Label  string
	Unit   string
}

var hostsMetadata = []statMetadata{
	{"mem", "Memory", "B"},
	{"cpu", "CPU", "pct"},
	{"docker_storage", "Docker Storage", "B"},
}

var mastersMetadata = []statMetadata{
	{"mem", "Memory", "B"},
	{"cpu", "CPU", "pct"},
	{"docker_storage", "Docker Storage", "B"},
	{"dfs_storage", "DFS Storage", "B"},
}

var backupsMetadata = []statMetadata{
	{"size", "Size", "B"},
}

var poolsMetadata = []statMetadata{
	{"mem", "Memory", "B"},
	{"cpu", "CPU", "pct"},
}

var isvcsMetadata = []statMetadata{
	{"mem", "Memory", "B"},
	{"cpu", "CPU", "pct"},
	{"size", "Size", "B"},
}

var availableMetadatas = map[string][]statMetadata{
	"hosts":   hostsMetadata,
	"masters": mastersMetadata,
	"backups": backupsMetadata,
	"pools":   poolsMetadata,
	"isvcs":   isvcsMetadata,
}

func restGetStatsMeta(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	entity, err := url.QueryUnescape(r.PathParam("entity"))
	if err != nil {
		restBadRequest(w, fmt.Errorf("Missing entity name"))
		return
	}

	metadata, ok := availableMetadatas[entity]
	if !ok {
		restBadRequest(w, fmt.Errorf("No stat metadata for entity %s", entity))
	}

	glog.V(4).Infof("restGetStatsMeta: entity %s, metadata: %#v", entity, metadata)
	writeJSON(w, metadata, 200)
}
