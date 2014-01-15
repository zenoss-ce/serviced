package web

import (
	"github.com/ant0ine/go-json-rest"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"

	"mime"
	"net/http"
	"net/url"
)

type ServiceConfig struct {
	bindPort   string
	agentPort  string
	zookeepers []string
}

func NewServiceConfig(bindPort string, agentPort string, zookeepers []string) *ServiceConfig {
	cfg := ServiceConfig{bindPort, agentPort, zookeepers}
	if len(cfg.bindPort) == 0 {
		cfg.bindPort = ":8787"
	}
	if len(cfg.agentPort) == 0 {
		cfg.agentPort = "127.0.0.1:4979"
	}
	if len(cfg.zookeepers) == 0 {
		cfg.zookeepers = []string{"127.0.0.1:2181"}
	}
	return &cfg
}

func (this *ServiceConfig) Serve() {
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".woff", "application/font-woff")

	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
	}
	routes := []rest.Route{
		rest.Route{"GET", "/", MainPage},
		rest.Route{"GET", "/test", TestPage},
		// Hosts
		rest.Route{"GET", "/hosts", this.AuthorizedClient(RestGetHosts)},
		rest.Route{"POST", "/hosts/add", this.AuthorizedClient(RestAddHost)},
		rest.Route{"DELETE", "/hosts/:hostId", this.AuthorizedClient(RestRemoveHost)},
		rest.Route{"PUT", "/hosts/:hostId", this.AuthorizedClient(RestUpdateHost)},
		rest.Route{"GET", "/hosts/:hostId/running", this.AuthorizedClient(RestGetRunningForHost)},
		rest.Route{"DELETE", "/hosts/:hostId/:serviceStateId", this.AuthorizedClient(RestKillRunning)},
		// Pools
		rest.Route{"POST", "/pools/add", this.AuthorizedClient(RestAddPool)},
		rest.Route{"GET", "/pools/:poolId/hosts", this.AuthorizedClient(RestGetHostsForResourcePool)},
		rest.Route{"DELETE", "/pools/:poolId", this.AuthorizedClient(RestRemovePool)},
		rest.Route{"PUT", "/pools/:poolId", this.AuthorizedClient(RestUpdatePool)},
		rest.Route{"GET", "/pools", this.AuthorizedClient(RestGetPools)},
		// Services (Apps)
		rest.Route{"GET", "/services", this.AuthorizedClient(RestGetAllServices)},
		rest.Route{"GET", "/services/:serviceId", this.AuthorizedClient(RestGetService)},
		rest.Route{"GET", "/services/:serviceId/running", this.AuthorizedClient(RestGetRunningForService)},
		rest.Route{"GET", "/services/:serviceId/running/:serviceStateId", this.AuthorizedClient(RestGetRunningService)},
		rest.Route{"GET", "/services/:serviceId/:serviceStateId/logs", this.AuthorizedClient(RestGetServiceStateLogs)},
		rest.Route{"POST", "/services/add", this.AuthorizedClient(RestAddService)},
		rest.Route{"DELETE", "/services/:serviceId", this.AuthorizedClient(RestRemoveService)},
		rest.Route{"GET", "/services/:serviceId/logs", this.AuthorizedClient(RestGetServiceLogs)},
		rest.Route{"PUT", "/services/:serviceId", this.AuthorizedClient(RestUpdateService)},
		// Service templates (App templates)
		rest.Route{"GET", "/templates", this.AuthorizedClient(RestGetAppTemplates)},
		rest.Route{"POST", "/templates/deploy", this.AuthorizedClient(RestDeployAppTemplate)},
		// Login
		rest.Route{"POST", "/login", RestLogin},
		rest.Route{"DELETE", "/login", RestLogout},
		// "Misc" stuff
		rest.Route{"GET", "/top/services", this.AuthorizedClient(RestGetTopServices)},

		rest.Route{"GET", "/running", this.AuthorizedClient(RestGetAllRunning)},
		// Generic static data
		rest.Route{"GET", "/favicon.ico", FavIcon},
		rest.Route{"GET", "/static*resource", StaticData},
	}

	// Hardcoding these target URLs for now.
	// TODO: When internal services are allowed to run on other hosts, look that up.
	routes = routeToInternalServiceProxy("/elastic", "http://127.0.0.1:9200/", routes)
	routes = routeToInternalServiceProxy("/metrics", "http://127.0.0.1:8888/", routes)

	handler.SetRoutes(routes...)

	http.ListenAndServe(this.bindPort, &handler)
}

var methods []string = []string{"GET", "POST", "PUT", "DELETE"}

func routeToInternalServiceProxy(path string, target string, routes []rest.Route) []rest.Route {
	targetUrl, err := url.Parse(target)
	if err != nil {
		glog.Errorf("Unable to parse proxy target URL: %s", target)
		return routes
	}
	// Wrap the normal http.Handler in a rest.HandlerFunc
	handlerFunc := func(w *rest.ResponseWriter, r *rest.Request) {
		proxy := serviced.NewReverseProxy(path, targetUrl)
		proxy.ServeHTTP(w.ResponseWriter, r.Request)
	}
	// Add on a glob to match subpaths
	andsubpath := path + "*x"
	for _, method := range methods {
		routes = append(routes, rest.Route{method, path, handlerFunc})
		routes = append(routes, rest.Route{method, andsubpath, handlerFunc})
	}
	return routes
}

func (this *ServiceConfig) AuthorizedClient(realfunc HandlerClientFunc) HandlerFunc {
	return func(w *rest.ResponseWriter, r *rest.Request) {
		if !LoginOk(r) {
			RestUnauthorized(w)
			return
		}
		client, err := this.getClient()
		if err != nil {
			glog.Errorf("Unable to acquire client: %v", err)
			RestServerError(w)
			return
		}
		defer client.Close()
		realfunc(w, r, client)
	}
}

func (this *ServiceConfig) getClient() (c *serviced.ControlClient, err error) {
	// setup the client
	c, err = serviced.NewControlClient(this.agentPort)
	if err != nil {
		glog.Fatalf("Could not create a control plane client: %v", err)
	}
	return c, err
}
