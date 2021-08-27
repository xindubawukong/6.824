package main

import "fmt"

type asd struct {
	a int
}

func f(q asd) {
	q.a = 2
}

func main() {
	var q asd
	q.a = 1
	f(q)
	fmt.Println(q.a)
}
