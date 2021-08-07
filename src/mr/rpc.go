package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"

//
// example to show how to declare the arguments
// and reply for an RPC.
//

type ExampleArgs struct {
	X int
}

type ExampleReply struct {
	Y int
}

// Add your RPC definitions here.

type AskForWorkArgs struct {
	WorkerId string
}

type AskForWorkReply struct {
	ExistTask bool
	IsMapTask bool  // map or reduce
	InputFiles []string
	NReduce int
	Index int
	ShutDown bool
}

type WorkFinishArgs struct {
	WorkerId string
	IsSuccess bool
	OutputFiles []string
}

type WorkFinishReply struct {
}

// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the coordinator.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func coordinatorSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
