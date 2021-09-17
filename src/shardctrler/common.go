package shardctrler

import (
	"log"
)

//
// Shard controler: assigns shards to replication groups.
//
// RPC interface:
// Join(servers) -- add a set of groups (gid -> server-list mapping).
// Leave(gids) -- delete a set of groups.
// Move(shard, gid) -- hand off one shard from current owner to gid.
// Query(num) -> fetch Config # num, or latest config if num==-1.
//
// A Config (configuration) describes a set of replica groups, and the
// replica group responsible for each shard. Configs are numbered. Config
// #0 is the initial configuration, with no groups and all shards
// assigned to group 0 (the invalid group).
//
// You will need to add fields to the RPC argument structs.
//

const shardCtrlerDebugging = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if shardCtrlerDebugging {
		log.Printf(format, a...)
	}
	return
}

// The number of shards.
const NShards = 10

// A configuration -- an assignment of shards to groups.
// Please don't change this.
type Config struct {
	Num    int              // config number
	Shards [NShards]int     // shard -> gid
	Groups map[int][]string // gid -> servers[]
}

func (config *Config) Copy() Config {
	var newConfig Config
	newConfig.Num = config.Num
	newConfig.Shards = config.Shards
	newConfig.Groups = make(map[int][]string)
	for gid, servers := range config.Groups {
		tmp := make([]string, 0)
		tmp = append(tmp, servers...)
		newConfig.Groups[gid] = tmp
	}
	return newConfig
}

const (
	OK = "OK"
)

type Err string

type JoinArgs struct {
	Servers  map[int][]string // new GID -> servers mappings
	ClientId string
	OpId     string
}

type JoinReply struct {
	WrongLeader bool
	Err         Err
}

type LeaveArgs struct {
	GIDs     []int
	ClientId string
	OpId     string
}

type LeaveReply struct {
	WrongLeader bool
	Err         Err
}

type MoveArgs struct {
	Shard    int
	GID      int
	ClientId string
	OpId     string
}

type MoveReply struct {
	WrongLeader bool
	Err         Err
}

type QueryArgs struct {
	Num      int // desired config number
	ClientId string
	OpId     string
}

type QueryReply struct {
	WrongLeader bool
	Err         Err
	Config      Config
}
