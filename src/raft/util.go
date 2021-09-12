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

func get_min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func get_max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func isMajority(cnt, tot int) bool {
	return cnt*2 > tot
}

type ServerStatus string

const LEADER ServerStatus = "LEADER"
const FOLLOWER ServerStatus = "FOLLOWER"
const CANDIDATE ServerStatus = "CANDIDATE"

func getRandomElectionTimeout() time.Duration {
	var l = 400
	var r = 800
	var t = int64(rand.Int()%(r-l)+l) * int64(time.Millisecond)
	return time.Duration(t)
}

const REQUEST_VOTE_TIME_OUT = 10000 * time.Millisecond
const APPEND_ENTRIES_TIME_OUT = 200 * time.Millisecond
const INSTALL_SNAPSHOT_TIME_OUT = 400 * time.Millisecond

func binarySearch(log []LogEntry, index int) int {
	if len(log) == 0 {
		return -1
	}
	t := index - log[0].Index
	if t < 0 || t >= len(log) {
		return -1
	}
	return t
	// l := 0
	// r := len(log) - 1
	// for l <= r {
	// 	m := (l + r) / 2
	// 	if index == log[m].Index {
	// 		return m
	// 	} else if index < log[m].Index {
	// 		r = m - 1
	// 	} else {
	// 		l = m + 1
	// 	}
	// }
	// return -1
}
