// Copyright 2014 The Serviced Authors.
// Use of sc source code is governed by a

package web

import (
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
)

/*
{
  regions: {
    'us-west-2': {
      updated: 1462471847,
      maxVolume: 200,
      nodes: [
        { name: 'INTERNET' },
        { name: 'service' }
      ],
      connections: [
        {
          source: 'INTERNET',
          target: 'service',
          metrics: { total: 100, success: 95, error: 5 },
          streaming: 1
        }
      ]
    }
  }
}
*/

type Connection struct {
	Class   string             `json:"class,omitempty"`
	Source  string             `json:"source"`
	Target  string             `json:"target"`
	Metrics map[string]float64 `json:"metrics"`
	Notices []string           `json:"notices"`
}

type Node struct {
	Name        string       `json:"name"`
	Class       string       `json:"class,omitempty"`
	Renderer    string       `json:"renderer"`
	Updated     int64        `json:"updated,omitempty"`
	MaxVolume   int          `json:"maxVolume"`
	Connections []Connection `json:"connections"`
}

type GlobalNode struct {
	Node
	Nodes            []RegionNode `json:"nodes"`
	ServerUpdateTime int64        `json:"serverUpdateTime,omitempty"`
}

type RegionNode struct {
	Node
	Nodes   []Node `json:"nodes"`
	Updated int64  `json:"updated"`
}

func NewConnection() Connection {
	return Connection{Class: "normal", Metrics: map[string]float64{}}
}

func NewGlobalNode(name string) GlobalNode {
	return GlobalNode{
		Node{
			Name:        name,
			Renderer:    "global",
			Connections: []Connection{},
		},
		[]RegionNode{},
		time.Now().Unix() * 1000,
	}
}

func NewRegionNode(name string) RegionNode {
	return RegionNode{
		Node{
			Name:        name,
			Class:       "normal",
			Renderer:    "region",
			Connections: []Connection{},
		},
		[]Node{},
		time.Now().Unix() * 1000,
	}
}

func restGetVizceralHosts(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	facade := ctx.getFacade()
	dataCtx := ctx.getDatastoreContext()
	hosts, err := facade.GetHosts(dataCtx)
	if err != nil {
		glog.Error("Could not get hosts: ", err)
		restServerError(w, err)
		return
	}
	global := NewGlobalNode("edge")
	global.MaxVolume = 10

	internet := NewRegionNode("Internet")
	internet.Renderer = "region"
	global.Nodes = append(global.Nodes, internet)
	for _, host := range hosts {
		node := NewRegionNode(host.Name)
		global.Nodes = append(global.Nodes, node)
	}
	conn := NewConnection()
	conn.Source = "Internet"
	conn.Target = "devian"
	conn.Metrics["danger"] = 25.01
	conn.Metrics["normal"] = 504.123
	conn.Metrics["warning"] = 55.006
	global.Connections = append(global.Connections, conn)
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.WriteJson(global)
}
