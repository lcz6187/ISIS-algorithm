package main

import (
	"sync"
)

// Priority Queue implementation for use with ISIS
// Implemented with assistance from the example at https://pkg.go.dev/container/heap

// Priority queue pending message
type QueueItem struct {
	MsgID       string
	Transaction Transaction
	Proposal    Proposal
	Deliverable bool
	Index       int
}

// Priority queue for ISIS
type PriorityQueue []*QueueItem

func (pq PriorityQueue) Len() int {
	return len(pq)
}

func (pq PriorityQueue) Less(i, j int) bool {
	// Compare sequence number and use node id for tie breaking
	if pq[i].Proposal.Seq == pq[j].Proposal.Seq {
		return pq[i].Proposal.Node < pq[j].Proposal.Node
	}

	return pq[i].Proposal.Seq < pq[j].Proposal.Seq
}

func (pq PriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]

	pq[i].Index = i
	pq[j].Index = j
}

func (pq *PriorityQueue) Push(x interface{}) {
	n := len(*pq)

	item := x.(*QueueItem)
	item.Index = n

	*pq = append(*pq, item)
}

func (pq *PriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)

	item := old[n-1]
	item.Index = -1

	*pq = old[0 : n-1]

	return item
}

// Node global priority queue and mutex
var (
	pq   PriorityQueue
	pqMu sync.Mutex
)
