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
)

func init() {
	MODULELIST.Register("elasticsearch-put", "Insert to elasticsearch from JSON", NewElasticsearchPut)
}

type ElasticsearchPut struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	fs *pflag.FlagSet
	client *elastic.Client
	stdin io.WriteCloser
	stdout io.ReadCloser
	flags *ElasticsearchPutFlags
	cancel chan struct{}

	// Set to true only once by only one goroutine if querying needs to stop 
	close bool
}

type ElasticsearchPutFlags struct {
	Version int
	Server string
	Index string
	Create bool
	Raw bool
	BulkSize int
	BulkActions int
	FlushInterval time.Duration
}

func (m *ElasticsearchPut) SetFlagSet(fs *pflag.FlagSet) {
	m.flags = &ElasticsearchPutFlags{}

	fs.IntVar(&m.flags.Version, "version", 5, "Set the elasticsearch library version")
	fs.StringVar(&m.flags.Server, "server", "http://localhost:9200", "Specify elasticsearch server to query")
	fs.StringVar(&m.flags.Index, "index", "", "Default index to write to. Uses \"_index\" if found in input")
	fs.BoolVar(&m.flags.Create, "create", false, "Fail if the document ID already exists")
	fs.BoolVar(&m.flags.Raw, "raw", false, "Use the json as the _source directly, automatically generating ids. Expects \"--index\" and \"--type\" to be present")
	fs.IntVar(&m.flags.BulkActions, "bulk-actions", 500, "Max bulk actions when indexing")
	fs.DurationVar(&m.flags.FlushInterval, "flush-interval", 5 * time.Second, "Max interval duration between two bulk requests")
	fs.IntVar(&m.flags.BulkSize, "bulk-size", 10 << 20 /* 10m */, "Max bulk size in bytes when indexing")
}

func (m *ElasticsearchPut) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *ElasticsearchPut) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *ElasticsearchPut) Init(global *GlobalFlags) (error) {
	switch version := m.flags.Version; version {
		case 5:
		default:
			return errors.Errorf("Version %d is not supported", version)
	}

	if m.flags.Raw && m.flags.Index == "" {
		return errors.Errorf("Flag %q cannot be empty when %q is set", "--index", "--raw")
	}

	if m.flags.BulkSize < 1 << 20 {
		return errors.Errorf("Flag %q has to be at least %d", "--bulk-size", 1 << 20)
	}

	if m.flags.BulkActions < 1 {
		return errors.Errorf("Flag %q has to be at least 1", "--bulk-actions")
	}

	if m.flags.FlushInterval < 0 {
		return errors.Errorf("Duration for flag %q cannot be negative", "--flush-interval")
	}

	setURL := elastic.SetURL(m.flags.Server)

	var err error
	m.client, err = elastic.NewClient(setURL, elastic.SetSniff(false))
	if err != nil {
		return errors.Wrapf(err, "Err creating connection to server %s", m.flags.Server)
	}

	return nil
}

type ElasticsearchPutRawMessage struct {
	Timestamp time.Time `json:"@timestamp"`
	Message string `json:"message"`
}

func (m *ElasticsearchPut) Start() {
	m.sync.Add(1)

	reader, writer := io.Pipe()

	go func() {
		// set to true when receiving messages
		started := false

		defer func() {
			writer.Close()

			<- m.cancel

			if ! started {
				m.sync.Done()
				return
			}
		}()

		previousTime := time.Now()

		for message := range m.in {
			started = true

			payload := message.Payload

			if m.flags.Raw {
				now := time.Now()

				// Make sure that events are one millisecond appart
				// otherwise elasticsearch won't sort them correctly
				if now.Sub(previousTime) < time.Millisecond {
					time.Sleep(time.Millisecond)
					now = time.Now()
				}

				// TODO: use template maybe?
				source, _ := json.Marshal(&ElasticsearchPutRawMessage{
					Message: string(payload[:]),
					Timestamp: now,
				})

				data := &ElasticsearchPutInput{
					Index: m.flags.Index,
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
	}()

	go func() {
		defer reader.Close()
		defer close(m.out)

		//TODO: uuid name
		processor, err := m.client.BulkProcessor().
			Name("my uniq name").
			Workers(1).
			BulkActions(m.flags.BulkActions).
			BulkSize(m.flags.BulkSize).
			FlushInterval(m.flags.FlushInterval).
			After(ElasticsearchPutAfterFunc(m.sync, m.out)).
			Do(context.Background())
		if err != nil {
			err = errors.Wrap(err, "Unable to setup elasticsearch bulk processor")
			log.Println(err.Error())
			m.sync.Done()
			return
		}

		defer ElasticsearchPutFlushCloseFunc(processor, m.sync)

		decoder := json.NewDecoder(reader)

		for {
			data := &ElasticsearchPutInput{}
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
				data.Index = m.flags.Index
			}

			processor.Add(elastic.NewBulkIndexRequest().
				Index(data.Index).
				Id(data.Id).
				Doc(data.Source))
		}
	}()
}

func ElasticsearchPutFlushCloseFunc(processor *elastic.BulkProcessor, wg *sync.WaitGroup) {
	defer wg.Done()

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

type ElasticsearchPutInput struct {
	Id string `json:"_id"`
	Index string `json:"_index"`
	Source *json.RawMessage `json:"_source"`
}

func (m *ElasticsearchPut) Wait() {
	m.cancel <- struct{}{}
	m.sync.Wait()

	for range m.in {}

	m.close = true
}

func NewElasticsearchPut() (Module) {
	return &ElasticsearchPut{
		sync: &sync.WaitGroup{},
		cancel: make(chan struct{}),
		close: false,
	}
}

func ElasticsearchPutAfterFunc(
	sync *sync.WaitGroup,
	out chan *Message,
) (elastic.BulkAfterFunc) {
	return func(
		executionId int64,
		requests []elastic.BulkableRequest,
		response *elastic.BulkResponse,
		err error,
	) {

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

			SendMessageLine(payload, out)
		}
	}
}
