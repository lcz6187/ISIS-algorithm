package main

import (
	"bufio"
	"log"
	"os"
	"strings"
)

// Lots of go snippets referenced from the pkg.go.dev standard library
// - String conversion (https://pkg.go.dev/strconv)

// Lots of go snippets referenced from the gobyexample website
// (especially concurrency things and channel buffering)
// - https://gobyexample.com/

func main() {
	if len(os.Args) < 3 {
		log.Println("Usage: ./mp1_node <identifier> <config file>")
		os.Exit(1)
	}

	nodeID = os.Args[1]
	configFile = os.Args[2]

	// Load the node configurations
	readConfig(configFile)

	// Initialize the processing log
	initProcLog()

	// Start tracking memberships
	for id := range nodes {
		alive[id] = true
	}

	// Listen for incoming connections
	go startServer()

	// Connect to the other nodes in the system
	connectToNodes()
	// Wait for all nodes to connect at least once
	startupWg.Wait()

	// Now announce this node is ready
	broadcastReady()

	// Special edge case if there is only a single node,
	// then we can never expect an incoming connection to
	// close the channel for us.
	if len(nodes) == 1 {
		close(readyComplete)
	}

	// https://gobyexample.com/closing-channels
	// Wait for all the ready messages from the other nodes
	<-readyComplete // This will block until the channel is closed

	// Get transactions from stdin
	readTransactions()
}

func readConfig(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		log.Println("Error opening config file:", err)
		os.Exit(1)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	// Num nodes, not necessary for us because we can easily use len(nodes)
	// with minimal overhead.
	scanner.Scan()

	nodes = make(map[string]NodeConfig)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) != 3 {
			// Malformatted config
			continue
		}

		id, host, port := parts[0], parts[1], parts[2]
		nodes[id] = NodeConfig{ID: id, Host: host, Port: port}
	}
}
