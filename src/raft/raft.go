package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	//	"bytes"
	"bytes"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	//	"6.824/labgob"
	"6.824/labgob"
	"6.824/labrpc"
)

//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in part 2D you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh, but set CommandValid to false for these
// other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int

	// For 2D:
	SnapshotValid bool
	Snapshot      []byte
	SnapshotTerm  int
	SnapshotIndex int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	// Your data here (2A, 2B, 2C).
	// Look at the paper's Figure 2 for a description of what
	// state a Raft server must maintain.

	status      ServerStatus
	currentTerm int
	votedFor    int
	log         []LogEntry
	commitIndex int
	nextIndex   []int
	matchIndex  []int

	electionTimer   time.Time
	electionTimeout time.Duration

	syncing        []sync.Mutex
	applyCh        chan ApplyMsg
	lastSentCommit int

	snapshot []byte
}

type LogEntry struct {
	Command interface{}
	Index   int
	Term    int
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.lockFields("GetState")
	var term = rf.currentTerm
	var isleader = rf.status == LEADER
	rf.unlockFields("GetState")
	return term, isleader
}

func (rf *Raft) demote(term int) {
	rf.votedFor = -1
	rf.currentTerm = term
	rf.status = FOLLOWER
	rf.persist()
}

func checkAsUpToDate(term1, index1, term2, index2 int) bool {
	if term1 != term2 {
		return term1 > term2
	}
	return index1 >= index2
}

func (rf *Raft) lockFields(info string) {
	rf.mu.Lock()
	// if len(info) > 0 && rf.me == 0 {
	// 	DPrintf("%v lock[%d]  term: %d\n", info, rf.me, rf.currentTerm)
	// }
}

func (rf *Raft) unlockFields(info string) {
	// if len(info) > 0 && rf.me == 0 {
	// 	DPrintf("%v unlock[%d]  term: %d\n", info, rf.me, rf.currentTerm)
	// }
	rf.mu.Unlock()
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
// inside lock
//
func (rf *Raft) persist() {
	// DPrintf("persisting")
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.votedFor)
	e.Encode(rf.log)
	e.Encode(rf.lastSentCommit)
	data := w.Bytes()
	// DPrintf("data: %v\n", data)
	go rf.persister.SaveStateAndSnapshot(data, rf.snapshot)
}

//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	// DPrintf("readPersist  me: %d  data: %v\n", rf.me, data)
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var currentTerm int
	var votedFor int
	var log []LogEntry
	var lastSentCommit int
	if d.Decode(&currentTerm) != nil ||
		d.Decode(&votedFor) != nil ||
		d.Decode(&log) != nil ||
		d.Decode(&lastSentCommit) != nil {
		DPrintf("Decode error\n")
	} else {
		rf.lockFields("readPersist")
		rf.currentTerm = currentTerm
		rf.votedFor = votedFor
		rf.log = log
		rf.lastSentCommit = lastSentCommit
		rf.unlockFields("readPersist")
	}
}

//
// A service wants to switch to snapshot.  Only do so if Raft hasn't
// have more recent info since it communicate the snapshot on applyCh.
//
func (rf *Raft) CondInstallSnapshot(lastIncludedTerm int, lastIncludedIndex int, snapshot []byte) bool {
	// Your code here (2D).
	rf.lockFields("CondInstallSnapshot")
	res := false
	if rf.log[0].Index <= lastIncludedIndex {
		res = true
		rf.snapshot = snapshot
		pos := binarySearch(rf.log, lastIncludedIndex)
		if pos != -1 && rf.log[pos].Term == lastIncludedTerm {
			rf.log = rf.log[pos:]
		} else {
			rf.log = make([]LogEntry, 0)
			rf.log = append(rf.log, LogEntry{nil, lastIncludedIndex, lastIncludedTerm})
		}
		rf.persist()
	}
	// DPrintf("CondInstallSnapshot  me: %d  lastIndex: %d  lastTerm: %d  res: %v\n", rf.me, lastIncludedIndex, lastIncludedTerm, res)
	rf.unlockFields("CondInstallSnapshot")
	return res
}

