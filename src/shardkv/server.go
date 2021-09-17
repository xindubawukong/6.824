package shardkv

import (
	"bytes"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
	"6.824/shardctrler"
)

type ShardStatus struct {
	Shard         int
	Status        string // NO_DATA, READY, SERVING, PUSHING
	Version       int
	Data          map[string]string
	SendTo        int
	ResultHistory map[string]ApplyResult
}

func (status *ShardStatus) Copy() ShardStatus {
	var res ShardStatus
	res.Shard = status.Shard
	res.Status = status.Status
	res.Version = status.Version
	res.Data = make(map[string]string)
	for k, v := range status.Data {
		res.Data[k] = v
	}
	res.SendTo = status.SendTo
	res.ResultHistory = make(map[string]ApplyResult)
	for k, v := range status.ResultHistory {
		res.ResultHistory[k] = v
	}
	return res
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
	config           shardctrler.Config
	firstConfig      shardctrler.Config
	shardCtrlerClerk *shardctrler.Clerk
	shards           [NShards]ShardStatus
	knownMaxVersion  [NShards]int
	resultChannel    map[string]chan ApplyResult
	opCnt            int64
}

func (kv *ShardKV) getSnapshot() []byte {
	kv.mu.Lock()
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(kv.config)
	e.Encode(kv.firstConfig)
	e.Encode(kv.shards)
	e.Encode(kv.knownMaxVersion)
	data := w.Bytes()
	kv.mu.Unlock()
	return data
}

func (kv *ShardKV) readFromSnapshot(data []byte) {
	if data == nil || len(data) < 1 {
		return
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var config shardctrler.Config
	var firstConfig shardctrler.Config
	var shards [NShards]ShardStatus
	var knownMaxVersion [NShards]int
	if d.Decode(&config) != nil || d.Decode(&firstConfig) != nil || d.Decode(&shards) != nil || d.Decode(&knownMaxVersion) != nil {
		DPrintf("Decode error\n")
	} else {
		kv.mu.Lock()
		kv.config = config.Copy()
		kv.firstConfig = firstConfig.Copy()
		for i := 0; i < NShards; i++ {
			kv.shards[i] = shards[i].Copy()
		}
		kv.knownMaxVersion = knownMaxVersion
		kv.mu.Unlock()
	}
}

type Op struct {
	OpType              string
	ClientId            string
	OpId                string
	PutAppendArgs       *PutAppendArgs
	GetArgs             *GetArgs
	UpdateConfigArgs    *UpdateConfigArgs
	PushShardDataArgs   *PushShardDataArgs
	DeleteShardDataArgs *DeleteShardDataArgs
	NeedResult          bool
}

type ApplyResult struct {
	OpType               string
	ClientId             string
	OpId                 string
	PutAppendReply       *PutAppendReply
	GetReply             *GetReply
	PushShardDataReply   *PushShardDataReply
	DeleteShardDataReply *DeleteShardDataReply
}

func (res *ApplyResult) isSuccess() bool {
	if res.PutAppendReply != nil {
		return res.PutAppendReply.Err == OK
	}
	if res.GetReply != nil {
		return res.GetReply.Err == OK
	}
	if res.PushShardDataReply != nil {
		return res.PushShardDataReply.Err == OK
	}
	return false
}

func (kv *ShardKV) getMyClientId() string {
	return fmt.Sprintf("ShardKV_gid[%d]_me[%d]", kv.gid, kv.me)
}

func (kv *ShardKV) getMyNewOpId() string {
	tmp := atomic.AddInt64(&kv.opCnt, 1)
	return fmt.Sprintf("%v_Op[%d]", kv.getMyClientId(), tmp)
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
	var op Op
	op.OpType = "Get"
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	op.GetArgs = args
	op.NeedResult = true
	index, term, isLeader := kv.rf.Start(op)
	if !isLeader {
		reply.Err = ErrWrongLeader
	} else {
		ch := kv.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			var result ApplyResult
			result.OpType = op.OpType
			result.ClientId = op.ClientId
			result.OpId = op.OpId
			result.GetReply = &GetReply{}
			result.GetReply.Err = ErrWrongLeader
			ch <- result
		}()
		var result = <-ch
		if result.OpId == op.OpId {
			reply.Err = result.GetReply.Err
			reply.Value = result.GetReply.Value
			kv.deleteResultChannel(term, index)
		} else {
			reply.Err = ErrWrongLeader
		}
	}
}

