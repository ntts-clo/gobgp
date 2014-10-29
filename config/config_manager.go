// config_manager
package config

import (
//"net"
)

type NeighborConfiguration struct {
	UUID string
	/*
		ASNumber int
		PeerAddress       net.Addr
		IPv4Enable        bool
		IPv6Enable        bool
		VPNv4Enable       bool
		VPNv6Enable       bool
		RouteServerClient bool
	*/
	ASNumber          string
	PeerAddress       string
	IPv4Enable        string
	IPv6Enable        string
	VPNv4Enable       string
	VPNv6Enable       string
	RouteServerClient string
}

type GlobalConfiguration struct {
	ID string
	//ID net.Addr
	MyAS string
	//MyAS int
	HoldTime string
	//HoldTime uint16
	// need Capability

}

type ConfigManager struct {
	GlobalConfig    *GlobalConfiguration
	NeighborsConfig map[string]*NeighborConfiguration
}

func NewConfigManger() *ConfigManager {
	manager := &ConfigManager{}
	manager.GlobalConfig = &GlobalConfiguration{}
	manager.NeighborsConfig = make(map[string]*NeighborConfiguration)
	return manager
}

func (manager *ConfigManager) AddNeighborConfiguration(neighborConfig *NeighborConfiguration) bool {
	uuid := neighborConfig.UUID
	manager.NeighborsConfig[uuid] = neighborConfig
	for _, neighbor := range manager.NeighborsConfig {
		if manager.NeighborsConfig[uuid].PeerAddress == neighbor.PeerAddress {
			return false
		}
	}
	return true
}
func (manager *ConfigManager) FindAllNeighborConfiguration() *ConfigManager {
	return manager
}
func (manager *ConfigManager) FindNeighborConfiguration(uuid string) *NeighborConfiguration {

	conf, ok := manager.NeighborsConfig[uuid]
	if !ok {
		return nil
	} else {
		return conf
	}
}
func (manager *ConfigManager) UpdateNeighborConfiguration(neighborConfig *NeighborConfiguration) {
	uuid := neighborConfig.UUID
	_, ok := manager.NeighborsConfig[uuid]
	if !ok {
		manager.NeighborsConfig[uuid] = neighborConfig
	} else {
		//TODO handle duplication error
	}
}
func (manager *ConfigManager) DeleteNeighborConfiguration(uuid string) {
	delete(manager.NeighborsConfig, uuid)
}
func (manager *ConfigManager) SetGlobalConfiguration(gConfig *GlobalConfiguration) {
	manager.GlobalConfig = gConfig
	// TODO send notification to handlers
}
func (manager *ConfigManager) GetGlobalConfiguration() *GlobalConfiguration {
	return manager.GlobalConfig
	// TODO send notification to handlers
}
