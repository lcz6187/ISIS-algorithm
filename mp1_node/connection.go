package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"time"
)

func startServer() {
	config := nodes[nodeID]

	ln, err := net.Listen("tcp", ":"+config.Port)
	if err != nil {
		log.Println("Error starting server: ", err)
		os.Exit(1)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println("Error accepting connection: ", err)
			continue
		}

		go handleConnection(conn)
	}
}

func connectToNodes() {
	// This was causing problems somehow so I had to wrap it in a mutex lock
	connectionsMu.Lock()
	connections = make(map[string]net.Conn)
	connectionsMu.Unlock()

	// Add nodes to waitgroup.
	// However, I really don't like how its just a counter.
	// TODO: Find alternative that takes advantage of having all the node ids.
	otherNodes := 0
	for id := range nodes {
		if id != nodeID {
			otherNodes++
		}
	}
	startupWg.Add(otherNodes)

	for id, config := range nodes {
		if id == nodeID {
			continue
		}

		go func(id string, config NodeConfig) {
			// Attempt connection until successful
			for {
				conn, err := net.Dial("tcp", config.Host+":"+config.Port)
				if err != nil {
					// Keep trying until successful
					time.Sleep(100 * time.Millisecond)
					continue
				}

				connectionsMu.Lock()
				connections[id] = conn
				connectionsMu.Unlock()

				startupWg.Done()

				go handleConnection(conn)
				break
			}
		}(id, config)
	}
}

func handleConnection(conn net.Conn) {
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		message := scanner.Text()

		// Dispatch message to ordering
		handleIncomingMessage(message)
	}
	defer conn.Close()
}

func multicast(message string) {
	for node, conn := range connections {
		if node == nodeID || !isAlive(node) {
			continue
		}

		if _, err := fmt.Fprintln(conn, message); err != nil {
			markNodeAsDead(node)
		}
	}
}

func sendToNode(target string, message string) error {
	if !isAlive(target) {
		return fmt.Errorf("node %s is dead", target)
	}

	connectionsMu.Lock()
	conn, ok := connections[target]
	connectionsMu.Unlock()

	if !ok {
		// Not sure if this startupComplete is really necessary here with the new channel block
		// in the main function?
		// This is a relic of a previous implementation that I'm too scared to remove
		// for fear of breaking something that depends on it.
		if !startupComplete {
			return fmt.Errorf("connection to node %s not established yet", target)
		}

		markNodeAsDead(target)
		return fmt.Errorf("no connection to node %s", target)
	}

	// TODO: Reformat this if statement into a single line.
	_, err := fmt.Fprintln(conn, message)
	if err != nil {
		if startupComplete {
			markNodeAsDead(target)
		}
		return err
	}

	return nil
}

func broadcastReady() {
	msg := "READY " + nodeID
	multicast(msg)
}
