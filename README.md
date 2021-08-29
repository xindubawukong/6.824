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

Run tests:
```
cd src/raft
time go test -race
```

Result:
```
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