// the service says it has created a snapshot that has
// all info up to and including index. this means the
// service no longer needs the log through (and including)
// that index. Raft should now trim its log as much as possible.
func (rf *Raft) Snapshot(index int, snapshot []byte) {
	// Your code here (2D).
	rf.lockFields("Snapshot")
	if index > rf.log[0].Index {
		pos := binarySearch(rf.log, index)
		if pos != -1 {
			rf.snapshot = snapshot
			rf.log = rf.log[pos:]
			rf.persist()
		}
	}
	// DPrintf("Snapshot done. logs: %v\n", rf.log)
	rf.unlockFields("Snapshot")
}

//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term        int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.lockFields("RequestVote")
	// DPrintf("RequestVote  me: %d  candidate: %d  myterm: %d\n", rf.me, args.CandidateId, rf.currentTerm)
	if args.Term > rf.currentTerm {
		rf.demote(args.Term)
	}
	reply.Term = rf.currentTerm
	if args.Term < rf.currentTerm {
		reply.VoteGranted = false
	} else if (rf.votedFor == -1 || rf.votedFor == args.CandidateId) && checkAsUpToDate(args.LastLogTerm, args.LastLogIndex, rf.log[len(rf.log)-1].Term, rf.log[len(rf.log)-1].Index) {
		rf.votedFor = args.CandidateId
		reply.VoteGranted = true
		rf.electionTimer = time.Now()
		rf.electionTimeout = getRandomElectionTimeout()
		rf.persist()
	} else {
		reply.VoteGranted = false
	}
	// DPrintf("RequestVote  me: %d  candidate: %d  myterm: %d  vote: %v\n", rf.me, args.CandidateId, rf.currentTerm, reply.VoteGranted)
	rf.unlockFields("RequestVote")
}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	// DPrintf("sendRequestVote %d %v\n", server, args)
	ch := make(chan int, 2)
	var tmp RequestVoteReply
	go func() {
		ok := rf.peers[server].Call("Raft.RequestVote", args, &tmp)
		if ok {
			ch <- 1
		} else {
			ch <- 2
		}
	}()
	go func() {
		time.Sleep(REQUEST_VOTE_TIME_OUT)
		ch <- 3
	}()
	t := <-ch
	if t == 1 {
		*reply = tmp
		return true
	} else {
		return false
	}
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term    int
	Success bool
	Samples []LogEntry
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	// DPrintf("receive AppendEntries  me: %d\n", rf.me)
	rf.lockFields("AppendEntries")
	if args.Term > rf.currentTerm || (rf.status == CANDIDATE && args.Term >= rf.currentTerm) {
		rf.demote(args.Term)
	}
	reply.Term = rf.currentTerm
	reply.Samples = make([]LogEntry, 0)
	if args.Term >= rf.currentTerm && rf.status == FOLLOWER {
		rf.electionTimer = time.Now()
		rf.electionTimeout = getRandomElectionTimeout()
		pos := binarySearch(rf.log, args.PrevLogIndex)
		if pos == -1 || rf.log[pos].Term != args.PrevLogTerm {
			reply.Success = false
			now := len(rf.log) - 1
			if pos != -1 {
				now = pos - 1
			}
			step := 1
			for now >= 0 {
				var entry LogEntry
				entry.Index = rf.log[now].Index
				entry.Term = rf.log[now].Term
				reply.Samples = append(reply.Samples, entry)
				now -= step
				step *= 2
			}
			reply.Samples = append(reply.Samples, rf.log[0])
		} else {
			reply.Success = true
			notMatch := pos + 1
			i := 0
			for ; i < len(args.Entries); i++ {
				if notMatch >= len(rf.log) {
					break
				}
				if rf.log[notMatch].Index != args.Entries[i].Index || rf.log[notMatch].Term != args.Entries[i].Term {
					break
				}
				notMatch++
			}
			if i < len(args.Entries) {
				rf.log = rf.log[:notMatch]
				rf.log = append(rf.log, args.Entries[i:]...)
				rf.persist()
			}
			if args.LeaderCommit > rf.commitIndex {
				if len(args.Entries) > 0 {
					rf.commitIndex = get_min(args.LeaderCommit, args.Entries[len(args.Entries)-1].Index)
				} else {
					rf.commitIndex = get_min(args.LeaderCommit, args.PrevLogIndex)
				}
				// DPrintf("me: %d  commitIndex: %d\n", rf.me, rf.commitIndex)
			}
		}
	}
	// if len(args.Entries) > 0 {
	// 	DPrintf("Handling AppendEntries  me: %d  args: %v  reply: %v  log: %v\n", rf.me, args, reply, rf.log)
	// }
	rf.unlockFields("AppendEntries")
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	// if len(args.Entries) > 0 {
	// 	DPrintf("sendAppendEntries  me: %d  server: %d  args: %v\n", rf.me, server, args)
	// }
	ch := make(chan int, 2)
	var tmp AppendEntriesReply
	go func() {
		ok := rf.peers[server].Call("Raft.AppendEntries", args, &tmp)
		if ok {
			ch <- 1
		} else {
			ch <- 2
		}
	}()
	go func() {
		time.Sleep(APPEND_ENTRIES_TIME_OUT)
		ch <- 3
	}()
	t := <-ch
	if t == 1 {
		*reply = tmp
		// if len(args.Entries) > 0 {
		// 	DPrintf("sendAppendEntries  me: %d  server: %d  reply: %v\n", rf.me, server, reply)
		// }
		return true
	} else {
		// if len(args.Entries) > 0 {
		// 	DPrintf("sendAppendEntries  me: %d  server: %d  fail!", rf.me, server)
		// }
		return false
	}
}

