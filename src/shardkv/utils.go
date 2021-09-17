package shardkv

import "fmt"

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
		return "no reply"
	}
	return fmt.Sprintf(
		"{OpType: %v, ClientId: %v, OpId: %v, reply: %v}",
		res.OpType, res.ClientId, res.OpId, f(res))
}

func (kv *ShardKV) getShardStatus() string {
	var s = "\n"
	for i := 0; i < NShards; i++ {
		s += fmt.Sprintf("shard: %d, status: %v, version: %d, data: %v\n", i, kv.shards[i].Status, kv.shards[i].Version, len(kv.shards[i].Data))
	}
	return s
}
