package main

import (
	"fmt"
	"time"
)

func main() {
	fmt.Println("hello world")
	for {
		time.Sleep(1 * time.Second)
	}
}
