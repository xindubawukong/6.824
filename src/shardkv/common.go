package shardkv

import (
	"log"

	"6.824/shardctrler"
)

//
// Sharded key/value server.
// Lots of replica groups, each running Raft.
// Shardctrler decides which group serves each shard.
// Shardctrler may change shard assignment from time to time.
//
// You will have to modify these definitions.
//

const shardKvDebugging = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if shardKvDebugging {
		log.Printf(format, a...)
	}
	return
}

const NShards = shardctrler.NShards

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongGroup  = "ErrWrongGroup"
	ErrWrongLeader = "ErrWrongLeader"
)

type Err string

// Put or Append
type PutAppendArgs struct {
	ClientId string
	OpId     string
	Key      string
	Value    string
	Op       string
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	ClientId string
	OpId     string
	Key      string
}

type GetReply struct {
	Err   Err
	Value string
}

type UpdateConfigArgs struct {
	ClientId string
	OpId     string
	Config   shardctrler.Config
}

type PushShardDataArgs struct {
	ShardStatus ShardStatus
}

type PushShardDataReply struct {
	Err Err
}

type DeleteShardDataArgs struct {
	Shard int
	Version int
}