type InstallSnapshotArgs struct {
	Term              int
	LeaderId          int
	LastIncludedIndex int
	LastIncludedTerm  int
	Snapshot          []byte
}

type InstallSnapshotReply struct {
	Term int
}

func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {
	var msgs []ApplyMsg = make([]ApplyMsg, 0)
	rf.lockFields("InstallSnapshot")
	if args.Term > rf.currentTerm {
		rf.demote(args.Term)
	}
	reply.Term = rf.currentTerm
	if args.Term >= rf.currentTerm && rf.status == FOLLOWER {
		var msg ApplyMsg
		msg.SnapshotValid = true
		msg.SnapshotIndex = args.LastIncludedIndex
		msg.SnapshotTerm = args.LastIncludedTerm
		msg.Snapshot = args.Snapshot
		msgs = append(msgs, msg)
	}
	rf.unlockFields("InstallSnapshot")
	for i := 0; i < len(msgs); i++ {
		msg := msgs[i]
		rf.applyCh <- msg
	}
}

func (rf *Raft) sendInstallSnapshot(server int, args *InstallSnapshotArgs, reply *InstallSnapshotReply) bool {
	// DPrintf("sendInstallSnapshot  me: %d  server: %d  args: %v\n", rf.me, server, args)
	ch := make(chan int, 2)
	var tmp InstallSnapshotReply
	go func() {
		ok := rf.peers[server].Call("Raft.InstallSnapshot", args, &tmp)
		if ok {
			ch <- 1
		} else {
			ch <- 2
		}
	}()
	go func() {
		time.Sleep(INSTALL_SNAPSHOT_TIME_OUT)
		ch <- 3
	}()
	t := <-ch
	if t == 1 {
		*reply = tmp
		// DPrintf("sendInstallSnapshot  me: %d  server: %d  success!\n", rf.me, server)
		return true
	} else {
		// DPrintf("sendInstallSnapshot  me: %d  server: %d  fail!\n", rf.me, server)
		return false
	}
}

