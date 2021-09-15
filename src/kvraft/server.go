package kvraft

import (
	"bytes"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
)

func DPrintf(format string, a ...interface{}) (n int, err error) {
	raft.DPrintf(format, a...)
	return
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	Op       string // "Get", "Put" or "Append"
	Key      string
	Value    string
	ClientId string
	OpId     string
}

type Response struct {
	OpId  string
	Err   string
	Value string
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	kvMap           map[string]string
	result          map[string]chan string
	responseHistory map[string]Response
}

func (kv *KVServer) getSnapshot() []byte {
	kv.mu.Lock()
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(kv.kvMap)
	e.Encode(kv.responseHistory)
	data := w.Bytes()
	kv.mu.Unlock()
	return data
}

func (kv *KVServer) readFromSnapshot(data []byte) {
	if data == nil || len(data) < 1 {
		return
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var kvMap map[string]string
	var responseHistory map[string]Response
	if d.Decode(&kvMap) != nil ||
		d.Decode(&responseHistory) != nil {
		DPrintf("Decode error\n")
	} else {
		kv.mu.Lock()
		kv.kvMap = kvMap
		kv.responseHistory = responseHistory
		kv.mu.Unlock()
	}
}

func (kv *KVServer) getResultChannel(term, index int) chan string {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	key := fmt.Sprintf("%d,%d", term, index)
	ch, ok := kv.result[key]
	if ok {
		return ch
	}
	ch = make(chan string, 1)
	kv.result[key] = ch
	return ch
}

func (kv *KVServer) deleteResultChannel(term, index int) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	key := fmt.Sprintf("%d,%d", term, index)
	_, ok := kv.result[key]
	if ok {
		delete(kv.result, key)
	}
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	var op Op
	op.Op = "Get"
	op.Key = args.Key
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	index, term, isLeader := kv.rf.Start(op)
	if !isLeader {
		reply.Err = ErrWrongLeader
	} else {
		ch := kv.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			ch <- args.OpId + ":timeout"
		}()
		rawResult := <-ch
		tmp := strings.Split(rawResult, ":")
		opId := tmp[0]
		opRes := tmp[1]
		if opId == args.OpId {
			if opRes == "timeout" {
				reply.Err = ErrWrongLeader
			} else if opRes == "nokey" {
				reply.Err = ErrNoKey
			} else {
				reply.Err = OK
				reply.Value = opRes
			}
			kv.deleteResultChannel(term, index)
		} else {
			reply.Err = ErrWrongLeader
		}
	}
	if isLeader {
		DPrintf("\n")
		DPrintf("KvServer %d Get finished. index: %d\n", kv.me, index)
		DPrintf("Args: %v\n", *args)
		DPrintf("Reply: %v\n", *reply)
	}
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// DPrintf("KvServer %d PutAppend start.\n", kv.me)
	// DPrintf("Args: %v\n", *args)
	// DPrintf("Reply: %v\n", *reply)
	// Your code here.
	var op Op
	op.Op = args.Op
	op.Key = args.Key
	op.Value = args.Value
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	index, term, isLeader := kv.rf.Start(op)
	if !isLeader {
		reply.Err = ErrWrongLeader
	} else {
		ch := kv.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			ch <- args.OpId + ":timeout"
		}()
		rawResult := <-ch
		DPrintf("rawResult: %v\n", rawResult)
		tmp := strings.Split(rawResult, ":")
		opId := tmp[0]
		opRes := tmp[1]
		if opId == args.OpId {
			if opRes == "timeout" {
				reply.Err = ErrWrongLeader
			} else {
				reply.Err = OK
			}
			kv.deleteResultChannel(term, index)
		} else {
			reply.Err = ErrWrongLeader
		}
	}
	if isLeader {
		DPrintf("\n")
		DPrintf("KvServer %d PutAppend finished. index: %d\n", kv.me, index)
		DPrintf("Args: %v\n", *args)
		DPrintf("Reply: %v\n", *reply)
	}
}

//
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

func (kv *KVServer) startApply() {
	lastAppliedIndex := 0
	for msg := range kv.applyCh {
		if kv.killed() {
			break
		}
		term, isLeader := kv.rf.GetState()
		DPrintf("KVServer %d commit msg: %v. term: %d, isLeader: %v\n", kv.me, msg, term, isLeader)
		if msg.CommandValid {
			index := msg.CommandIndex
			if index <= lastAppliedIndex {
				continue
			}
			lastAppliedIndex = index
			var ch chan string
			if isLeader {
				ch = kv.getResultChannel(term, index)
			}
			op := msg.Command.(Op)
			lastResponse, ok := kv.responseHistory[op.ClientId]
			if ok && lastResponse.OpId == op.OpId {
				if isLeader {
					if lastResponse.Err == OK {
						ch <- lastResponse.OpId + ":" + lastResponse.Value
					} else {
						ch <- lastResponse.OpId + ":" + lastResponse.Err
					}
				}
			} else {
				kv.mu.Lock()
				var response Response
				response.OpId = op.OpId
				if op.Op == "Put" {
					kv.kvMap[op.Key] = op.Value
					response.Err = OK
				} else if op.Op == "Append" {
					value, ok := kv.kvMap[op.Key]
					if !ok {
						value = ""
					}
					kv.kvMap[op.Key] = value + op.Value
					response.Err = OK
				} else if op.Op == "Get" {
					value, ok := kv.kvMap[op.Key]
					if !ok {
						response.Err = "nokey"
					} else {
						response.Err = OK
						response.Value = value
					}
				}
				if isLeader {
					if response.Err == OK {
						ch <- response.OpId + ":" + response.Value
					} else {
						ch <- response.OpId + ":" + response.Err
					}
				}
				kv.responseHistory[op.ClientId] = response
				kv.mu.Unlock()
			}
			raftStateSize := kv.rf.GetStateSize()
			if kv.maxraftstate != -1 && raftStateSize > kv.maxraftstate {
				kv.rf.Snapshot(index, kv.getSnapshot())
			}
		} else if msg.SnapshotValid {
			if kv.rf.CondInstallSnapshot(msg.SnapshotTerm, msg.SnapshotIndex, msg.Snapshot) {
				kv.readFromSnapshot(msg.Snapshot)
			}
		}
		DPrintf("KVServer %d commit msg: %v end.\n", kv.me, msg)
	}
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	DPrintf("Start KVServer %d", me)

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.

	kv.kvMap = make(map[string]string)
	kv.result = make(map[string]chan string)
	kv.responseHistory = make(map[string]Response)

	kv.readFromSnapshot(kv.rf.GetSnapshot())

	go kv.startApply()

	return kv
}
