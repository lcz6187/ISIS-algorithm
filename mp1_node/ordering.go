package main

import (
	"bufio"
	"container/heap"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// Decentralized ISIS
var (
	localSeq       int = 0
	seqMu              = sync.Mutex{}
	nodeMsgCounter int = 0

	proposalResponses = make(map[string][]Proposal)
	proposalMu        sync.Mutex

	finalizedSet = make(map[string]bool)
	finalizedMu  sync.Mutex

	waitAgreedTimers = make(map[string]*time.Timer)
	waitTimerMu      sync.Mutex
)

// Proposals for ISIS
type Proposal struct {
	Seq  int
	Node string // Tiebreaker
}

func CleanupFailedNode(deadNode string) {
	// Given the ID of a failed node, remove all of its pending agreements
	// and such from the priority queue and remove its dependencies.

	proposalMu.Lock()
	defer proposalMu.Unlock()
	for msgID, proposals := range proposalResponses {
		var updated []Proposal
		for _, p := range proposals {
			if p.Node != deadNode {
				updated = append(updated, p)
			}
		}
		proposalResponses[msgID] = updated
	}
}

func handleIncomingMessage(message string) {
	// All incoming messages go through here
	// But they are handled differently

	fields := strings.Fields(message)
	if len(fields) < 1 {
		return
	}

	msgType := fields[0]
	switch msgType {
	// Bank Request
	case "REQUEST":
		// Format: REQUEST <msgId> <senderId> <startTime> <transaction...>
		if len(fields) < 4 {
			return
		}

		msgID := fields[1]
		sender := fields[2]

		// Parse the time, it should be a UnixNano string
		startTimeInt, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return
		}
		startTime := time.Unix(0, startTimeInt)

		transaction := strings.Join(fields[4:], " ")

		tx := Transaction{
			MsgID:     msgID,
			Payload:   transaction,
			StartTime: startTime,
		}

		processRequest(tx, sender)

	// ISIS Protocol
	case "PROPOSE":
		// Format: PROPOSE <msgId> <proposed> <proposer>
		if len(fields) < 4 {
			return
		}
		msgId := fields[1]
		proposed, err := strconv.Atoi(fields[2])
		if err != nil {
			return
		}

		proposer := fields[3]

		processPropose(msgId, proposed, proposer)

		// Only the node that originated the message collects proposals.
		// if strings.HasPrefix(msgId, nodeID+"_") {
		// 	processPropose(msgId, proposed, proposer)
		// }

	// ISIS Protocol
	case "AGREED":
		// Format: AGREED <msgId> <finalSeq> <finalProposer> <sender>
		if len(fields) < 5 {
			return
		}

		msgId := fields[1]
		finalSeq, err := strconv.Atoi(fields[2])
		if err != nil {
			return
		}

		finalProposer := fields[3]
		sender := fields[4]

		processAgreed(msgId, finalSeq, finalProposer, sender)

	// A node announces that they have connected to all the other nodes.
	// Once all nodes have announced this, then transactions start flowing.
	case "READY":
		// Format: READY <senderId>
		if len(fields) < 2 {
			return
		}

		sender := fields[1]

		readyMu.Lock()
		if !readySet[sender] {
			readySet[sender] = true
		}

		if len(readySet) == len(nodes)-1 {
			close(readyComplete)
		}

		readyMu.Unlock()
	default:
		// Unknown message type or malformatted message.
		// Ignore.
	}
}

func processRequest(tx Transaction, sender string) {
	// Increment local logical clock (essentially Lamport style).
	seqMu.Lock()
	localSeq++
	proposed := localSeq
	seqMu.Unlock()

	// Push it onto the priority queue.
	// Follows the ISIS protocol.
	item := &QueueItem{
		MsgID:       tx.MsgID,
		Transaction: tx,
		Proposal:    Proposal{Seq: proposed, Node: nodeID},
		Deliverable: false,
	}
	pqMu.Lock()
	heap.Push(&pq, item)
	pqMu.Unlock()

	// if sender == nodeID {
	// 	proposalMu.Lock()
	// 	proposalResponses[tx.MsgID] = []Proposal{}
	// 	proposalMu.Unlock()
	// }

	processPropose(tx.MsgID, proposed, nodeID)

	// Send a PROPOSE message back to the sender.

	if sender != nodeID {
		proposeMsg := "PROPOSE " + tx.MsgID + " " + strconv.Itoa(proposed) + " " + nodeID
		sendToNode(sender, proposeMsg)
	}

	startWaitForAgreedTimer(tx.MsgID)
}

func startWaitForAgreedTimer(msgID string) {
	waitTimerMu.Lock()
	// If there's already a timer, don't double create
	if _, exists := waitAgreedTimers[msgID]; exists {
		waitTimerMu.Unlock()
		return
	}
	waitAgreedTimers[msgID] = time.AfterFunc(5*time.Second, func() {
		handleAgreedTimeout(msgID)
	})
	waitTimerMu.Unlock()
}

