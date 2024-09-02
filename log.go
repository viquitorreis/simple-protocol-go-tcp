package main

import (
	"log"
	"time"
)

func logger(msg string) {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.Print(msg)
}

func getTimeWithMicroseconds() string {
	loc, err := time.LoadLocation("America/Sao_Paulo")
	if err != nil {
		// Handle the error, maybe fallback to UTC or log the error
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02 15:04:05.000000")
}
