package main

import (
	"sync"
	"time"
)

type CurrentFail struct {
	FailStart time.Time
	FailCount int
	OkCount   int
	// probably need some sort of uniq ID for the fail (rand?)

	// optional "payload"
	RelatedTask *Task
}

var (
	currentFails      map[string]*CurrentFail
	currentFailsMutex sync.Mutex
)

func CurrentFailsCreate() {
	currentFails = make(map[string]*CurrentFail)
}

func CurrentFailDelete(hash string) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	delete(currentFails, hash)
}

func CurrentFailAdd(hash string, failedCheck *CurrentFail) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	currentFails[hash] = failedCheck
}

func CurrentFailInc(hash string) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	currentFails[hash].FailCount++
	currentFails[hash].OkCount = 0
}

func CurrentFailDec(hash string) {
	currentFailsMutex.Lock()
	defer currentFailsMutex.Unlock()
	currentFails[hash].OkCount++
}

func CurrentFailGetAndInc(hash string) *CurrentFail {
	cf, ok := currentFails[hash]
	if !ok {
		var cf CurrentFail
		cf.FailCount = 1
		cf.OkCount = 0
		cf.FailStart = time.Now()
		CurrentFailAdd(hash, &cf)
		return &cf
	}

	CurrentFailInc(hash)
	return cf
}

func CurrentFailGetAndDec(hash string) *CurrentFail {
	cf, ok := currentFails[hash]
	if !ok {
		return nil
	}
	CurrentFailDec(hash)
	return cf
}