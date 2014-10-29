package core

import (
	"fmt"
	"gobgp/config"
	//"net"
	//"sync"
)

var configManager *config.ConfigManager

func StartCoreService() {
	configManager = config.NewConfigManger()
	// start rest server

	// start listening to incoming TCP connection

	//
}
func GetManager() *config.ConfigManager {

	manager := configManager.FindAllNeighborConfiguration()
	return manager

}
func GetNeighbor(uuid string) *config.NeighborConfiguration {

	nConfig := configManager.FindNeighborConfiguration(uuid)
	return nConfig

}

func AddNeighbor(nConfig *config.NeighborConfiguration) error {

	uuid := nConfig.UUID
	if c := configManager.FindNeighborConfiguration(uuid); c != nil {
		return fmt.Errorf("Neighbor configuration exists.")
	}
	if !configManager.AddNeighborConfiguration(nConfig) {
		return fmt.Errorf("Neighbor configuration doesn't exists [ipaddress].")
	}

	return nil
}

func UpdateNeighbor(nConfig *config.NeighborConfiguration) error {

	uuid := nConfig.UUID
	if c := configManager.FindNeighborConfiguration(uuid); c == nil {
		return fmt.Errorf("Neighbor configuration doesn't exists.")
	}
	configManager.UpdateNeighborConfiguration(nConfig)
	return nil
}
func DeleteNeighbor(uuid string) error {
	if c := configManager.FindNeighborConfiguration(uuid); c == nil {
		return fmt.Errorf("Neighbor configuration doesn't exists.")
	}
	return nil
}
func GetGlobal() *config.GlobalConfiguration {
	return configManager.GetGlobalConfiguration()
}
func SetGlobal(gConfig *config.GlobalConfiguration) {
	configManager.SetGlobalConfiguration(gConfig)
}
