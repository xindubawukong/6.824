package kvraft

import (
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
	Op string // "Get", "Put" or "Append"
	Key string
	Value string
	ClientId string
	OpId string
}

type Response struct {
	opId string
	err string
	value string
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	kvMap	map[string]string
	result map[int]chan string
	responseHistory map[string] Response
}

func (kv *KVServer) getResultChannel(index int) chan string {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	ch, ok := kv.result[index]
	if ok {
		return ch
	}
	ch = make(chan string)
	kv.result[index] = ch
	return ch
}

func (kv *KVServer) deleteResultChannel(index int) {
	kv.mu.Lock()
	defer kv.mu.Unlock()
	_, ok := kv.result[index]
	if ok {
		delete(kv.result, index)
	}
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	var op Op
	op.Op = "Get"
	op.Key = args.Key
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		reply.Err = ErrWrongLeader
	} else {
		ch := kv.getResultChannel(index)
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
			} else if opRes == "no_key" {
				reply.Err = ErrNoKey
			} else {
				reply.Err = OK
				reply.Value = opRes
			}
			kv.deleteResultChannel(index)
		} else {
			reply.Err = ErrWrongLeader
		}
	}
	// if isLeader {
	// 	DPrintf("\n")
	// 	DPrintf("KvServer %d Get finished.\n", kv.me)
	// 	DPrintf("Args: %v\n", *args)
	// 	DPrintf("Reply: %v\n", *reply)
	// }
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	var op Op
	op.Op = args.Op
	op.Key = args.Key
	op.Value = args.Value
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		reply.Err = ErrWrongLeader
	} else {
		ch := kv.getResultChannel(index)
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
			} else {
				reply.Err = OK
			}
			kv.deleteResultChannel(index)
		} else {
			reply.Err = ErrWrongLeader
		}
	}
	// if isLeader {
	// 	DPrintf("\n")
	// 	DPrintf("KvServer %d PutAppend finished.\n", kv.me)
	// 	DPrintf("Args: %v\n", *args)
	// 	DPrintf("Reply: %v\n", *reply)
	// }
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
	for msg := range(kv.applyCh) {
		if kv.killed() {
			break
		}
		_, isLeader := kv.rf.GetState()
		if msg.CommandValid {
			index := msg.CommandIndex
			var ch chan string
			if isLeader {
				ch = kv.getResultChannel(index)
			}
			op := msg.Command.(Op)
			// if kv.rf.IsLeader() {
			// 	DPrintf("commit op: %v\n", op)
			// }
			lastResponse, ok := kv.responseHistory[op.ClientId]
			if ok && lastResponse.opId == op.OpId {
				if lastResponse.err == OK {
					ch <- lastResponse.opId + ":" + lastResponse.value
				} else {
					ch <- lastResponse.opId + ":" + lastResponse.err
				}
			} else {
				kv.mu.Lock()
				var response Response
				response.opId = op.OpId
				if op.Op == "Put" {
					kv.kvMap[op.Key] = op.Value
					response.err = OK
				} else if op.Op == "Append" {
					value, ok := kv.kvMap[op.Key]
					if !ok {
						value = ""
					}
					kv.kvMap[op.Key] = value + op.Value
					response.err = OK
				} else if op.Op == "Get" {
					value, ok := kv.kvMap[op.Key]
					if !ok {
						response.err = "nokey"
					} else {
						response.err = OK
						response.value = value
					}
				}
				if isLeader {
					if response.err == OK {
						ch <- response.opId + ":" + response.value
					} else {
						ch <- response.opId + ":" + response.err
					}
				}
				kv.responseHistory[op.ClientId] = response
				kv.mu.Unlock()
			}
		}
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
	kv.result = make(map[int]chan string)
	kv.responseHistory = make(map[string]Response)

	go kv.startApply()

	return kv
}
