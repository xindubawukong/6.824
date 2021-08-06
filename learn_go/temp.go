package main

import (
	"fmt"
	"time"
)

func main() {
	go fmt.Println(1)
	time.Sleep(1000 * time.Millisecond)
	fmt.Println(2)
}
