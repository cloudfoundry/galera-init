package main

import (
	"fmt"
	"os"
	"time"
)

func main() {

	args := os.Args[1:]
	for _, arg := range args {
		if arg == "--skip-networking" {
			fmt.Println("Starting Isaac in skip-networking mode. Returning in 1 second")
			time.Sleep(5 * time.Second)
			return
		}
	}
	fmt.Println("Isaac is about to sleep")
	time.Sleep(180 * time.Second)
}
