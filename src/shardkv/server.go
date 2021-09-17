package shardkv

import (
	"fmt"
	"sync"

	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
	"6.824/shardctrler"
)

type ShardStatus struct {
	status  string // NO_DATA, READY, SERVING, PUSHING
	version int
	data    map[string]string
}

type ShardKV struct {
	mu           sync.Mutex
	me           int
	rf           *raft.Raft
	applyCh      chan raft.ApplyMsg
	make_end     func(string) *labrpc.ClientEnd
	gid          int
	ctrlers      []*labrpc.ClientEnd
	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	config          shardctrler.Config
	shards          [NShards]ShardStatus
	knownMaxVersion [NShards]int
	resultChannel   map[string]chan ApplyResult
	resultHistory   map[string]ApplyResult
}

type Op struct {
	OpType        string
	ClientId      string
	OpId          string
	PutAppendArgs PutAppendArgs
	GetArgs       GetArgs
	NeedResult    bool
}

type ApplyResult struct {
	OpType         string
	ClientId       string
	OpId           string
	PutAppendReply PutAppendReply
	GetReply       GetReply
}

func (kv *ShardKV) getResultChannel(term, index int) chan ApplyResult {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	key := fmt.Sprintf("%d,%d", term, index)
	ch, ok := kv.resultChannel[key]
	if ok {
		return ch
	}
	ch = make(chan ApplyResult, 1)
	kv.resultChannel[key] = ch
	return ch
}

func (kv *ShardKV) deleteResultChannel(term, index int) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	key := fmt.Sprintf("%d,%d", term, index)
	_, ok := kv.resultChannel[key]
	if ok {
		delete(kv.resultChannel, key)
	}
}

func (kv *ShardKV) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
}

func (kv *ShardKV) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
}

//
// the tester calls Kill() when a ShardKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *ShardKV) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.
}

//
// servers[] contains the ports of the servers in this group.
//
// me is the index of the current server in servers[].
//
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
//
// the k/v server should snapshot when Raft's saved state exceeds
// maxraftstate bytes, in order to allow Raft to garbage-collect its
// log. if maxraftstate is -1, you don't need to snapshot.
//
// gid is this group's GID, for interacting with the shardctrler.
//
// pass ctrlers[] to shardctrler.MakeClerk() so you can send
// RPCs to the shardctrler.
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs. You'll need this to send RPCs to other groups.
//
// look at client.go for examples of how to use ctrlers[]
// and make_end() to send RPCs to the group owning a specific shard.
//
// StartServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int, gid int, ctrlers []*labrpc.ClientEnd, make_end func(string) *labrpc.ClientEnd) *ShardKV {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := new(ShardKV)
	kv.me = me
	kv.maxraftstate = maxraftstate
	kv.make_end = make_end
	kv.gid = gid
	kv.ctrlers = ctrlers

	// Your initialization code here.

	// Use something like this to talk to the shardctrler:
	// kv.mck = shardctrler.MakeClerk(kv.ctrlers)

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	return kv
}
