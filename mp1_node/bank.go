package main

import (
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"
	"time"
)

func enterCriticalSection(tx Transaction) {
	// Function to mirror "entering the critical section" mentioned in the notes
	// to explicitly handle the total ordered transaction.

	accessResource(tx.Payload)
	exitCriticalSection(tx)
}

// accessResource performs the bank update based on the transaction.
func accessResource(transaction string) {
	// Update the bank with the transaction

	// Transaction formats:
	//   "DEPOSIT <dst> <amount>" or
	//   "TRANSFER <src> -> <dst> <amount>"

	parts := strings.Fields(transaction)
	if len(parts) < 3 {
		return
	}

	bankMu.Lock()
	defer bankMu.Unlock()

	switch parts[0] {
	case "DEPOSIT":
		account := parts[1]
		amount, err := strconv.Atoi(parts[2])
		if err != nil {
			return
		}

		// TODO: Determine if this or to allow depositing negative amounts
		// but not letting the account go below 0
		if amount > 0 {
			accounts[account] += amount
		}

	case "TRANSFER":
		if len(parts) < 5 {
			return
		}

		src := parts[1]
		dst := parts[3]

		amount, err := strconv.Atoi(parts[4])
		if err != nil {
			return
		}

		if accounts[src] >= amount {
			accounts[src] -= amount
			accounts[dst] += amount
		}
	}
}

func exitCriticalSection(tx Transaction) {
	// Now we can safely print out the balances after we handled the balances "atomically"
	printBalances()

	// This is the perfect place to compute processing time
	processingTime := time.Since(tx.StartTime)
	// log.Printf("Transaction %s processed in %v\n", tx.MsgID, processingTime)
	_, err := fmt.Fprintf(procLog, "%s,%.6f\n", tx.MsgID, processingTime.Seconds())
	if err != nil {
		log.Printf("Error writing to log: %v", err)
	}
}

func printBalances() {
	var keys []string
	for account, balance := range accounts {
		if balance != 0 {
			keys = append(keys, account)
		}
	}

	// The output balances should be sorted
	sort.Strings(keys)
	output := "BALANCES"
	for _, account := range keys {
		output += " " + account + ":" + strconv.Itoa(accounts[account])
	}

	fmt.Println(output)
}
