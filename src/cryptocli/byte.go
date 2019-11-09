package main

import (
	"sync"
	"github.com/spf13/pflag"
	"io"
	"log"
	"github.com/tehmoon/errors"
	"bufio"
	"regexp"
	"fmt"
)

func init() {
	MODULELIST.Register("byte", "Byte manipulation module", NewByte)
}

type Byte struct {
	messageSize int
	skipMessages int
	maxMessages int
	delimFlag string
	delimiter *regexp.Regexp
	prepend string
	append string
}

func (m *Byte) Init(in, out chan *Message, global *GlobalFlags) (err error) {
	if m.messageSize > 0 && m.delimFlag != "" {
		return errors.Errorf("Flag %q is mutually exclusive with flag %q\n", "message-size", "delimiter")
	}

	if m.delimFlag != "" {
		// This will wrap the user's regex into two capture groups.
		// The first group will match the begining of the data.
		// Since ? is not greedy, it won't count as a match if the user's
		// regex does not match either, returning the whole token if necessary.
		m.delimFlag = fmt.Sprintf("(.*?)(%s)", m.delimFlag)
	}

	m.delimiter, err = regexp.Compile(m.delimFlag)
	if err != nil {
		return errors.Wrapf(err, "Error parsing flag %q", "delimiter")
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					inc, ok := message.Interface.(MessageChannel)
					if ok {
						outc := make(MessageChannel)

						out <- &Message{
							Type: MessageTypeChannel,
							Interface: outc,
						}
						wg.Add(1)
						go startByteHandler(m, inc, outc, wg)
					}
			}
		}

		wg.Wait()
		// Last message will signal the closing of the channel
		<- in
		close(out)
	}(in, out)

	return nil
}

// Read from the callback function and return data
// for further processing.
type ByteReaderCallback func() ([]byte, error)

var ByteReaderCallbackFull = func(reader io.Reader, n int) (ByteReaderCallback) {
	return func() ([]byte, error) {
		buff := make([]byte, n)
		n, err := io.ReadFull(reader, buff)
		if n > 0 {
			return buff[:n], err
		}

		return nil, err
	}
}

var ByteReaderCallbackMessage = func(reader *ChannelReader) (ByteReaderCallback) {
	return func() ([]byte, error) {
		return reader.ReadMessage()
	}
}

var ByteReaderCallbackDelim = func(reader io.Reader, delim *regexp.Regexp) (ByteReaderCallback) {
	scanner := bufio.NewScanner(reader)

	// Use a split function to split and advance from the byte stream
	// If there is nothing to match, more data is asked.
	// If there is a match, everything up to the match is returned.
	scanner.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		if atEOF && len(data) > 0 {
			return 0, data, bufio.ErrFinalToken
		}

		finds := delim.FindSubmatch(data)

		if len(finds) < 2 {
			return 0, nil, nil
		}

		token = append(finds[1], finds[2]...)
		if len(token) == 0 {
			return 0, nil, nil
		}

		advance = len(token)

		return advance, token, nil
	})

	return func() (data []byte, err error) {
		cont := scanner.Scan()

		if cont {
			data = scanner.Bytes()

			if data == nil {
				data = make([]byte, 0)
			}

			return data, nil
		}

		err = scanner.Err()
		if err == nil {
			err = io.EOF
		}

		return nil, err
	}
}

func startByteHandler(m *Byte, inc, outc MessageChannel, wg *sync.WaitGroup) {
	defer wg.Done()
	defer DrainChannel(inc, nil)
	defer close(outc)

	reader := NewChannelReader(inc)

	var cb ByteReaderCallback

	if m.delimFlag != "" {
		cb = ByteReaderCallbackDelim(reader, m.delimiter)
	} else if m.messageSize > 0 {
		cb = ByteReaderCallbackFull(reader, m.messageSize)
	} else {
		cb = ByteReaderCallbackMessage(reader)
	}

	skipped := 0
	count := 1

	if (m.maxMessages - m.skipMessages) <= 0 && m.skipMessages > 0 && m.maxMessages > 0 {
		return
	}

	for {
		payload, err := cb()
		if err != nil {
			err = errors.Wrapf(err, "Err reading from byte reader")
			log.Println(err.Error())
			break
		}
		if payload != nil {
			if m.skipMessages > 0 && skipped < m.skipMessages {
				skipped++
				count++
				continue
			}

			if len(payload) > 0 {
				payload = append([]byte(m.prepend), payload...)
				payload = append(payload, []byte(m.append)...)

				outc <- payload

				if m.maxMessages > 0 && count >= m.maxMessages {
					break
				}
			}

			count++
		}
	}
}

func NewByte() (Module) {
	return &Byte{}
}

func (m *Byte) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.IntVar(&m.messageSize, "message-size", -1, "Split stream into messages of byte length. Mutually exclusive with \"--delimiter\"")
	fs.StringVar(&m.delimFlag, "delimiter", "", "Split stream into messages delimited by specified by the regexp delimiter. Mutually exclusive with \"--message-size\"")
	fs.IntVar(&m.skipMessages, "skip-messages", 0, "Skip x messages after splitting")
	fs.IntVar(&m.maxMessages, "max-messages", 0, "Stream x messages after skipped messages")
	fs.StringVar(&m.append, "append", "", "Append string to messages")
	fs.StringVar(&m.prepend, "prepend", "", "Prepend string to messages")
}
