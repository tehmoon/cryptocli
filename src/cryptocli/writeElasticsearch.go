package main

import (
	"context"
	"encoding/json"
	"log"
	"github.com/tehmoon/errors"
	"time"
	"github.com/spf13/pflag"
	"sync"
	"github.com/olivere/elastic"
	"io"
	"text/template"
	"bytes"
)

func init() {
	MODULELIST.Register("write-elasticsearch", "Insert to elasticsearch from JSON", NewWriteElasticsearch)
}

type WriteElasticsearch struct {
	version int
	server string
	index string
	create bool
	raw bool
	bulkSize int
	bulkActions int
	flushInterval time.Duration
}

func (m *WriteElasticsearch) Init(in, out chan *Message, global *GlobalFlags) (error) {
	if m.raw && m.index == "" {
		return errors.Errorf("Flag %q cannot be empty when %q is set", "--index", "--raw")
	}

	if m.bulkSize < 1 << 20 {
		return errors.Errorf("Flag %q has to be at least %d", "--bulk-size", 1 << 20)
	}

	if m.bulkActions < 1 {
		return errors.Errorf("Flag %q has to be at least 1", "--bulk-actions")
	}

	if m.flushInterval < 0 {
		return errors.Errorf("Duration for flag %q cannot be negative", "--flush-interval")
	}

	setURL := elastic.SetURL(m.server)

	client, err := elastic.NewClient(setURL, elastic.SetSniff(false))
	if err != nil {
		return errors.Wrapf(err, "Err creating connection to server %s", m.server)
	}

	indexTmpl, err := template.New("root").Parse(m.index)
	if err != nil {
		return errors.Wrap(err, "Error parsing template for \"--index\" flag")
	}

	go func(in, out chan *Message) {
		wg := &sync.WaitGroup{}

		init := false
		mc := NewMessageChannel()

		out <- &Message{
			Type: MessageTypeChannel,
			Interface: mc.Callback,
		}

		LOOP: for message := range in {
			switch message.Type {
				case MessageTypeTerminate:
					if ! init {
						close(mc.Channel)
					}

					wg.Wait()
					out <- message
					break LOOP
				case MessageTypeChannel:
					cb, ok := message.Interface.(MessageChannelFunc)
					if ok {
						if ! init {
							init = true
						} else {
							mc = NewMessageChannel()

							out <- &Message{
								Type: MessageTypeChannel,
								Interface: mc.Callback,
							}
						}

						wg.Add(1)
						go func() {
							defer wg.Done()
							defer close(mc.Channel)

							mc.Start(map[string]interface{}{
								"index": m.index,
							})
							metadata, inc := cb()
							defer DrainChannel(inc, nil)

							buff := bytes.NewBuffer(make([]byte, 0))
							err := indexTmpl.Execute(buff, metadata)
							if err != nil {
								err = errors.Wrap(err, "Error executing template index")
								log.Println(err.Error())
								return
							}

							index := string(buff.Bytes()[:])
							buff.Reset()

							startWriteElasticsearch(m, index, client, inc, mc.Channel)
						}()

						if ! global.MultiStreams {
							wg.Wait()
							out <- &Message{Type: MessageTypeTerminate,}
							break LOOP
						}
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

func startWriteElasticsearch(m *WriteElasticsearch, index string, client *elastic.Client, inc, outc chan []byte) {
	reader, writer := io.Pipe()

	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func(m *WriteElasticsearch, index string, writer *io.PipeWriter, inc chan []byte, wg *sync.WaitGroup) {
		defer wg.Done()
		defer DrainChannel(inc, nil)
		defer writer.Close()

		previousTime := time.Now()

		for payload := range inc {
			if m.raw {
				now := time.Now()

				// Make sure that events are one millisecond appart
				// otherwise elasticsearch won't sort them correctly
				if now.Sub(previousTime) < time.Millisecond {
					time.Sleep(time.Millisecond)
					now = time.Now()
				}

				// TODO: use template maybe?
				source, _ := json.Marshal(&WriteElasticsearchRawMessage{
					Message: string(payload[:]),
					Timestamp: now,
				})

				data := &WriteElasticsearchInput{
					Index: index,
					Source: (*json.RawMessage)(&source),
				}

				payload, _ = json.Marshal(&data)
				previousTime = now
			}

			_, err := writer.Write(payload)
			if err != nil {
				return
			}
		}
	}(m, index, writer, inc, wg)

	wg.Add(1)
	go func(m *WriteElasticsearch, client *elastic.Client, index string, reader *io.PipeReader, outc chan []byte, wg *sync.WaitGroup) {
		defer wg.Done()
		defer reader.Close()

		//TODO: uuid name
		processor, err := client.BulkProcessor().
			Name("my uniq name").
			Workers(1).
			BulkActions(m.bulkActions).
			BulkSize(m.bulkSize).
			FlushInterval(m.flushInterval).
			After(WriteElasticsearchAfterFunc(outc)).
			Do(context.Background())
		if err != nil {
			err = errors.Wrap(err, "Unable to setup elasticsearch bulk processor")
			log.Println(err.Error())
			return
		}

		defer WriteElasticsearchFlushCloseFunc(processor)

		decoder := json.NewDecoder(reader)

		for {
			data := &WriteElasticsearchInput{}
			err := decoder.Decode(&data)
			if err != nil {
				if err == io.EOF {
					return
				}

				err = errors.Wrapf(err, "Error unmarshaling JSON")
				log.Println(err.Error())
				return
			}

			if data.Index == "" {
				data.Index = index
			}

			processor.Add(elastic.NewBulkIndexRequest().
				Index(data.Index).
				Id(data.Id).
				Doc(data.Source))
		}
	}(m, client, index, reader, outc, wg)

	wg.Wait()
}

func NewWriteElasticsearch() (Module) {
	return &WriteElasticsearch{}
}

func (m *WriteElasticsearch) SetFlagSet(fs *pflag.FlagSet, args []string) {
	fs.StringVar(&m.server, "server", "http://localhost:9200", "Specify elasticsearch server to query")
	fs.StringVar(&m.index, "index", "", "Default index to write to. Uses \"_index\" if found in input")
	fs.BoolVar(&m.create, "create", false, "Fail if the document ID already exists")
	fs.BoolVar(&m.raw, "raw", false, "Use the json as the _source directly, automatically generating ids. Expects \"--index\" to be present")
	fs.IntVar(&m.bulkActions, "bulk-actions", 500, "Max bulk actions when indexing")
	fs.DurationVar(&m.flushInterval, "flush-interval", 5 * time.Second, "Max interval duration between two bulk requests")
	fs.IntVar(&m.bulkSize, "bulk-size", 10 << 20 /* 10MiB*/, "Max bulk size in bytes when indexing")
}

type WriteElasticsearchRawMessage struct {
	Timestamp time.Time `json:"@timestamp"`
	Message string `json:"message"`
}

type WriteElasticsearchInput struct {
	Id string `json:"_id"`
	Index string `json:"_index"`
	Source *json.RawMessage `json:"_source"`
}

func WriteElasticsearchAfterFunc(outc chan []byte) elastic.BulkAfterFunc {
	return func(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
		if err != nil {
			log.Printf("Found error in after: %s\n", err.Error())
			return
		}

		for _, item := range response.Items {
			payload, err := json.Marshal(&item)
			if err != nil {
				err = errors.Wrapf(err, "Un-expected error marshaling %T\n", item)
				log.Println(err.Error())
				continue
			}

			outc <- payload
		}
	}
}

func WriteElasticsearchFlushCloseFunc(processor *elastic.BulkProcessor) {
	var e error

	err := processor.Flush()
	if err != nil {
		e = errors.Wrap(err, "Error flushing the processor")
	}

	err = processor.Close()
	if err != nil {
		e = errors.Wrap(err, "Error closing the processor")
	}

	if e != nil {
		log.Println(e.Error())
	}
}
