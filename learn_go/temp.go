package main

import (
	"fmt"
)

type BB struct {
	x int
}

type AA struct {
	p *BB
	q *BB
}

func main() {
	var a AA
	fmt.Println(a.p)
}
