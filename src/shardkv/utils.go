package shardkv

import (
	"bytes"
	"fmt"

	"6.824/labgob"
	"6.824/shardctrler"
)

func (op *Op) toString() string {
	f := func(op *Op) string {
		if op.PutAppendArgs != nil {
			return fmt.Sprintf("%v", op.PutAppendArgs)
		}
		if op.GetArgs != nil {
			return fmt.Sprintf("%v", op.GetArgs)
		}
		if op.UpdateConfigArgs != nil {
			return fmt.Sprintf("%v", *op.UpdateConfigArgs)
		}
		if op.PushShardDataArgs != nil {
			return fmt.Sprintf("%v", *op.PushShardDataArgs)
		}
		if op.DeleteShardDataArgs != nil {
			return fmt.Sprintf("%v", *op.DeleteShardDataArgs)
		}
		return "no args"
	}
	return fmt.Sprintf(
		"{OpType: %v, ClientId: %v, OpId: %v, args: %v, NeedResult: %v}",
		op.OpType, op.ClientId, op.OpId, f(op), op.NeedResult)
}

func (res *ApplyResult) toString() string {
	f := func(res *ApplyResult) string {
		if res.PutAppendReply != nil {
			return fmt.Sprintf("%v", res.PutAppendReply.Err)
		}
		if res.GetReply != nil {
			return fmt.Sprintf("%v", res.GetReply.Err)
		}
		if res.PushShardDataReply != nil {
			return fmt.Sprintf("%v", res.PushShardDataReply.Err)
		}
		if res.DeleteShardDataReply != nil {
			return fmt.Sprintf("%v", res.DeleteShardDataReply.Err)
		}
		return "no reply"
	}
	return fmt.Sprintf(
		"{OpType: %v, ClientId: %v, OpId: %v, reply: %v}",
		res.OpType, res.ClientId, res.OpId, f(res))
}

func getSize2(t map[string]string) int {
	size := 0
	for k, v := range t {
		size += len(k)
		size += len(v)
	}
	return size
}

func getSize(t map[string]ApplyResult) int {
	size := 0
	for k, v := range t {
		size += len(k)
		size += len(v.ClientId)
		size += len(v.OpId)
		size += len(v.OpType)
		if v.GetReply != nil {
			size += len(v.GetReply.Value)
		}
	}
	return size
}

func getConfigSize(config shardctrler.Config) int {
	size := 4
	size += 40
	for _, v := range config.Groups {
		size += 4
		for _, s := range v {
			size += len(s)
		}
	}
	return size
}

func (kv *ShardKV) getTotal(t ShardStatus) int {
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(t)
	data := w.Bytes()
	return len(data)
}

func (kv *ShardKV) getShardStatus() string {
	var s = "\n"
	for i := 0; i < NShards; i++ {
		s += fmt.Sprintf("shard: %d, status: %v, version: %d, data: %v, ResultHistory: %v, tot: %v\n", i, kv.shards[i].Status, kv.shards[i].Version, getSize2(kv.shards[i].Data), getSize(kv.shards[i].ResultHistory), kv.getTotal(kv.shards[i]))
	}
	return s
}
