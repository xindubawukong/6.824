# 6.824

Author: xindubawukong@gmail.com

https://pdos.csail.mit.edu/6.824/schedule.html

This repo is for Spring 2021.

```
$ go version
go version go1.15.14 darwin/amd64
```

```
$ git diff 04a0ed2d03ffa3ad7b92d51507587ef44152f726 --name-only
.gitignore
README.md
learn_go/temp
learn_go/temp.go
learn_go/temp.py
lectures/crawler.go
lectures/kv.go
src/go.mod
src/go.sum
src/kvraft/client.go
src/kvraft/common.go
src/kvraft/server.go
src/mr/coordinator.go
src/mr/rpc.go
src/mr/worker.go
src/raft/.gitignore
src/raft/go-test-many.sh
src/raft/raft.go
src/raft/util.go
src/shardctrler/client.go
src/shardctrler/common.go
src/shardctrler/reblance.go
src/shardctrler/server.go
src/shardkv/client.go
src/shardkv/common.go
src/shardkv/server.go
src/shardkv/utils.go
```

## Lab 1: MapReduce

https://pdos.csail.mit.edu/6.824/labs/lab-mr.html

非常简单，worker周期性向coordinator拉任务即可。

```
$ cd src/main
$ go build -race -buildmode=plugin ../mrapps/wc.go
$ bash test-mr.sh
```

## Lab 2: Raft

https://pdos.csail.mit.edu/6.824/labs/lab-raft.html

参照论文，以及这个guide：https://thesquareplanet.com/blog/students-guide-to-raft/

实现细节：
- 对于snapshot，为了方便对log数组相关的操作，我把log[0]存着snapshot的最后一个entry的term和index。因此log的长度至少是1。
- 对nextIndex减一的优化，论文里没有详细讲明白，因此我采用了一种可能会多发送一些entries的方式，但最多不会超过应该发送的两倍，并且只需试错一次
  - 在AppendEntries的返回里，从不匹配的位置往前开始采样，间隔分别为1，2，4，8，16...。leader拿到返回值后，就可以找到最靠后的match的log，然后再发一次即可。
- 注意installSnapshot返回成功了也不一定就成功安装了snapshot。因此不能更新matchIndex。installSnapshot返回成功后，会继续尝试appendEntries来sync log并更新matchIndex。

```
$ cd src/raft
$ time go test -race
Test (2A): initial election ...
  ... Passed --   3.6  3   52   16758    0
Test (2A): election after network failure ...
  ... Passed --   4.5  3   88   18572    0
Test (2A): multiple elections ...
  ... Passed --   6.6  7  564  120022    0
Test (2B): basic agreement ...
  ... Passed --   0.8  3   14    4518    3
Test (2B): RPC byte count ...
  ... Passed --   1.8  3   48  116562   11
Test (2B): agreement despite follower disconnection ...
  ... Passed --   5.7  3  117   36060    8
Test (2B): no agreement if too many followers disconnect ...
  ... Passed --   3.8  5  197   46218    4
Test (2B): concurrent Start()s ...
  ... Passed --   1.1  3   18    5512    6
Test (2B): rejoin of partitioned leader ...
  ... Passed --   6.5  3  185   47626    4
Test (2B): leader backs up quickly over incorrect follower logs ...
  ... Passed --  19.6  5 1876  907823  103
Test (2B): RPC counts aren't too high ...
  ... Passed --   2.1  3   34   11558   12
Test (2C): basic persistence ...
  ... Passed --   4.6  3   79   22875    6
Test (2C): more persistence ...
  ... Passed --  19.2  5  983  238959   17
Test (2C): partitioned leader and one follower crash, leader restarts ...
  ... Passed --   2.0  3   35   10146    4
Test (2C): Figure 8 ...
  ... Passed --  38.4  5  835  193838   47
Test (2C): unreliable agreement ...
  ... Passed --   4.2  5  551  195481  246
Test (2C): Figure 8 (unreliable) ...
  ... Passed --  50.2  5 5692 4294951  351
Test (2C): churn ...
  ... Passed --  16.6  5 1411  492628  312
Test (2C): unreliable churn ...
  ... Passed --  16.2  5 1866  745714  401
Test (2D): snapshots basic ...
  ... Passed --   4.6  3  219   86584  251
Test (2D): install snapshots (disconnect) ...
  ... Passed --  54.9  3 1219  387646  359
Test (2D): install snapshots (disconnect+unreliable) ...
  ... Passed --  66.4  3 1525  448042  363
Test (2D): install snapshots (crash) ...
  ... Passed --  39.0  3  829  273622  399
Test (2D): install snapshots (unreliable+crash) ...
  ... Passed --  42.9  3  873  264984  322
PASS
ok      6.824/raft      415.706s
go test -race  82.97s user 18.03s system 24% cpu 6:57.00 total
```

