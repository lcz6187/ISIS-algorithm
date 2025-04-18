package main

import (
	"log"
	"net"
	"sync"
	"time"
)

// Transaction struct to carry MsgID with the transaction
// Useful for instrumentation
type Transaction struct {
	MsgID     string
	Payload   string
	StartTime time.Time
}

// Instrumentation (for the graph)
// This heavily slows down performance, so use with caution.
// No longer can handle 10000+ requests per second.
var (
	transactions = make(map[string]Transaction)
	txMu         sync.Mutex
)

// Global Configuration
var (
	// Node identification and other node tracking information
	nodeID        string
	configFile    string
	nodes         map[string]NodeConfig
	connections   map[string]net.Conn
	connectionsMu sync.Mutex

	// https://gobyexample.com/waitgroups
	// Use a waitgroup to wait until all goroutines finish
	startupWg       sync.WaitGroup
	startupComplete bool

	// Wait for all nodes to be ready and startedup
	readyMu       sync.Mutex
	readySet      = make(map[string]bool)
	readyComplete = make(chan struct{}) // Close this when all READY messages are received
)

// Node Configuration from the config file
type NodeConfig struct {
	ID   string
	Host string
	Port string
}

// Bank state
var (
	accounts = make(map[string]int)
	bankMu   sync.Mutex
)

var (
	aliveMu sync.Mutex
	alive   = make(map[string]bool)
)

func isAlive(node string) bool {
	aliveMu.Lock()
	defer aliveMu.Unlock()

	return alive[node]
}

func markNodeAsDead(node string) {
	aliveMu.Lock()
	defer aliveMu.Unlock()

	if alive[node] {
		log.Println("Marking", node, "as dead.")

		alive[node] = false

		if conn, ok := connections[node]; ok {
			conn.Close()
			delete(connections, node)
		}

		CleanupFailedNode(node)

		log.Println("Node", node, "has been marked dead")
	}
}

func getAliveNodes() []string {
	aliveMu.Lock()
	defer aliveMu.Unlock()

	var result []string
	for id, a := range alive {
		if a {
			result = append(result, id)
		}
	}

	return result
}