func (kv *ShardKV) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	var op Op
	op.OpType = "PutAppend"
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	op.PutAppendArgs = args
	op.NeedResult = true
	index, term, isLeader := kv.rf.Start(op)
	if !isLeader {
		reply.Err = ErrWrongLeader
	} else {
		ch := kv.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			var result ApplyResult
			result.OpType = op.OpType
			result.ClientId = op.ClientId
			result.OpId = op.OpId
			result.PutAppendReply = &PutAppendReply{}
			result.PutAppendReply.Err = ErrWrongLeader
			ch <- result
		}()
		var result = <-ch
		if result.OpId == op.OpId {
			reply.Err = result.PutAppendReply.Err
			kv.deleteResultChannel(term, index)
		} else {
			reply.Err = ErrWrongLeader
		}
	}
}

func (kv *ShardKV) UpdateConfig(args *UpdateConfigArgs) {
	var op Op
	op.OpType = "UpdateConfig"
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	op.UpdateConfigArgs = args
	op.NeedResult = false
	kv.rf.Start(op)
}

func (kv *ShardKV) ReceiveShardData(args *PushShardDataArgs) PushShardDataReply {
	var op Op
	op.OpType = "ReceiveShardData"
	op.ClientId = kv.getMyClientId()
	op.OpId = kv.getMyNewOpId()
	op.PushShardDataArgs = args
	op.NeedResult = true
	index, term, isLeader := kv.rf.Start(op)
	var reply PushShardDataReply
	if !isLeader {
		reply.Err = ErrWrongLeader
	} else {
		ch := kv.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			var result ApplyResult
			result.OpType = op.OpType
			result.ClientId = op.ClientId
			result.OpId = op.OpId
			result.PushShardDataReply = &PushShardDataReply{}
			result.PushShardDataReply.Err = "timeout"
			ch <- result
		}()
		var result = <-ch
		if result.OpId == op.OpId {
			reply.Err = result.PushShardDataReply.Err
			kv.deleteResultChannel(term, index)
		} else {
			reply.Err = "error"
		}
	}
	return reply
}

func (kv *ShardKV) DeleteShardData(shard int, version int) DeleteShardDataReply {
	var op Op
	op.OpType = "DeleteShardData"
	op.ClientId = kv.getMyClientId()
	op.OpId = kv.getMyNewOpId()
	op.DeleteShardDataArgs = &DeleteShardDataArgs{}
	op.DeleteShardDataArgs.Shard = shard
	op.DeleteShardDataArgs.Version = version
	op.NeedResult = true
	index, term, isLeader := kv.rf.Start(op)
	var reply DeleteShardDataReply
	if !isLeader {
		reply.Err = ErrWrongLeader
	} else {
		ch := kv.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			var result ApplyResult
			result.OpType = op.OpType
			result.ClientId = op.ClientId
			result.OpId = op.OpId
			result.DeleteShardDataReply = &DeleteShardDataReply{}
			result.DeleteShardDataReply.Err = "timeout"
			ch <- result
		}()
		var result = <-ch
		if result.OpId == op.OpId {
			reply.Err = result.DeleteShardDataReply.Err
			kv.deleteResultChannel(term, index)
		} else {
			reply.Err = "error"
		}
	}
	return reply
}

