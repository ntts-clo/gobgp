// config_manager
package config

import (
	"net"
)

type NeighborConfiguration struct {

	ASNumber	int
	PeerAddress	net.Addr
	IPv4Enable	bool
	IPv6Enable	bool
	VPNv4Enable	bool
	VPNv6Enable	bool
	RouteServerClient	bool

}

type GlobalConfiguration struct {

	ID 			net.Addr
	MyAS		int
	HoldTime	uint16
	// need Capability

}

type ConfigManager struct {
	globalConfig *GlobalConfiguration
	neighborsConfig map[net.Addr]*NeighborConfiguration
}

func NewConfigManger() *ConfigManager {
	manager := &ConfigManager{}
	manager.globalConfig = &GlobalConfiguration{}
	manager.neighborsConfig = make(map[net.Addr]*NeighborConfiguration)
	return manager
}

func (manager *ConfigManager) addNeighborConfiguration (neighborConfig *NeighborConfiguration){
	addr := neighborConfig.PeerAddress
	_, ok := manager.neighborsConfig[addr]
	if !ok {
		manager.neighborsConfig[addr] = neighborConfig
	} else {
		//TODO handle duplication error
	}
}

func (manager *ConfigManager) findNeighborConfiguration(addr net.Addr) NeighborConfiguration {

	conf, ok := manager.neighborsConfig[addr]
	if !ok {
		return nil
	} else {
		return conf
	}
}

func (manager *ConfigManager) setGlobalConfiguration(gConfig *GlobalConfiguration){
	manager.globalConfig = gConfig
	// TODO send notification to handlers
}

