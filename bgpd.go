//bgpd.go
package main

import (
	"fmt"
	"gobgp/api"
	"gobgp/core"
	"time"
)

func main() {
	fmt.Println("config_manager_th start")
	core.StartCoreService()
	//go config_manager_th()
	api.Config_for_Rest()
	time.Sleep(10 * time.Minute)
}

/*
func config_manager_th() {
	//api.Config_for_Rest()
}
*/
