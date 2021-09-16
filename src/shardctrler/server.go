package shardctrler

import (
	"fmt"
	"sync"
	"time"

	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
)

type ShardCtrler struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	// Your data here.
	configs       []Config                    // indexed by config num
	resultChannel map[string]chan ApplyResult // term,index -> result channel
	resultHistory map[string]ApplyResult      // clientId -> last apply result
}

type Op struct {
	OpType    string
	ClientId  string
	OpId      string
	JoinArgs  JoinArgs
	LeaveArgs LeaveArgs
	MoveArgs  MoveArgs
	QueryArgs QueryArgs
}

type ApplyResult struct {
	OpType     string
	ClientId   string
	OpId       string
	JoinReply  JoinReply
	LeaveReply LeaveReply
	MoveReply  MoveReply
	QueryReply QueryReply
}

func (sc *ShardCtrler) getResultChannel(term, index int) chan ApplyResult {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	key := fmt.Sprintf("%d,%d", term, index)
	ch, ok := sc.resultChannel[key]
	if ok {
		return ch
	}
	ch = make(chan ApplyResult, 1)
	sc.resultChannel[key] = ch
	return ch
}

func (sc *ShardCtrler) deleteResultChannel(term, index int) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	key := fmt.Sprintf("%d,%d", term, index)
	_, ok := sc.resultChannel[key]
	if ok {
		delete(sc.resultChannel, key)
	}
}

func (sc *ShardCtrler) Join(args *JoinArgs, reply *JoinReply) {
	var op Op
	op.OpType = "Join"
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	op.JoinArgs = *args
	index, term, isLeader := sc.rf.Start(op)
	if !isLeader {
		reply.WrongLeader = true
	} else {
		ch := sc.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			var result ApplyResult
			result.OpType = op.OpType
			result.ClientId = op.ClientId
			result.OpId = op.OpId
			result.JoinReply.WrongLeader = true
			result.JoinReply.Err = "timeout"
			ch <- result
		}()
		var result = <-ch
		if result.OpId == op.OpId {
			reply.Err = result.JoinReply.Err
			reply.WrongLeader = result.JoinReply.WrongLeader
			sc.deleteResultChannel(term, index)
		} else {
			reply.WrongLeader = true
		}
	}
}

func (sc *ShardCtrler) Leave(args *LeaveArgs, reply *LeaveReply) {
	var op Op
	op.OpType = "Leave"
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	op.LeaveArgs = *args
	index, term, isLeader := sc.rf.Start(op)
	if !isLeader {
		reply.WrongLeader = true
	} else {
		ch := sc.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			var result ApplyResult
			result.OpType = op.OpType
			result.ClientId = op.ClientId
			result.OpId = op.OpId
			result.LeaveReply.WrongLeader = true
			result.LeaveReply.Err = "timeout"
			ch <- result
		}()
		var result = <-ch
		if result.OpId == op.OpId {
			reply.Err = result.LeaveReply.Err
			reply.WrongLeader = result.LeaveReply.WrongLeader
			sc.deleteResultChannel(term, index)
		} else {
			reply.WrongLeader = true
		}
	}
}

func (sc *ShardCtrler) Move(args *MoveArgs, reply *MoveReply) {
	var op Op
	op.OpType = "Move"
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	op.MoveArgs = *args
	index, term, isLeader := sc.rf.Start(op)
	if !isLeader {
		reply.WrongLeader = true
	} else {
		ch := sc.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			var result ApplyResult
			result.OpType = op.OpType
			result.ClientId = op.ClientId
			result.OpId = op.OpId
			result.MoveReply.WrongLeader = true
			result.MoveReply.Err = "timeout"
			ch <- result
		}()
		var result = <-ch
		if result.OpId == op.OpId {
			reply.Err = result.MoveReply.Err
			reply.WrongLeader = result.MoveReply.WrongLeader
			sc.deleteResultChannel(term, index)
		} else {
			reply.WrongLeader = true
		}
	}
}

func (sc *ShardCtrler) Query(args *QueryArgs, reply *QueryReply) {
	var op Op
	op.OpType = "Query"
	op.ClientId = args.ClientId
	op.OpId = args.OpId
	op.QueryArgs = *args
	index, term, isLeader := sc.rf.Start(op)
	if !isLeader {
		reply.WrongLeader = true
	} else {
		ch := sc.getResultChannel(term, index)
		go func() {
			time.Sleep(500 * time.Millisecond)
			var result ApplyResult
			result.OpType = op.OpType
			result.ClientId = op.ClientId
			result.OpId = op.OpId
			result.QueryReply.WrongLeader = true
			result.QueryReply.Err = "timeout"
			ch <- result
		}()
		var result = <-ch
		if result.OpId == op.OpId {
			reply.Err = result.QueryReply.Err
			reply.WrongLeader = result.QueryReply.WrongLeader
			reply.Config = result.QueryReply.Config // deep copy config
			sc.deleteResultChannel(term, index)
		} else {
			reply.WrongLeader = true
		}
	}
}

