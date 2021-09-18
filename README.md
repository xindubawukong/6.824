# 6.824

Author: xindubawukong@gmail.com

https://pdos.csail.mit.edu/6.824/schedule.html

This repo is for Spring 2021.

```
$ go version
go version go1.15.14 darwin/amd64
```

## Lab 1: MapReduce

https://pdos.csail.mit.edu/6.824/labs/lab-mr.html

```
cd src/main
go build -race -buildmode=plugin ../mrapps/wc.go
bash test-mr.sh
```

## Lab 2: Raft

https://pdos.csail.mit.edu/6.824/labs/lab-raft.html

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
