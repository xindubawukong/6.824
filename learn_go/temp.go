package main

import (
	"fmt"
)

func main() {
	var a []int = []int{1,2,3,4,5}
	var b []int = a[:3]
	a = append([]int{90}, a[4:]...)
	fmt.Println(a)
	fmt.Println(b)
}
