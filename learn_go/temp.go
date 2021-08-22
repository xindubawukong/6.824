package main

import "log"

// Debugging
const Debug = true

func DPrintf(format string, a ...interface{}) (n int) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

func main() {
	var x = DPrintf("asd")
	DPrintf("%v", x)
}