func (kv *ShardKV) SendPushShardData(shard int) {
	var args PushShardDataArgs
	kv.mu.Lock()
	if kv.shards[shard].Status == "PUSHING" {
		args.ShardStatus = kv.shards[shard].Copy()
		args.ConfigNum = kv.config.Num
	}
	kv.mu.Unlock()
	gid := args.ShardStatus.SendTo
	if gid == 0 || gid == kv.gid {
		return
	}
	for {
		kv.mu.Lock()
		servers, ok := kv.config.Groups[gid]
		kv.mu.Unlock()
		if ok {
			for si := 0; si < len(servers); si++ {
				server := kv.make_end(servers[si])
				var reply PushShardDataReply
				ok := server.Call("ShardKV.PushShardData", &args, &reply)
				if ok && reply.Err == OK {
					kv.DeleteShardData(shard, args.ShardStatus.Version)
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (kv *ShardKV) PushShardData(args *PushShardDataArgs, reply *PushShardDataReply) {
	res := kv.ReceiveShardData(args)
	reply.Err = res.Err
}

func (kv *ShardKV) applyGet(args *GetArgs) GetReply {
	var reply GetReply
	key := args.Key
	shard := key2shard(key)
	if kv.shards[shard].Status != "SERVING" {
		reply.Err = ErrWrongGroup
	} else {
		value, ok := kv.shards[shard].Data[key]
		if !ok {
			reply.Err = ErrNoKey
		} else {
			reply.Err = OK
			reply.Value = value
		}
	}
	return reply
}

func (kv *ShardKV) applyPutAppend(args *PutAppendArgs) PutAppendReply {
	var reply PutAppendReply
	shard := key2shard(args.Key)
	if kv.shards[shard].Status != "SERVING" {
		reply.Err = ErrWrongGroup
	} else {
		kv.shards[shard].Version++
		kv.knownMaxVersion[shard] = kv.shards[shard].Version
		if args.Op == "Put" {
			kv.shards[shard].Data[args.Key] = args.Value
		} else if args.Op == "Append" {
			value, ok := kv.shards[shard].Data[args.Key]
			if !ok {
				value = ""
			}
			kv.shards[shard].Data[args.Key] = value + args.Value
		} else {
			DPrintf("applyPutAppend error! args: %v\n", args)
		}
		reply.Err = OK
	}
	return reply
}

func (kv *ShardKV) applyUpdateConfig(args *UpdateConfigArgs) {
	newConfig := args.Config
	if newConfig.Num > kv.config.Num {
		kv.config = newConfig.Copy()
	}
	if kv.firstConfig.Num != 1 && args.FirstConfig.Num == 1 {
		kv.firstConfig = args.FirstConfig.Copy()
	}
	// For initial data
	if kv.firstConfig.Num == 1 {
		for i := 0; i < NShards; i++ {
			if kv.firstConfig.Shards[i] == kv.gid && kv.shards[i].Version == 0 && kv.knownMaxVersion[i] == -1 {
				kv.shards[i].Status = "READY"
				kv.knownMaxVersion[i] = 0
			}
		}
	}
	for i := 0; i < NShards; i++ {
		if kv.shards[i].Status == "NO_DATA" {
			// Do nothing
		} else if kv.shards[i].Status == "READY" {
			if kv.config.Shards[i] == kv.gid {
				kv.shards[i].Status = "SERVING"
			} else {
				kv.shards[i].Status = "PUSHING"
				kv.shards[i].SendTo = kv.config.Shards[i]
				if kv.rf.IsLeader() {
					go kv.SendPushShardData(i)
				}
			}
		} else if kv.shards[i].Status == "SERVING" {
			if kv.config.Shards[i] != kv.gid {
				kv.shards[i].Status = "PUSHING"
				kv.shards[i].SendTo = kv.config.Shards[i]
				if kv.rf.IsLeader() {
					go kv.SendPushShardData(i)
				}
			}
		} else if kv.shards[i].Status == "PUSHING" {
			if kv.rf.IsLeader() {
				go kv.SendPushShardData(i)
			}
		} else {
			DPrintf("Error! status: %v\n", kv.shards[i].Status)
		}
	}
}

func (kv *ShardKV) applyReceiveShardData(args *PushShardDataArgs) PushShardDataReply {
	var reply PushShardDataReply
	shard := args.ShardStatus.Shard
	version := args.ShardStatus.Version
	if kv.knownMaxVersion[shard] >= version {
		reply.Err = OK
		return reply
	}
	kv.shards[shard] = args.ShardStatus.Copy()
	if kv.config.Shards[shard] == kv.gid {
		kv.shards[shard].Status = "SERVING"
	} else if args.ConfigNum >= kv.config.Num {
		kv.shards[shard].Status = "READY"
	} else {
		kv.shards[shard].Status = "PUSHING"
		kv.shards[shard].SendTo = kv.config.Shards[shard]
		if kv.rf.IsLeader() {
			kv.SendPushShardData(shard)
		}
	}
	kv.shards[shard].Version++
	kv.knownMaxVersion[shard] = version
	reply.Err = OK
	return reply
}

func (kv *ShardKV) applyDeleteShardData(args *DeleteShardDataArgs) DeleteShardDataReply {
	if kv.shards[args.Shard].Status == "PUSHING" && kv.shards[args.Shard].Version == args.Version {
		kv.shards[args.Shard].Status = "NO_DATA"
		kv.shards[args.Shard].Data = make(map[string]string)
	}
	var reply DeleteShardDataReply
	reply.Err = OK
	return reply
}

// Run inside lock
func (kv *ShardKV) applyOp(op Op) ApplyResult {
	var result ApplyResult
	result.OpType = op.OpType
	result.ClientId = op.ClientId
	result.OpId = op.OpId
	if op.OpType == "Get" {
		reply := kv.applyGet(op.GetArgs)
		result.GetReply = &reply
	} else if op.OpType == "PutAppend" {
		reply := kv.applyPutAppend(op.PutAppendArgs)
		result.PutAppendReply = &reply
	} else if op.OpType == "UpdateConfig" {
		kv.applyUpdateConfig(op.UpdateConfigArgs)
	} else if op.OpType == "ReceiveShardData" {
		reply := kv.applyReceiveShardData(op.PushShardDataArgs)
		result.PushShardDataReply = &reply
	} else if op.OpType == "DeleteShardData" {
		reply := kv.applyDeleteShardData(op.DeleteShardDataArgs)
		result.DeleteShardDataReply = &reply
	} else {
		fmt.Printf("ShardKV: Op %v is not supported!!", op)
	}
	return result
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

func (kv *ShardKV) startUpdateConfig() {
	var firstConfig shardctrler.Config
	tmp := 0
	for {
		tmp += 1
		if kv.rf.IsLeader() {
			for firstConfig.Num != 1 {
				firstConfig = kv.shardCtrlerClerk.Query(1)
				time.Sleep(10 * time.Millisecond)
			}
			config := kv.shardCtrlerClerk.Query(-1)
			kv.mu.Lock()
			if config.Num > kv.config.Num || tmp%10 == 0 {
				var args UpdateConfigArgs
				args.ClientId = kv.getMyClientId()
				args.OpId = kv.getMyNewOpId()
				args.Config = config.Copy()
				args.FirstConfig = firstConfig.Copy()
				kv.UpdateConfig(&args)
			}
			kv.mu.Unlock()
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (kv *ShardKV) startApply() {
	lastAppliedIndex := 0
	for msg := range kv.applyCh {
		term, isLeader := kv.rf.GetState()
		if msg.CommandValid {
			index := msg.CommandIndex
			if index <= lastAppliedIndex {
				continue
			}
			op := msg.Command.(Op)
			var ch chan ApplyResult
			if isLeader && op.NeedResult {
				ch = kv.getResultChannel(term, index)
			}
			shard := -1
			if op.OpType == "PutAppend" {
				shard = key2shard(op.PutAppendArgs.Key)
			}
			if op.OpType == "Get" {
				shard = key2shard(op.GetArgs.Key)
			}
			processed := false
			if shard != -1 {
				lastResult, ok := kv.shards[shard].ResultHistory[op.ClientId]
				if ok && lastResult.OpId == op.OpId {
					processed = true
					// duplicate op from client
					if isLeader {
						ch <- lastResult
					}
				}
			}
			if !processed {
				kv.mu.Lock()
				result := kv.applyOp(op)
				if isLeader && op.NeedResult {
					ch <- result
				}
				if isLeader {
					DPrintf("ShardKV me=%d gid=%d commit op: %v. term: %d, isLeader: %v, shard: %d, result: %v\nshard status: %v\n", kv.me, kv.gid, op.toString(), term, isLeader, shard, result.toString(), kv.getShardStatus())
				}
				if result.isSuccess() && shard != -1 {
					kv.shards[shard].ResultHistory[op.ClientId] = result
				}
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
	}
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
	DPrintf("StartKVServer, me: %d, gid: %d\n", me, gid)

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

	kv.config.Num = 0
	kv.config.Shards = [NShards]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	kv.config.Groups = make(map[int][]string)
	kv.shardCtrlerClerk = shardctrler.MakeClerk(ctrlers)
	for i := 0; i < NShards; i++ {
		kv.shards[i].Shard = i
		kv.shards[i].Status = "NO_DATA"
		kv.shards[i].Data = make(map[string]string)
		kv.shards[i].Version = 0
		kv.shards[i].ResultHistory = make(map[string]ApplyResult)
		kv.knownMaxVersion[i] = -1
	}
	kv.resultChannel = make(map[string]chan ApplyResult)

	kv.readFromSnapshot(kv.rf.GetSnapshot())

	go kv.startUpdateConfig()
	go kv.startApply()

	return kv
}
