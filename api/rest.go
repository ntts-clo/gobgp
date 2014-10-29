package api

import (
	"github.com/ant0ine/go-json-rest/rest"
	"gobgp/config"
	"gobgp/core"
	"gobgp/utils"
	//"net"
	"net/http"
	//"strconv"
	//"time"
	//"encoding/json"
)

func Config_for_Rest() {
	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
	}
	handler.SetRoutes(
		&rest.Route{"GET", "/route_server", GetRouteServer},
		&rest.Route{"POST", "/route_server", PostRouteServer},
		&rest.Route{"GET", "/neighbor", GetAllNeighbor},
		&rest.Route{"GET", "/neighbor/:uuid", GetNeighbor},
		&rest.Route{"POST", "/neighbor", PostNeighbor},
		&rest.Route{"PUT", "/neighbor/:uuid", PutNeighbor},
		&rest.Route{"DELETE", "/neighbor", DeleteAllNeighbor},
		&rest.Route{"DELETE", "/neighbor/:uuid", DeleteNeighbor},
		//&rest.Route{"GET", "/local_rib", GetAllLocalRib},
		//&rest.Route{"GET", "/local_rib/:uuid", GetLocalRib},
	)
	http.ListenAndServe(":"+utils.REST_PORT, &handler)
}

func GetRouteServer(w rest.ResponseWriter, r *rest.Request) {
	gConfig := core.GetGlobal()
	w.WriteJson(gConfig)
}
func PostRouteServer(w rest.ResponseWriter, r *rest.Request) {
	gConfig := config.GlobalConfiguration{}
	err := r.DecodeJsonPayload(&gConfig)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//gConfig.HoldTime = gConfig.HoldTime * 60
	core.SetGlobal(&gConfig)
	w.WriteJson(&gConfig)
}

func GetAllNeighbor(w rest.ResponseWriter, r *rest.Request) {
	manager := core.GetManager()
	neighbors := make([]config.NeighborConfiguration, len(manager.NeighborsConfig))
	i := 0
	for _, neighbor := range manager.NeighborsConfig {
		neighbors[i] = *neighbor
		i++
	}
	w.WriteJson(&neighbors)
}

func GetNeighbor(w rest.ResponseWriter, r *rest.Request) {
	uuid := r.PathParam("uuid")
	nConfig := core.GetNeighbor(uuid)
	w.WriteJson(nConfig)
}

func PostNeighbor(w rest.ResponseWriter, r *rest.Request) {
	nConfig := &config.NeighborConfiguration{}
	err := r.DecodeJsonPayload(nConfig)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	uuid := utils.Gen_uuid()
	nConfig.UUID = uuid
	core.AddNeighbor(nConfig)
	w.WriteJson(nConfig)
}
func PutNeighbor(w rest.ResponseWriter, r *rest.Request) {
	uuid := r.PathParam("uuid")
	nConfig := &config.NeighborConfiguration{}
	err := r.DecodeJsonPayload(nConfig)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	nConfig.UUID = uuid
	core.UpdateNeighbor(nConfig)
	w.WriteJson(nConfig)
}
func DeleteAllNeighbor(w rest.ResponseWriter, r *rest.Request) {
	manager := core.GetManager()
	manager.NeighborsConfig = make(map[string]*config.NeighborConfiguration)
	w.WriteHeader(http.StatusOK)
}

func DeleteNeighbor(w rest.ResponseWriter, r *rest.Request) {
	uuid := r.PathParam("uuid")
	core.DeleteNeighbor(uuid)
	w.WriteHeader(http.StatusOK)
}

/*
func GetAllLocalRib(w rest.ResponseWriter, r *rest.Request) {
	paths := make([]Path, len(lr.Path_map))
	i := 0
	for _, path := range lr.Path_map {
		paths[i] = *path
		i++
	}
	w.WriteJson(&paths)
}
func GetLocalRib(w rest.ResponseWriter, r *rest.Request) {
	neighbor_id := r.PathParam("neighbor_id")
	var path *Path
	if lr.Path_map[neighbor_id] != nil {
		path = &Path{}
		*path = *lr.Path_map[neighbor_id]
	}
	w.WriteJson(path)
}
*/
