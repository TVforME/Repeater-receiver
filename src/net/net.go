package net

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/TVforME/Repeater-receiver/src/config"
	"github.com/TVforME/Repeater-receiver/src/state"
)

// AdapterStatus represents the lock status of an adapter
type AdapterLock struct {
	AdapterIndex int   `json:"index"`
	State        uint8 `json:"state"`
	Lock         bool  `json:"lock"`
	CarrierLock  bool  `json:"carrier_lock"`
}

// sendLockStats sends the lock status of all adapters as a JSON message
func SendLockStats(config config.NetworkConfig, stateMachines []*state.StateMachine) {
	ip := fmt.Sprintf("%s:%d", config.StatsIP, config.Port)

	var localAddr *net.UDPAddr
	var err error

	if config.InterfaceIP != "" {
		// Resolve the local network interface IP to send multicast on
		localAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", config.InterfaceIP, 0))
		if err != nil {
			log.Fatalf("Error resolving interface address: %v", err)
		}
	} else {
		localAddr = nil
	}

	statsAddr, err := net.ResolveUDPAddr("udp", ip)
	if err != nil {
		log.Fatalf("Error resolving Stats UDP address: %v", err)
	}

	conn, err := net.DialUDP("udp", localAddr, statsAddr)
	if err != nil {
		log.Fatalf("Error creating UDP connection: %v", err)
	}
	defer conn.Close()

	for {
		var statuses []AdapterLock
		for _, sm := range stateMachines {
			sm.Mu.Lock()

			/*
				fmt.Printf(
				"Adapter %d:
				State=%d,
				Lock=%v,
				CarrierLock=%v\n",
				sm.adapterIndex,
				sm.OsdStats.State,
				sm.OsdStats.Lock,
				sm.OsdStats.CarrierLock
				) // Debug info
			*/

			statuses = append(statuses, AdapterLock{
				AdapterIndex: sm.AdapterIndex,
				State:        sm.OsdStats.State,
				Lock:         sm.OsdStats.Lock,
				CarrierLock:  sm.OsdStats.CarrierLock,
			})
			sm.Mu.Unlock()
		}

		jsonMsg, err := json.Marshal(statuses)
		if err != nil {
			log.Printf("Error marshaling JSON: %v", err)
			continue
		}

		//fmt.Printf("JSON Message: %s\n", jsonMsg) // Debug info

		_, err = conn.Write(jsonMsg)
		if err != nil {
			log.Printf("Error sending UDP message: %v", err)
		}

		time.Sleep(1 * time.Second)
	}
}
