package main

import (
	"github.com/ant0ine/go-json-rest/rest"
	"log"
	"net/http"
	"net"
	"strconv"
	"time"
	"encoding/json"
)


func Config_for_Rest() {
	rserver := RServer{}
	neighbors := Neighbors{
		Store: map[string]*Neighbor{},
	}
	/*
		lrib := LocRib{
			Path_map: map[string]*Path{},
		}
	*/
	// test
	ma := make(map[string]*Path)
	ma["10.0.0.1"] = &Path{"10.0.0.1", "10.0.1.1", 10, 11, 12, "13", "com1"}
	ma["172.16.0.1"] = &Path{"172.16.0.1", "172.16.1.1", 20, 21, 22, "23", "com2"}
	ma["192.168.0.1"] = &Path{"192.168.0.1", "192.168.1.1", 30, 31, 32, "33", "com3"}
	lrib := LocRib{"aaa", ma}
	//
	handler := rest.ResourceHandler{
		EnableRelaxedContentType: true,
	}
	err := handler.SetRoutes(
	rest.RouteObjectMethod("GET", "/route_server", &rserver, "GetRouteServer"),
	rest.RouteObjectMethod("POST", "/route_server", &rserver, "PostRouteServer"),
	rest.RouteObjectMethod("DELETE", "/route_server", &rserver, "DeleteRouteServer"),
	rest.RouteObjectMethod("GET", "/neighbor", &neighbors, "GetAllNeighbor"),
	rest.RouteObjectMethod("GET", "/neighbor/:neighbor_id", &neighbors, "GetNeighbor"),
	rest.RouteObjectMethod("POST", "/neighbor", &neighbors, "PostNeighbor"),
	rest.RouteObjectMethod("PUT", "/neighbor/:neighbor_id", &neighbors, "PutNeighbor"),
	rest.RouteObjectMethod("DELETE", "/neighbor", &neighbors, "DeleteAllNeighbor"),
	rest.RouteObjectMethod("DELETE", "/neighbor/:neighbor_id", &neighbors, "DeleteNeighbor"),
	rest.RouteObjectMethod("GET", "/local_rib", &lrib, "GetAllLocalRib"),
	rest.RouteObjectMethod("GET", "/local_rib/:neighbor_id", &lrib, "GetLocalRib"),
	)
	if err != nil {
	log.Fatal(err)
	}
	log.Fatal(http.ListenAndServe(":"+REST_PORT, &handler))
}

func (rs *RServer) GetRouteServer(w rest.ResponseWriter, r *rest.Request) {
	w.WriteJson(rs)
}
func (rs *RServer) PostRouteServer(w rest.ResponseWriter, r *rest.Request) {
	rserver := RServer{}
	err := r.DecodeJsonPayload(&rserver)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	//c_rs <- rs
	gConfig := &GlobalConfiguration{}
	gConfig.ID = net.ResolveIPAddr(rserver.Router_id)
	gConfig.MyAS = strconv.Atoi(rserver.Local_as)
	gConfig.HoldTime = time.Second * 60
	SetGlobalConfiguration(gConfig)

	*rs = rserver
	w.WriteJson(&rserver)
}
func (rs *RServer) DeleteRouteServer(w rest.ResponseWriter, r *rest.Request) {
	rs = nil
	//c_rs <- rs
	w.WriteHeader(http.StatusOK)
}
func (n *Neighbors) GetAllNeighbor(w rest.ResponseWriter, r *rest.Request) {
	n.RLock()
	neighbors := make([]Neighbor, len(n.Store))
	i := 0
	for _, neighbor := range n.Store {
		neighbors[i] = *neighbor
		i++
	}
	n.RUnlock()
	w.WriteJson(&neighbors)
}

func (n *Neighbors) GetNeighbor(w rest.ResponseWriter, r *rest.Request) {
	neighbor_id := r.PathParam("neighbor_id")

	n.RLock()
	var neighbor *Neighbor
	if n.Store[neighbor_id] != nil {
		neighbor = &Neighbor{}
		*neighbor = *n.Store[neighbor_id]
	}
	n.RUnlock()
	if neighbor == nil {
		rest.NotFound(w, r)
		return
	}
	w.WriteJson(neighbor)
}

func (n *Neighbors) PostNeighbor(w rest.ResponseWriter, r *rest.Request) {
	neighbor := Neighbor{}
	err := r.DecodeJsonPayload(&neighbor)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	n.Lock()
	uuid := Gen_uuid()
	neighbor.Neighbor_id = uuid
	n.Store[uuid] = &neighbor
	n.Unlock()
	//c_ne <- n
	w.WriteJson(&neighbor)
}
func (n *Neighbors) PutNeighbor(w rest.ResponseWriter, r *rest.Request) {
	neighbor_id := r.PathParam("neighbor_id")
	n.Lock()
	if n.Store[neighbor_id] == nil {
		rest.NotFound(w, r)
		n.Unlock()
		return
	}
	neighbor := Neighbor{}
	err := r.DecodeJsonPayload(&neighbor)
	if err != nil {
		rest.Error(w, err.Error(), http.StatusInternalServerError)
		n.Unlock()
		return
	}
	neighbor.Neighbor_id = neighbor_id
	n.Store[neighbor_id] = &neighbor
	n.Unlock()
	//c_ne <- n
	w.WriteJson(&neighbor)
}
func (n *Neighbors) DeleteAllNeighbor(w rest.ResponseWriter, r *rest.Request) {
	n.Lock()
	n.Store = nil
	n.Unlock()
	//c_ne <- n
	w.WriteHeader(http.StatusOK)
}

func (n *Neighbors) DeleteNeighbor(w rest.ResponseWriter, r *rest.Request) {
	neighbor_id := r.PathParam("neighbor_id")
	n.Lock()
	delete(n.Store, neighbor_id)
	n.Unlock()
	//c_ne <- n
	w.WriteHeader(http.StatusOK)
}
func (lr *LocRib) GetAllLocalRib(w rest.ResponseWriter, r *rest.Request) {
	paths := make([]Path, len(lr.Path_map))
	i := 0
	for _, path := range lr.Path_map {
		paths[i] = *path
		i++
	}
	w.WriteJson(&paths)
}
func (lr *LocRib) GetLocalRib(w rest.ResponseWriter, r *rest.Request) {
	neighbor_id := r.PathParam("neighbor_id")
	var path *Path
	if lr.Path_map[neighbor_id] != nil {
		path = &Path{}
		*path = *lr.Path_map[neighbor_id]
	}
	/*
		if path == nil {
			rest.NotFound(w, r)
			return
		}
	*/
	w.WriteJson(path)
}