func handleAgreedTimeout(msgID string) {
	finalizedMu.Lock()
	if finalizedSet[msgID] {
		// Already finalized, no need to drop
		finalizedMu.Unlock()
		return
	}
	finalizedMu.Unlock()

	proposalMu.Lock()
	proposals := proposalResponses[msgID]
	proposalMu.Unlock()

	// Build a set of nodes that responded.
	respondedNodes := make(map[string]bool)
	for _, p := range proposals {
		respondedNodes[p.Node] = true
	}

	// Determine which nodes are considered alive.
	aliveNodes := getAliveNodes()

	// If the number of responses is less than expected,
	// mark the missing nodes as dead.
	if len(proposals) < len(aliveNodes) {
		for _, node := range aliveNodes {
			if !respondedNodes[node] {
				log.Println("Timeout for msgID", msgID, ": no response from", node, "marking it as dead.")
				markNodeAsDead(node)
			}
		}
	}

	// Remove from PQ
	pqMu.Lock()
	for i, it := range pq {
		if it.MsgID == msgID {
			heap.Remove(&pq, i)
			break
		}
	}
	pqMu.Unlock()

	// Mark finalized in the sense "we're done with it," so we don’t reinsert or re-wait.
	finalizedMu.Lock()
	finalizedSet[msgID] = true
	finalizedMu.Unlock()

	waitTimerMu.Lock()
	delete(waitAgreedTimers, msgID)
	waitTimerMu.Unlock()
}

func processPropose(msgID string, proposed int, proposer string) {
	proposalMu.Lock()
	proposals := proposalResponses[msgID]
	proposals = append(proposals, Proposal{Seq: proposed, Node: proposer})
	proposalResponses[msgID] = proposals
	proposalMu.Unlock()

	// If we have proposals from all alive nodes, finalize
	if len(proposals) == len(getAliveNodes()) {
		finalizeAgreement(msgID)
	}
}

func finalizeAgreement(msgID string) {
	// Mark as finalized if not already
	finalizedMu.Lock()
	if finalizedSet[msgID] {
		finalizedMu.Unlock()
		return
	}
	finalizedSet[msgID] = true
	finalizedMu.Unlock()

	// Pick the largest <seq, node>
	proposalMu.Lock()
	props := proposalResponses[msgID]
	proposalMu.Unlock()

	finalSeq := 0
	finalNode := ""
	for _, p := range props {
		if p.Seq > finalSeq || (p.Seq == finalSeq && p.Node > finalNode) {
			finalSeq = p.Seq
			finalNode = p.Node
		}
	}

	// Ensure local clock is >= finalSeq
	seqMu.Lock()
	if localSeq < finalSeq {
		localSeq = finalSeq
	}
	seqMu.Unlock()

	// Multicast: AGREED <msgID> <finalSeq> <finalNode>
	agreedMsg := "AGREED " + msgID + " " + strconv.Itoa(finalSeq) + " " + finalNode + " " + nodeID
	multicast(agreedMsg)

	// Process locally
	processAgreed(msgID, finalSeq, finalNode, nodeID)
}

func processAgreed(msgID string, finalSeq int, finalProposer string, sender string) {
	// If we already have finalized/dropped it, do nothing
	finalizedMu.Lock()
	alreadyFinal := finalizedSet[msgID]
	finalizedMu.Unlock()
	if alreadyFinal && (sender != nodeID) {
		return
	}

	// Mark as finalized
	finalizedMu.Lock()
	finalizedSet[msgID] = true
	finalizedMu.Unlock()

	// Cancel the wait-for-AGREED timer if it's still running
	waitTimerMu.Lock()
	if t, ok := waitAgreedTimers[msgID]; ok {
		t.Stop()
		delete(waitAgreedTimers, msgID)
	}
	waitTimerMu.Unlock()

	// Bump our local clock if necessary
	seqMu.Lock()
	if localSeq < finalSeq {
		localSeq = finalSeq
	}
	seqMu.Unlock()

	// Fix the final seq/priority in the PQ item
	pqMu.Lock()
	for i, it := range pq {
		if it.MsgID == msgID {
			it.Proposal = Proposal{Seq: finalSeq, Node: finalProposer}
			it.Deliverable = true
			heap.Fix(&pq, i)
			break
		}
	}
	pqMu.Unlock()

	deliverMessages()

	// Re-multicast the same AGREED in case the origin has crashed
	// and some nodes still need to see it
	go multicast("AGREED " + msgID + " " + strconv.Itoa(finalSeq) + " " + finalProposer + " " + nodeID)
}

func deliverMessages() {
	// Pop and deliver messages from pq in order.

	pqMu.Lock()
	defer pqMu.Unlock()
	for pq.Len() > 0 {
		item := pq[0]
		if item.Deliverable {
			heap.Pop(&pq)
			// "Enter the critical section" and process.
			// Mirrors the mutual exclusion from the lecture slides.
			// "AccessResource"
			enterCriticalSection(item.Transaction)
		} else {
			break
		}
	}
}

func readTransactions() {
	// Read from standard input, multicasts a request, and then processes the local copy.
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		payload := scanner.Text()
		nodeMsgCounter++
		// TOOD: Fix the inconsistency of my Id and ID capitalization...
		// Driving me crazy... but I don't know which one to choose...
		msgID := nodeID + "_" + strconv.Itoa(nodeMsgCounter)

		// Create a transaction with current timestamp
		tx := Transaction{
			MsgID:     msgID,
			Payload:   payload,
			StartTime: time.Now(),
		}

		// Store transaction
		txMu.Lock()
		transactions[msgID] = tx
		txMu.Unlock()

		// This is getting super complicated.
		// Move to using rpc/gob or grpc/proto when possible.
		requestMsg := "REQUEST " + tx.MsgID + " " + nodeID + " " +
			strconv.FormatInt(tx.StartTime.UnixNano(), 10) + " " + tx.Payload
		multicast(requestMsg)
		processRequest(tx, nodeID)
	}
}
