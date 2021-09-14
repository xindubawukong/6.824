package kvraft

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"6.824/labrpc"
)

type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	clientId   string
	opCnt      int64
	lastLeader int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	// You'll have to add code here.
	ck.clientId = fmt.Sprintf("Clerk[%d]", nrand())
	return ck
}

func (ck *Clerk) getNextOpId() string {
	tmp := atomic.AddInt64(&ck.opCnt, 1)
	return fmt.Sprintf("%v_Op[%d]", ck.clientId, tmp)
}

//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) Get(key string) string {
	DPrintf(">> Clerk Get start, key: %v\n", key)
	// You will have to modify this function.
	var args GetArgs
	args.Key = key
	args.ClientId = ck.clientId
	args.OpId = ck.getNextOpId()
	var reply GetReply
	for {
		ok := ck.servers[ck.lastLeader].Call("KVServer.Get", &args, &reply)
		if !ok {
			ck.lastLeader = (ck.lastLeader + 1) % len(ck.servers)
			time.Sleep(50 * time.Millisecond)
		} else if reply.Err == ErrWrongLeader {
			ck.lastLeader = (ck.lastLeader + 1) % len(ck.servers)
		} else {
			break
		}
	}
	DPrintf(">> Clerk Get end, key: %v, reply: %v\n", key, reply)
	if reply.Err == OK {
		return reply.Value
	} else {
		return ""
	}
}

//
// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	DPrintf(">> Clerk PutAppend start, key: %v, value: %v\n", key, value)
	// You will have to modify this function.
	var args PutAppendArgs
	args.Key = key
	args.Value = value
	args.Op = op
	args.ClientId = ck.clientId
	args.OpId = ck.getNextOpId()
	var reply PutAppendReply
	for {
		DPrintf("lastLeader: %d\n", ck.lastLeader)
		ok := ck.servers[ck.lastLeader].Call("KVServer.PutAppend", &args, &reply)
		if !ok {
			ck.lastLeader = (ck.lastLeader + 1) % len(ck.servers)
			time.Sleep(50 * time.Millisecond)
		} else if reply.Err == ErrWrongLeader {
			ck.lastLeader = (ck.lastLeader + 1) % len(ck.servers)
		} else {
			break
		}
	}
	DPrintf(">> Clerk PutAppend end, key: %v, value: %v\n", key, value)
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}
