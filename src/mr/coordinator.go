package mr

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"sync"
	"time"
)

type TaskStatus string

var IDLE TaskStatus = "IDLE"
var IN_PROGRESS TaskStatus = "IN_PROGRESS"
var COMPLETED TaskStatus = "COMPLETED"

type Task struct {
	taskId    string
	status    TaskStatus
	startTime time.Time
}

type Coordinator struct {
	// Your definitions here.
	files   []string
	nReduce int

	mutex sync.Mutex

	mapTasks    []Task
	reduceTasks []Task

	workerIdToTaskId map[string]string
	intermediate     map[int][]string
}

// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
//
func (c *Coordinator) Example(args *ExampleArgs, reply *ExampleReply) error {
	fmt.Println("Coordinator Example")
	reply.Y = args.X + 1
	return nil
}

func (c *Coordinator) AssignMapTask(i int, args *AskForWorkArgs, reply *AskForWorkReply) {
	reply.ExistTask = true
	reply.IsMapTask = true
	reply.InputFiles = c.files[i : i+1]
	reply.NReduce = c.nReduce
	reply.Index = i

	c.mapTasks[i].status = IN_PROGRESS
	c.mapTasks[i].startTime = time.Now()

	c.workerIdToTaskId[args.WorkerId] = c.mapTasks[i].taskId
}

func (c *Coordinator) AssignReduceTask(i int, args *AskForWorkArgs, reply *AskForWorkReply) {
	reply.ExistTask = true
	reply.IsMapTask = false
	reply.InputFiles = c.intermediate[i]
	reply.NReduce = c.nReduce
	reply.Index = i

	c.reduceTasks[i].status = IN_PROGRESS
	c.reduceTasks[i].startTime = time.Now()

	c.workerIdToTaskId[args.WorkerId] = c.reduceTasks[i].taskId
}

func (c *Coordinator) AskForWork(args *AskForWorkArgs, reply *AskForWorkReply) error {
	// log.Println("AskForWork " + args.WorkerId)
	c.mutex.Lock()
	var mapFinished = true
	for i := 0; i < len(c.mapTasks); i++ {
		if c.mapTasks[i].status == IDLE {
			c.AssignMapTask(i, args, reply)
			c.mutex.Unlock()
			return nil
		}
		if c.mapTasks[i].status != COMPLETED {
			mapFinished = false
		}
	}
	if mapFinished {
		for i := 0; i < len(c.reduceTasks); i++ {
			if c.reduceTasks[i].status == IDLE {
				c.AssignReduceTask(i, args, reply)
				c.mutex.Unlock()
				return nil
			}
		}
	}
	c.mutex.Unlock()
	reply.ExistTask = false
	return nil
}

func (c *Coordinator) getTaskByTaskId(taskId string) *Task {
	for i := 0; i < len(c.mapTasks); i++ {
		if c.mapTasks[i].taskId == taskId {
			return &c.mapTasks[i]
		}
	}
	for i := 0; i < len(c.reduceTasks); i++ {
		if c.reduceTasks[i].taskId == taskId {
			return &c.reduceTasks[i]
		}
	}
	log.Fatal("task not found: " + taskId)
	return nil
}

func (c *Coordinator) WorkFinish(args *WorkFinishArgs, reply *WorkFinishReply) error {
	c.mutex.Lock()
	// log.Println("WorkFinish " + args.WorkerId)
	taskId := c.workerIdToTaskId[args.WorkerId]
	task := c.getTaskByTaskId(taskId)
	if task.status == IN_PROGRESS {
		if args.IsSuccess {
			if taskId[:3] == "map" {
				if len(args.OutputFiles) != c.nReduce {
					log.Fatal("error map output files number")
				}
				task.status = COMPLETED
				for i := 0; i < c.nReduce; i++ {
					c.intermediate[i] = append(c.intermediate[i], args.OutputFiles[i])
				}
			} else {
				if len(args.OutputFiles) != 1 {
					log.Fatal("error reduce output files number")
				}
				task.status = COMPLETED
			}
		} else {
			task.status = IDLE
		}
	}
	c.mutex.Unlock()
	return nil
}

func checkTimeOut(task *Task) bool {
	if task.status == IN_PROGRESS {
		now := time.Now()
		duration := now.Sub(task.startTime)
		return duration.Seconds() > 10
	}
	return false
}

func (c *Coordinator) checkAll() {
	c.mutex.Lock()

	// log.Printf("\n\nmapTasks: %v\n\n", c.mapTasks)
	// log.Printf("\n\nreduceTasks: %v\n\n", c.reduceTasks)

	for i := 0; i < c.nReduce; i++ {
		if c.reduceTasks[i].status == COMPLETED {
			for j := 0; j < len(c.intermediate[i]); j++ {
				os.Remove(c.intermediate[i][j])
			}
		}
	}

	for i := 0; i < len(c.mapTasks); i++ {
		if checkTimeOut(&c.mapTasks[i]) {
			c.mapTasks[i].status = IDLE
		}
	}
	for i := 0; i < len(c.reduceTasks); i++ {
		if checkTimeOut(&c.reduceTasks[i]) {
			c.reduceTasks[i].status = IDLE
		}
	}

	c.mutex.Unlock()
}

func (c *Coordinator) checkStatus() {
	for {
		c.checkAll()
		time.Sleep(5000 * time.Millisecond)
	}
}

//
// start a thread that listens for RPCs from worker.go
//
func (c *Coordinator) server() {
	rpc.Register(c)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := coordinatorSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrcoordinator.go calls Done() periodically to find out
// if the entire job has finished.
//
func (c *Coordinator) Done() bool {
	c.checkAll()
	c.mutex.Lock()
	ret := true
	for i := 0; i < len(c.reduceTasks); i++ {
		if c.reduceTasks[i].status != COMPLETED {
			ret = false
			break
		}
	}
	c.mutex.Unlock()
	return ret
}

//
// create a Coordinator.
// main/mrcoordinator.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeCoordinator(files []string, nReduce int) *Coordinator {
	c := Coordinator{}

	// Your code here.

	c.files = files
	c.nReduce = nReduce
	c.mapTasks = make([]Task, len(files))
	for i := 0; i < len(c.mapTasks); i++ {
		c.mapTasks[i].taskId = "map_task_" + fmt.Sprint(i)
		c.mapTasks[i].status = IDLE
	}
	c.reduceTasks = make([]Task, nReduce)
	for i := 0; i < len(c.reduceTasks); i++ {
		c.reduceTasks[i].taskId = "reduce_task_" + fmt.Sprint(i)
		c.reduceTasks[i].status = IDLE
	}
	c.workerIdToTaskId = make(map[string]string)
	c.intermediate = make(map[int][]string)
	for i := 0; i < nReduce; i++ {
		c.intermediate[i] = []string{}
	}

	go c.checkStatus()
	c.server()
	return &c
}
