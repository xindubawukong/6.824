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

```bash
cd src/main
go build -race -buildmode=plugin ../mrapps/wc.go
bash test-mr.sh
```

## Lab 2: Raft

https://pdos.csail.mit.edu/6.824/labs/lab-raft.html

```bash
cd src/raft
go test -race
```

For multiple tests, you can use the script:
```bash
bash ./go-test-many.sh 20 4 2A
```
Please view the comments in the script to know how to use it.