For multiple tests, you can use the script:
```bash
bash ./go-test-many.sh 20 4 2A
```
Please view the comments in the script to know how to use it.

## Lab 3: Fault-tolerant Key/Value Service

https://pdos.csail.mit.edu/6.824/labs/lab-kvraft.html

实现细节：
- start一个entry后，根据term和index建立一个channel，如果这个entry被apply了，就会向channel中发送结果。channel不用了记得删掉。
- 客户端重复请求的过滤：因为一个客户端一定是在一个操作返回之后才进行下一个操作，所以server的状态除了kvMap之外还要保存一个dupMap，clientId -> lastResponse。如果发现现在要apply的entry跟lastResponse里的opId一样，那么就直接返回上次的结果，不apply这个entry。防止append等操作重复进行。
  - 返回OK的entry记录到lastResponse里。不OK的没有影响state，就不用了。
  - 其实这里还是有点瑕疵，可能client的某一个request过了很久才从client发到server。因此最好再实现client的opId必须大于lastResponse里的opId才能处理。
  - 如果允许客户端并发请求呢？我觉得可以在config的阶段设一个客户端最大允许的并发数，然后维护lastResponse的相同大小的set即可。

```
$ cd src/kvraft
$ time go test -race
Test: one client (3A) ...
labgob warning: Decoding into a non-default variable/field Err may not work
  ... Passed --  15.1  5  4389  637
Test: ops complete fast enough (3A) ...
  ... Passed --  26.0  3  4368    0
Test: many clients (3A) ...
  ... Passed --  16.0  5  4980  962
Test: unreliable net, many clients (3A) ...
  ... Passed --  17.6  5  4060  611
Test: concurrent append to same key, unreliable (3A) ...
  ... Passed --   2.2  3   256   52
Test: progress in majority (3A) ...
  ... Passed --   0.5  5    45    2
Test: no progress in minority (3A) ...
  ... Passed --   1.0  5   194    3
Test: completion after heal (3A) ...
  ... Passed --   1.0  5    59    3
Test: partitions, one client (3A) ...
  ... Passed --  22.6  5  4567  649
Test: partitions, many clients (3A) ...
  ... Passed --  23.5  5  5779 1099
Test: restarts, one client (3A) ...
  ... Passed --  19.9  5  4756  635
Test: restarts, many clients (3A) ...
  ... Passed --  22.0  5  6400 1129
Test: unreliable net, restarts, many clients (3A) ...
  ... Passed --  21.7  5  4913  667
Test: restarts, partitions, many clients (3A) ...
  ... Passed --  28.2  5  6233 1101
Test: unreliable net, restarts, partitions, many clients (3A) ...
  ... Passed --  28.9  5  4903  649
Test: unreliable net, restarts, partitions, random keys, many clients (3A) ...
  ... Passed --  33.2  7  7902  742
Test: InstallSnapshot RPC (3B) ...
  ... Passed --   3.0  3   943   63
Test: snapshot size is reasonable (3B) ...
  ... Passed --   8.8  3  3255  800
Test: ops complete fast enough (3B) ...
  ... Passed --  11.0  3  4004    0
Test: restarts, snapshots, one client (3B) ...
  ... Passed --  19.7  5 10252 1406
Test: restarts, snapshots, many clients (3B) ...
  ... Passed --  20.5  5 15036 3960
Test: unreliable net, snapshots, many clients (3B) ...
  ... Passed --  17.2  5  5435  918
Test: unreliable net, restarts, snapshots, many clients (3B) ...
  ... Passed --  21.2  5  5878  867
Test: unreliable net, restarts, partitions, snapshots, many clients (3B) ...
  ... Passed --  28.0  5  5467  726
Test: unreliable net, restarts, partitions, snapshots, random keys, many clients (3B) ...
  ... Passed --  31.4  7 12052 1417
PASS
ok      6.824/kvraft    440.838s
go test -race  619.90s user 50.43s system 151% cpu 7:21.26 total
```

