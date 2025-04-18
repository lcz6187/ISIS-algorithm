package main

import (
	"log"
	"os"
)

var procLog *os.File

func initProcLog() {
	var err error
	logFileName := nodeID + "_processing.log"
	procLog, err = os.OpenFile(logFileName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		log.Fatalf("Could not open processing log %s: %v", logFileName, err)
	}
}
