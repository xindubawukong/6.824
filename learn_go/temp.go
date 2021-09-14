package main

import (
	"fmt"
	"time"
)

func main() {
	var ch chan int
	go func() {
		fmt.Println("goroutine start")
		ch <- 1
		fmt.Println("goroutine end")
	}()
	time.Sleep(10 * time.Second)
	fmt.Println("end")
}
