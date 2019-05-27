package main

import (
	"os"
	"log"
	"io/ioutil"
	"io"
	"sync"
)

func main() {
	stdin := os.Stdin
	passFile := make(chan string, 0)
	ok := &sync.WaitGroup{}
	cancel := make(chan struct{}, 0)

	ok.Add(1)
	go process(passFile, ok, cancel)

	for {
		f, err := ioutil.TempFile("", "")
		if err != nil {
			panic(err)
		}

		var buff int64 = 1<<27

		i, err := io.CopyN(f, stdin, buff)
		if err != nil {
			if err == io.EOF {
				log.Printf("EOF: Written %d bytes to %s\n", i, f.Name())
				f.Close()
				passFile <- f.Name()
			}

			log.Printf("Cancelling with err: %s\n", err.Error())
			cancel <- struct{}{}
			break
		}

		log.Printf("Written %d bytes to %s\n", i, f.Name())
		f.Close()
		passFile <- f.Name()
	}

	ok.Wait()
}

// todo: add break to go cancel futur buff
func process(passFile chan string, ok *sync.WaitGroup, cancel chan struct{}) {
	defer ok.Done()

	for {
		select {
			case name := <- passFile:
				f, err := os.Open(name)
				if err != nil {
					panic(err)
				}

				i, err := io.Copy(os.Stdout, f)
				if err != nil {
					panic(err)
				}
				log.Printf("Read %d bytes from %s\n", i, f.Name())

				f.Close()
				os.Remove(name)
			case <- cancel:
				log.Printf("Cancel received\n")
				return
		}
	}
}
