package raft

import (
	"log"
	"math/rand"
	"time"
)

// Debugging
const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}


func isMajority(cnt, tot int) bool {
	return cnt * 2 > tot
}


type ServerStatus string

const LEADER ServerStatus = "LEADER"
const FOLLOWER ServerStatus = "FOLLOWER"
const CANDIDATE ServerStatus = "CANDIDATE"


func getRandomElectionTimeout() time.Duration {
	var l = 400
	var r = 800
	var t = int64(rand.Int() % (r - l) + l) * int64(time.Millisecond)
	return time.Duration(t)
}
