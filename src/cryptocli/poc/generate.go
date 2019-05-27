package main

import (
	"time"
	"os"
)

func main() {
	for i := 0; i < 15; i++ {
		os.Stdout.Write([]byte{0x41,})
		time.Sleep(time.Second * 1)
	}

	time.Sleep(time.Second * 2)

	os.Stdout.Write([]byte{0x41,})
}
