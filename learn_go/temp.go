package main

import (
	"fmt"
)

func main() {
	var a map[string]string
	a = make(map[string]string)
	a["123"] = "asd"
	a["456"] = "qwe"
	fmt.Println(a)
}