func (rf *Raft) tryStartElection() {
	rf.lockFields("tryStartElection 1")
	duration := time.Since(rf.electionTimer)
	shouldStart := rf.status != LEADER && duration > rf.electionTimeout
	if shouldStart {
		rf.status = CANDIDATE
		rf.currentTerm++
		rf.votedFor = rf.me
		rf.electionTimer = time.Now()
		rf.electionTimeout = getRandomElectionTimeout()
		rf.persist()
		// DPrintf("should start election  me: %d\n", rf.me)
	}
	var args RequestVoteArgs
	args.Term = rf.currentTerm
	args.CandidateId = rf.me
	args.LastLogIndex = rf.log[len(rf.log)-1].Index
	args.LastLogTerm = rf.log[len(rf.log)-1].Term
	rf.unlockFields("tryStartElection 1")

	if !shouldStart {
		return
	}

	// DPrintf("start election, me: %d, term: %d, time: %v\n", rf.me, rf.currentTerm, time.Now())

	var ch = make(chan RequestVoteReply)
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			go func(server int, args RequestVoteArgs) {
				var reply RequestVoteReply
				rf.sendRequestVote(server, &args, &reply)
				ch <- reply
			}(i, args)
		}
	}
	var replies = make([]RequestVoteReply, 0)
	cnt := 1
	for reply := range ch {
		replies = append(replies, reply)
		// DPrintf("start election, me: %d  replies: %v  time: %v\n", rf.me, replies, time.Now())
		if reply.VoteGranted {
			cnt++
		}
		rf.lockFields("tryStartElection 2")
		if reply.Term > rf.currentTerm {
			rf.demote(reply.Term)
		}
		if rf.status == CANDIDATE && rf.currentTerm == args.Term && isMajority(cnt, len(rf.peers)) {
			rf.status = LEADER
			for i := 0; i < len(rf.peers); i++ {
				if i != rf.me {
					rf.nextIndex[i] = rf.log[len(rf.log)-1].Index + 1
					rf.matchIndex[i] = 0
					go rf.trySyncLogWith(i)
				}
			}
			// DPrintf("!!!! %d became leader!  Term: %d\n", rf.me, rf.currentTerm)
		}
		rf.unlockFields("tryStartElection 2")
		if len(replies) == len(rf.peers)-1 {
			break
		}
	}
}

func (rf *Raft) trySendHeartBeat() {
	rf.lockFields("trySendHeartBeat")
	shouldSend := rf.status == LEADER
	var args AppendEntriesArgs
	args.Term = rf.currentTerm
	args.LeaderId = rf.me
	args.PrevLogIndex = rf.log[len(rf.log)-1].Index
	args.PrevLogTerm = rf.log[len(rf.log)-1].Term
	args.Entries = make([]LogEntry, 0)
	args.LeaderCommit = rf.commitIndex
	rf.unlockFields("trySendHeartBeat")
	// if shouldSend {
	// 	DPrintf("trySendHeartBeat %d %v\n", rf.me, shouldSend)
	// }
	if !shouldSend {
		return
	}

	var ch = make(chan AppendEntriesReply)
	for i := 0; i < len(rf.peers); i++ {
		if i != rf.me {
			go func(server int, args AppendEntriesArgs) {
				var reply AppendEntriesReply
				rf.sendAppendEntries(server, &args, &reply)
				ch <- reply
			}(i, args)
		}
	}

	var replies = make([]AppendEntriesReply, 0)
	for reply := range ch {
		rf.mu.Lock()
		if reply.Term > rf.currentTerm {
			rf.demote(reply.Term)
		}
		rf.mu.Unlock()
		replies = append(replies, reply)
		if len(replies) == len(rf.peers)-1 {
			break
		}
	}
}

