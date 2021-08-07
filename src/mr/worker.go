package mr

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"log"
	"net/rpc"
	"os"
	"sort"
	"time"

	"golang.org/x/sys/unix"
)

//
// Map functions return a slice of KeyValue.
//
type KeyValue struct {
	Key   string
	Value string
}

//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}

func workMap(mapf func(string, string) []KeyValue, files []string, mapId int, nReduce int, workerId string) ([]string, error) {
	// log.Println("workMap")
	intermediate := []KeyValue{}
	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			log.Printf("cannot open %v\n", filename)
			return nil, err
		}
		content, err := ioutil.ReadAll(file)
		if err != nil {
			log.Printf("cannot read %v\n", filename)
			return nil, err
		}
		file.Close()
		kva := mapf(filename, string(content))
		intermediate = append(intermediate, kva...)
	}
	var records map[int][]KeyValue = make(map[int][]KeyValue)
	for i := 0; i < nReduce; i++ {
		records[i] = []KeyValue{}
	}
	for _, kv := range intermediate {
		id := ihash(kv.Key) % nReduce
		records[id] = append(records[id], kv)
	}
	var outputFiles = make([]string, nReduce)
	for i := 0; i < nReduce; i++ {
		outputFiles[i] = "intermediate_" + fmt.Sprint(mapId) + "_" + fmt.Sprint(i)
		tempFile, err := ioutil.TempFile("./", outputFiles[i])
		if err != nil {
			log.Printf("cannot open %v\n", tempFile)
			return nil, err
		}
		enc := json.NewEncoder(tempFile)
		for _, kv := range records[i] {
			err := enc.Encode(&kv)
			if err != nil {
				log.Printf("encoding error\n")
				return nil, err
			}
		}
		tempFile.Close()
		os.Rename(tempFile.Name(), outputFiles[i])
	}
	return outputFiles, nil
}

// for sorting by key.
type ByKey []KeyValue

// for sorting by key.
func (a ByKey) Len() int           { return len(a) }
func (a ByKey) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByKey) Less(i, j int) bool { return a[i].Key < a[j].Key }

func workReduce(reducef func(string, []string) string, files []string, reduceId int, workerId string) ([]string, error) {
	// log.Println("workReduce " + fmt.Sprint(files))
	kva := []KeyValue{}
	for _, filename := range files {
		file, err := os.Open(filename)
		if err != nil {
			log.Printf("cannot open %v\n", filename)
			return nil, err
		}
		dec := json.NewDecoder(file)
		for {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			kva = append(kva, kv)
		}
	}
	sort.Sort(ByKey(kva))
	results := []KeyValue{}
	i := 0
	for i < len(kva) {
		j := i + 1
		for j < len(kva) && kva[j].Key == kva[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, kva[k].Value)
		}
		output := reducef(kva[i].Key, values)
		results = append(results, KeyValue{kva[i].Key, output})
		i = j
	}
	outputFileName := "mr-out-" + fmt.Sprint(reduceId)
	tmpFile, err := ioutil.TempFile("./", outputFileName)
	if err != nil {
		log.Printf("cannot open %v\n", tmpFile)
		return nil, err
	}
	for _, kv := range results {
		fmt.Fprintf(tmpFile, "%v %v\n", kv.Key, kv.Value)
	}
	tmpFile.Close()
	os.Rename(tmpFile.Name(), outputFileName)
	return []string{outputFileName}, nil
}

func workFinish(workerId string, isSuccess bool, outputFiles []string) {
	args := WorkFinishArgs{}
	args.WorkerId = workerId
	args.IsSuccess = isSuccess
	args.OutputFiles = outputFiles
	reply := WorkFinishReply{}
	call("Coordinator.WorkFinish", &args, &reply)
}

//
// main/mrworker.go calls this function.
//
func Worker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.

	// uncomment to send the Example RPC to the coordinator.
	// CallExample()

	var workerId = "worker-" + fmt.Sprint(unix.Getpid())

	for {
		args := AskForWorkArgs{}
		args.WorkerId = workerId
		reply := AskForWorkReply{}
		call("Coordinator.AskForWork", &args, &reply)
		if reply.ShutDown {
			break
		}
		if reply.ExistTask { // map task
			if reply.IsMapTask {
				files, err := workMap(mapf, reply.InputFiles, reply.Index, reply.NReduce, workerId)
				if err != nil {
					workFinish(workerId, false, files)
				}
				workFinish(workerId, true, files)
			} else { // reduce task
				files, err := workReduce(reducef, reply.InputFiles, reply.Index, workerId)
				if err != nil {
					workFinish(workerId, false, files)
				}
				workFinish(workerId, true, files)
			}
		}
		time.Sleep(1000 * time.Millisecond)
	}

}

//
// example function to show how to make an RPC call to the coordinator.
//
// the RPC argument and reply types are defined in rpc.go.
//
func CallExample() {

	// declare an argument structure.
	args := ExampleArgs{}

	// fill in the argument(s).
	args.X = 99

	// declare a reply structure.
	reply := ExampleReply{}

	// send the RPC request, wait for the reply.
	call("Coordinator.Example", &args, &reply)

	// reply.Y should be 100.
	fmt.Printf("reply.Y %v\n", reply.Y)
}

//
// send an RPC request to the coordinator, wait for the response.
// usually returns true.
// returns false if something goes wrong.
//
func call(rpcname string, args interface{}, reply interface{}) bool {
	// c, err := rpc.DialHTTP("tcp", "127.0.0.1"+":1234")
	sockname := coordinatorSock()
	c, err := rpc.DialHTTP("unix", sockname)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	defer c.Close()

	err = c.Call(rpcname, args, reply)
	if err == nil {
		return true
	}

	fmt.Println(err)
	return false
}