## Lab 4: Sharded Key/Value Service

https://pdos.csail.mit.edu/6.824/labs/lab-shard.html

实现细节：
- Controler跟kvraft基本一样，注意平衡算法的实现需要deterministic即可。
- shardkv实现了两个challange，包括不用的shard进行删除，以及不同的shard在transfer时互不影响。
- 对于一个shard，记录以下状态
  - shard id，表示是哪一个shard
  - status。一共有4种状态，NULL表示没有这个shard的数据，READY表示有这个shard的数据，SERVE表示这个group正在负责这个shard的读写请求，PUSH表示这个group正在将shard的数据发送给新的config下的group。
  - version
  - data，这个shard的kvMap
  - resultHistory，这个shard的dupMap，用来filter重复的客户端请求
  - sendTo，如果shard的状态是PUSH，那么sendTo表示要发给哪一个group。这个sendTo在状态变为PUSH的时候就已经确定了，不能再更改。即使config更新了，如S1本来由G1负责，现在变成了G2，那么sentTo就是2。即使G1发现config又更新了，S1由G3来负责，G1也不能直接发送给G3，因为它不知道G2是否已经接收到了这个shard的数据并且接收了读写请求。
- server state除了记录每一个shard的状态外，还需要记录
  - 最新的config，以及firstConfig。firstConfig是在第一次启动的时候用的，对应的group可以将shard的status从NULL直接设为READY。
  - 每一个shard的knownMaxVersion。防止过时的push请求。
    - 这里有一个问题。如果S1本来由G1负责，新的config里由G2负责，那么G1发送给G2。但G2接收到发现新的config里又由G1负责了，那么会发回给G1。这时，可能会有完全对称的两个请求，无论实现细节是什么，都可能会出现一个shard在G1和G2上同时ready的情况。因此，在G2接收到S1数据的时候，默认把version +1。这样两个请求就不对称了，只有大于knownMaxVersion的请求才是有效的。
- raft保存的是一个state，也就是replicated state machine。所有的apply操作都必须马上完成，无需等待。因此，PUSH操作需要在goroutine里完成。PUSH完之后，删除shard数据要通过raft来提交一个删除的entry。
- 总共5种log entry
  - PutAppend和Get，处理客户端请求
  - UpdateConfig，周期性地更新最新的config
  - ReceiveShardData
  - DeleteShardData

```
$ cd src/shardctrler
$ time go test -race
Test: Basic leave/join ...
  ... Passed
Test: Historical queries ...
  ... Passed
Test: Move ...
  ... Passed
Test: Concurrent leave/join ...
  ... Passed
Test: Minimal transfers after joins ...
  ... Passed
Test: Minimal transfers after leaves ...
  ... Passed
Test: Multi-group join/leave ...
  ... Passed
Test: Concurrent multi leave/join ...
  ... Passed
Test: Minimal transfers after multijoins ...
  ... Passed
Test: Minimal transfers after multileaves ...
  ... Passed
Test: Check Same config on servers ...
  ... Passed
PASS
ok      6.824/shardctrler       5.921s
go test -race  2.95s user 0.81s system 58% cpu 6.478 total
```

```
$ cd src/shardkv
$ time go test -race                                  
Test: static shards ...
  ... Passed
Test: join then leave ...
  ... Passed
Test: snapshots, join, and leave ...
  ... Passed
Test: servers miss configuration changes...
  ... Passed
Test: concurrent puts and configuration changes...
  ... Passed
Test: more concurrent puts and configuration changes...
  ... Passed
Test: concurrent configuration change and restart...
  ... Passed
Test: unreliable 1...
  ... Passed
Test: unreliable 2...
  ... Passed
Test: unreliable 3...
  ... Passed
Test: shard deletion (challenge 1) ...
  ... Passed
Test: unaffected shard access (challenge 2) ...
  ... Passed
Test: partial migration shard access (challenge 2) ...
  ... Passed
PASS
ok      6.824/shardkv   137.357s
go test -race  246.84s user 17.19s system 191% cpu 2:18.02 total
```