func (rf *Raft) trySyncLogWith(server int) {
	rf.syncing[server].Lock()
	for !rf.killed() {
		rf.lockFields("trySyncLogWith 1")
		shouldSend := (rf.status == LEADER) && rf.log[len(rf.log)-1].Index >= rf.nextIndex[server]
		shouldInstallSnapshot := false
		var appendEntriesArgs AppendEntriesArgs
		var installSnapshotArgs InstallSnapshotArgs
		if shouldSend {
			prev := binarySearch(rf.log, rf.nextIndex[server]-1)
			if prev != -1 {
				appendEntriesArgs.Term = rf.currentTerm
				appendEntriesArgs.LeaderId = rf.me
				appendEntriesArgs.PrevLogIndex = rf.log[prev].Index
				appendEntriesArgs.PrevLogTerm = rf.log[prev].Term
				appendEntriesArgs.Entries = rf.log[prev+1:]
				appendEntriesArgs.LeaderCommit = rf.commitIndex
			} else {
				shouldInstallSnapshot = true
				installSnapshotArgs.Term = rf.currentTerm
				installSnapshotArgs.LeaderId = rf.me
				installSnapshotArgs.LastIncludedIndex = rf.log[0].Index
				installSnapshotArgs.LastIncludedTerm = rf.log[0].Term
				installSnapshotArgs.Snapshot = rf.snapshot
			}
		}
		rf.unlockFields("trySyncLogWith 1")

		var shouldBreak = false
		if !shouldSend {
			shouldBreak = true
		} else if shouldSend && !shouldInstallSnapshot {
			var appendEntriesReply AppendEntriesReply
			rf.sendAppendEntries(server, &appendEntriesArgs, &appendEntriesReply)
			rf.lockFields("shouldSend && !shouldInstallSnapshot")
			if appendEntriesReply.Term > rf.currentTerm {
				shouldBreak = true
				rf.demote(appendEntriesReply.Term)
			} else if rf.currentTerm != appendEntriesArgs.Term || rf.status != LEADER {
				shouldBreak = true
			} else if appendEntriesReply.Success {
				shouldBreak = true
				rf.nextIndex[server] = appendEntriesArgs.Entries[len(appendEntriesArgs.Entries)-1].Index + 1
				rf.matchIndex[server] = appendEntriesArgs.Entries[len(appendEntriesArgs.Entries)-1].Index
			} else if appendEntriesReply.Term == 0 {
				shouldBreak = false
			} else { // optimization for nextIndex--
				shouldBreak = false
				updated := false
				if len(appendEntriesReply.Samples) > 0 {
					for i := 0; i < len(appendEntriesReply.Samples); i++ {
						entry := appendEntriesReply.Samples[i]
						pos := binarySearch(rf.log, entry.Index)
						if pos != -1 && rf.log[pos].Term == entry.Term {
							rf.nextIndex[server] = get_max(rf.matchIndex[server]+1, entry.Index+1)
							updated = true
							break
						}
					}
				}
				if !updated {
					rf.nextIndex[server] = rf.matchIndex[server] + 1
				}
				// DPrintf("trySyncLogWith failed. 	me: %d  to: %d  nextIndex: %d  matchIndex: %d\n", rf.me, server, rf.nextIndex[server], rf.matchIndex[server])
			}
			rf.unlockFields("shouldSend && !shouldInstallSnapshot")
		} else if shouldSend && shouldInstallSnapshot {
			var installSnapshotReply InstallSnapshotReply
			rf.sendInstallSnapshot(server, &installSnapshotArgs, &installSnapshotReply)
			rf.mu.Lock()
			if installSnapshotReply.Term == rf.currentTerm {
				rf.nextIndex[server] = installSnapshotArgs.LastIncludedIndex + 1
				rf.matchIndex[server] = installSnapshotArgs.LastIncludedIndex
				// DPrintf("nextIndex: %d  matchIndex: %d\n", rf.nextIndex[server], rf.matchIndex[server])
			} else if installSnapshotReply.Term == 0 {
				shouldBreak = false
			} else {
				shouldBreak = true
				if installSnapshotReply.Term > rf.currentTerm {
					rf.demote(installSnapshotReply.Term)
				}
			}
			rf.mu.Unlock()
			time.Sleep(50 * time.Millisecond)
		}
		if shouldBreak {
			break
		}
	}
	rf.syncing[server].Unlock()
}