func (sc *ShardCtrler) applyJoin(args JoinArgs) JoinReply {
	lastConfig := sc.configs[len(sc.configs)-1]
	var config Config = lastConfig.copy()
	config.Num++
	for k, v := range args.Servers {
		config.Groups[k] = v
	}
	config.Shards = reblance(config.Shards, config.Groups)
	sc.configs = append(sc.configs, config)
	var reply JoinReply
	reply.WrongLeader = false
	reply.Err = "success"
	return reply
}

func (sc *ShardCtrler) applyLeave(args LeaveArgs) LeaveReply {
	lastConfig := sc.configs[len(sc.configs)-1]
	var config Config = lastConfig.copy()
	config.Num++
	for _, gid := range args.GIDs {
		_, ok := config.Groups[gid]
		if ok {
			delete(config.Groups, gid)
		}
	}
	config.Shards = reblance(config.Shards, config.Groups)
	sc.configs = append(sc.configs, config)
	var reply LeaveReply
	reply.WrongLeader = false
	reply.Err = "success"
	return reply
}

func (sc *ShardCtrler) applyMove(args MoveArgs) MoveReply {
	var config Config = sc.configs[len(sc.configs)-1].copy()
	config.Num++
	config.Shards[args.Shard] = args.GID
	sc.configs = append(sc.configs, config)
	var reply MoveReply
	reply.WrongLeader = false
	reply.Err = "success"
	return reply
}

func (sc *ShardCtrler) applyQuery(args QueryArgs) QueryReply {
	var reply QueryReply
	if args.Num == -1 || args.Num > sc.configs[len(sc.configs)-1].Num {
		reply.Config = sc.configs[len(sc.configs)-1].copy()
	} else {
		reply.Config = sc.configs[args.Num].copy() // ?
	}
	reply.WrongLeader = false
	reply.Err = "success"
	return reply
}

// Run inside lock
func (sc *ShardCtrler) applyOp(op Op) ApplyResult {
	var result ApplyResult
	result.OpType = op.OpType
	result.ClientId = op.ClientId
	result.OpId = op.OpId
	if op.OpType == "Join" {
		result.JoinReply = sc.applyJoin(op.JoinArgs)
	} else if op.OpType == "Leave" {
		result.LeaveReply = sc.applyLeave(op.LeaveArgs)
	} else if op.OpType == "Move" {
		result.MoveReply = sc.applyMove(op.MoveArgs)
	} else if op.OpType == "Query" {
		result.QueryReply = sc.applyQuery(op.QueryArgs)
	} else {
		fmt.Println("ERROR!!!!!")
	}
	return result
}

//
// the tester calls Kill() when a ShardCtrler instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (sc *ShardCtrler) Kill() {
	sc.rf.Kill()
	// Your code here, if desired.
}

// needed by shardkv tester
func (sc *ShardCtrler) Raft() *raft.Raft {
	return sc.rf
}

func (sc *ShardCtrler) startApply() {
	lastAppliedIndex := 0
	for msg := range sc.applyCh {
		term, isLeader := sc.rf.GetState()
		if msg.CommandValid {
			index := msg.CommandIndex
			if index <= lastAppliedIndex {
				continue
			}
			var ch chan ApplyResult
			if isLeader {
				ch = sc.getResultChannel(term, index)
			}
			op := msg.Command.(Op)
			lastResult, ok := sc.resultHistory[op.ClientId]
			if ok && lastResult.OpId == op.OpId {
				// duplicate op from client
				if isLeader {
					ch <- lastResult
				}
			} else {
				sc.mu.Lock()
				result := sc.applyOp(op)
				if isLeader {
					ch <- result
					DPrintf("ShardCtrler %d commit msg: %v. term: %d, isLeader: %v, result: %v\n", sc.me, msg, term, isLeader, result)
				}
				sc.resultHistory[op.ClientId] = result
				sc.mu.Unlock()
			}
		}
	}
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant shardctrler service.
// me is the index of the current server in servers[].
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *ShardCtrler {
	sc := new(ShardCtrler)
	sc.me = me

	sc.configs = make([]Config, 1)
	sc.configs[0].Groups = map[int][]string{}

	labgob.Register(Op{})
	sc.applyCh = make(chan raft.ApplyMsg)
	sc.rf = raft.Make(servers, me, persister, sc.applyCh)

	// Your code here.
	sc.configs = make([]Config, 0)
	var config Config
	config.Num = 0
	config.Shards = [NShards]int{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
	config.Groups = make(map[int][]string)
	sc.configs = append(sc.configs, config)
	sc.resultChannel = make(map[string]chan ApplyResult)
	sc.resultHistory = make(map[string]ApplyResult)

	go sc.startApply()

	return sc
}
