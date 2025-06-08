package main

import (
	"fmt"
	"os"
	"time"
)

func main() {
	for {
		dt := time.Now()
		fmt.Printf("[%d] time.Now() returns: %s\n", os.Getpid(), dt.String())
		time.Sleep(1 * time.Second)
	}
}
