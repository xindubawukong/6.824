package main

import (
	"fmt"
	"time"
)

var a = 0
var ch = make(chan int)

func set(x int) {
	a = x;
	ch <- x
}

func main() {
	for {
		go set(1)
		go set(2)
		time.Sleep(1000 * time.Millisecond)
		var t  = <- ch
		t  = <- ch
		fmt.Println(t, a)
	}
}