//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	// Your code here (2B).
	rf.lockFields("Start")
	// DPrintf("Start  me: %d  command: %v\n", rf.me, command)
	var index, term int
	var isLeader bool
	if rf.status == LEADER {
		var entry LogEntry
		entry.Command = command
		entry.Index = rf.log[len(rf.log)-1].Index + 1
		entry.Term = rf.currentTerm
		rf.log = append(rf.log, entry)
		for i := 0; i < len(rf.peers); i++ {
			if i != rf.me {
				go rf.trySyncLogWith(i)
			}
		}
		index = entry.Index
		term = entry.Term
		isLeader = true
	} else {
		index = -1
		term = -1
		isLeader = false
	}
	rf.persist()
	rf.unlockFields("Start")
	return index, term, isLeader
}

//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}

// The ticker go routine starts a new election if this peer hasn't received
// heartsbeats recently.
func (rf *Raft) ticker() {
	for !rf.killed() {

		// Your code here to check if a leader election should
		// be started and to randomize sleeping time using
		// time.Sleep().

		go rf.tryStartElection()
		time.Sleep(50 * time.Millisecond)

	}
}

func (rf *Raft) startHeartBeat() {
	for !rf.killed() {
		go rf.trySendHeartBeat()
		time.Sleep(120 * time.Millisecond)
	}
}

func (rf *Raft) startTryCommit() {
	for !rf.killed() {
		var msgs []ApplyMsg = make([]ApplyMsg, 0)
		rf.lockFields("startTryCommit")
		if rf.status == LEADER {
			a := make([]int, 0)
			for i := 0; i < len(rf.peers); i++ {
				if i != rf.me {
					a = append(a, rf.matchIndex[i])
				}
			}
			sort.Ints(a)
			N := a[len(rf.peers)/2]
			pos := binarySearch(rf.log, N)
			if pos != -1 && N > rf.commitIndex && rf.log[pos].Term == rf.currentTerm {
				// DPrintf("leader %d commit to %d\n", rf.me, N)
				rf.commitIndex = N
			}
		}
		for rf.lastSentCommit < rf.commitIndex {
			rf.lastSentCommit++
			pos := binarySearch(rf.log, rf.lastSentCommit)
			if pos != -1 {
				var msg ApplyMsg
				msg.CommandValid = true
				msg.CommandIndex = rf.log[pos].Index
				msg.Command = rf.log[pos].Command
				msgs = append(msgs, msg)
			}
		}
		rf.persist()
		rf.unlockFields("startTryCommit")
		for i := 0; i < len(msgs); i++ {
			msg := msgs[i]
			rf.applyCh <- msg
		}
		time.Sleep(30 * time.Millisecond)
	}
}

//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	DPrintf("start server %d\n", me)
	if len(peers) == 0 {
		DPrintf("no peers!\n")
		return nil
	}

	rf := &Raft{}
	rf.peers = peers
	rf.persister = persister
	rf.me = me
	rf.applyCh = applyCh

	// Your initialization code here (2A, 2B, 2C).

	rf.status = FOLLOWER
	rf.currentTerm = 0
	rf.votedFor = -1
	var emptyEntry LogEntry
	emptyEntry.Index = 0
	emptyEntry.Term = 0
	rf.log = append(make([]LogEntry, 0), emptyEntry)
	rf.commitIndex = 0

	rf.electionTimer = time.Now()
	rf.electionTimeout = getRandomElectionTimeout()

	rf.syncing = make([]sync.Mutex, len(peers))
	rf.lastSentCommit = 0

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())
	rf.snapshot = clone(persister.ReadSnapshot())

	rf.nextIndex = make([]int, len(peers))
	rf.matchIndex = make([]int, len(peers))
	for i := 0; i < len(peers); i++ {
		rf.nextIndex[i] = rf.log[len(rf.log)-1].Index + 1
		rf.matchIndex[i] = 0
	}

	// start ticker goroutine to start elections
	go rf.ticker()
	go rf.startHeartBeat()
	go rf.startTryCommit()

	return rf
}
