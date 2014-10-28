package core

import (
	"fmt"
	"net"
	"./config/common"
	"sync"
)

var configManager *ConfigManager

func StartCoreService() {
	configManager = NewConfigManger()
	// start rest server

	// start listening to incoming TCP connection

	//
}

func GetNeighborConfiguration(peerAddr net.Addr) *NeighborConfiguration {

	nConfig := configManager.findNeighborConfiguration(peerAddr)
	return nConfig

}

func AddNeighborConfiguration(nConfig *NeighborConfiguration) error {

	peerAddr := nConfig.PeerAddress
	if c := configManager.findNeighborConfiguration(peerAddr); c != nil {
		return fmt.Errorf("Neighbor configuration exists.")
	}
	configManager.addNeighborConfiguration(nConfig)

}


