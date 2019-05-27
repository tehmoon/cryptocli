package main

import (
	"os"
	"log"
	"time"
	"io"
	"sync"
)

func init() {
	log.SetOutput(os.Stderr)
}

type Message struct {
	Payload []byte
	Close bool
}

func main() {
	c := make(chan *Message, 0)
	go process(c)

	for {
		buff := make([]byte, 1<<24)
		i, err := os.Stdin.Read(buff)
		if err != nil {
			if err != io.EOF {
				panic(err)
			}

			c <- &Message{Payload: buff[:i], Close: true,}
			log.Printf("EOF reached.")
			break
		}

		c <- &Message{Payload: buff[:i], Close: false}
		log.Printf("Read %d bytes\n", i)
	}
}

type Target int

const (
	DIRECT Target = iota
	FILE
)

func process(c chan *Message) {
	i := int64(0)
	toread := &i
	*toread = int64(0)
	lock := &sync.RWMutex{}
	syn := make(chan *Message, 0)

	target := DIRECT

	go processFile(toread, lock, syn)

	for {
		select {
			case c := <- c:
				now := time.Now()

				switch target {
					case DIRECT:
						direct(c)
					case FILE:
						lock.RLock()
							*toread = *toread + int64(len(c.Payload))
							log.Printf("Toread: %d\n", *toread)
							if *toread == 0 {
								target = FILE
								direct(c)
							}
						lock.RUnlock()

						syn <- c
				}

				took := time.Since(now)
				if took > time.Second {
					target = FILE
					log.Println("Too slow!")
				}

				log.Printf("Took %s\n", took)
		}
	}
}

func direct(message *Message) (err error) {
	_, err = os.Stdout.Write(message.Payload)
	if message.Close {
		os.Stdout.Close()
	}

	return err
}

func processFile(toread *int64, lock *sync.RWMutex, syn chan *Message) {
	for {
		select {
			case message := <- syn:
				lock.Lock()
				*toread = *toread - int64(len(message.Payload))
				direct(message)
				log.Printf("toread after: %d\n", *toread)
				lock.Unlock()
		}
	}
}
