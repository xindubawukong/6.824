package main

import (
	"fmt"
)

type A struct {
	x int
	y int
}

func main() {
	var a = "abcd"

	fmt.Println(a[:3])
}
