package main

//import (
//	"context"
//	"encoding/json"
//	"log"
//	"github.com/tehmoon/errors"
//	"time"
//	"github.com/spf13/pflag"
//	"sync"
//	"github.com/olivere/elastic"
//	"io"
//)
//
//func init() {
//	MODULELIST.Register("write-elasticsearch", "Insert to elasticsearch from JSON", NewWriteElasticsearch)
//}
//
//type WriteElasticsearch struct {
//	version int
//	server string
//	index string
//	create bool
//	raw bool
//	bulkSize int
//	bulkActions int
//	flushInterval time.Duration
//}
//
//func (m *WriteElasticsearch) Init(in, out chan *Message, global *GlobalFlags) (error) {
//	if m.raw && m.index == "" {
//		return errors.Errorf("Flag %q cannot be empty when %q is set", "--index", "--raw")
//	}
//
//	if m.bulkSize < 1 << 20 {
//		return errors.Errorf("Flag %q has to be at least %d", "--bulk-size", 1 << 20)
//	}
//
//	if m.bulkActions < 1 {
//		return errors.Errorf("Flag %q has to be at least 1", "--bulk-actions")
//	}
//
//	if m.flushInterval < 0 {
//		return errors.Errorf("Duration for flag %q cannot be negative", "--flush-interval")
//	}
//
//	setURL := elastic.SetURL(m.server)
//
//	client, err := elastic.NewClient(setURL, elastic.SetSniff(false))
//	if err != nil {
//		return errors.Wrapf(err, "Err creating connection to server %s", m.server)
//	}
//
//	go func(in, out chan *Message) {
//		wg := &sync.WaitGroup{}
//
//		LOOP: for message := range in {
//			switch message.Type {
//				case MessageTypeTerminate:
//					wg.Wait()
//					out <- message
//					break LOOP
//				case MessageTypeChannel:
//					inc, ok := message.Interface.(MessageChannel)
//					if ok {
//						outc := make(MessageChannel)
//
//						out <- &Message{
//							Type: MessageTypeChannel,
//							Interface: outc,
//						}
//
//						wg.Add(1)
//						go startWriteElasticsearch(m, client, inc, outc, wg)
//					}
//			}
//		}
//
//		wg.Wait()
//		// Last message will signal the closing of the channel
//		<- in
//		close(out)
//	}(in, out)
//
//	return nil
//}
//
//func startWriteElasticsearch(m *WriteElasticsearch, client *elastic.Client, inc, outc MessageChannel, wg *sync.WaitGroup) {
//	defer wg.Done()
//
//	reader, writer := io.Pipe()
//
//	wg.Add(1)
//	go func(m *WriteElasticsearch, writer *io.PipeWriter, inc MessageChannel, wg *sync.WaitGroup) {
//		defer wg.Done()
//		defer DrainChannel(inc, nil)
//		defer writer.Close()
//
//		previousTime := time.Now()
//
//		for payload := range inc {
//			if m.raw {
//				now := time.Now()
//
//				// Make sure that events are one millisecond appart
//				// otherwise elasticsearch won't sort them correctly
//				if now.Sub(previousTime) < time.Millisecond {
//					time.Sleep(time.Millisecond)
//					now = time.Now()
//				}
//
//				// TODO: use template maybe?
//				source, _ := json.Marshal(&WriteElasticsearchRawMessage{
//					Message: string(payload[:]),
//					Timestamp: now,
//				})
//
//				data := &WriteElasticsearchInput{
//					Index: m.index,
//					Source: (*json.RawMessage)(&source),
//				}
//
//				payload, _ = json.Marshal(&data)
//				previousTime = now
//			}
//
//			_, err := writer.Write(payload)
//			if err != nil {
//				return
//			}
//		}
//	}(m, writer, inc, wg)
//
//	wg.Add(1)
//	go func(m *WriteElasticsearch, client *elastic.Client, reader *io.PipeReader, outc MessageChannel, wg *sync.WaitGroup) {
//		defer wg.Done()
//		defer close(outc)
//		defer reader.Close()
//
//		//TODO: uuid name
//		processor, err := client.BulkProcessor().
//			Name("my uniq name").
//			Workers(1).
//			BulkActions(m.bulkActions).
//			BulkSize(m.bulkSize).
//			FlushInterval(m.flushInterval).
//			After(WriteElasticsearchAfterFunc(outc)).
//			Do(context.Background())
//		if err != nil {
//			err = errors.Wrap(err, "Unable to setup elasticsearch bulk processor")
//			log.Println(err.Error())
//			return
//		}
//
//		defer WriteElasticsearchFlushCloseFunc(processor)
//
//		decoder := json.NewDecoder(reader)
//
//		for {
//			data := &WriteElasticsearchInput{}
//			err := decoder.Decode(&data)
//			if err != nil {
//				if err == io.EOF {
//					return
//				}
//
//				err = errors.Wrapf(err, "Error unmarshaling JSON")
//				log.Println(err.Error())
//				return
//			}
//
//			if data.Index == "" {
//				data.Index = m.index
//			}
//
//			processor.Add(elastic.NewBulkIndexRequest().
//				Index(data.Index).
//				Id(data.Id).
//				Doc(data.Source))
//		}
//	}(m, client, reader, outc, wg)
//}
//
//func NewWriteElasticsearch() (Module) {
//	return &WriteElasticsearch{}
//}
//
//func (m *WriteElasticsearch) SetFlagSet(fs *pflag.FlagSet, args []string) {
//	fs.StringVar(&m.server, "server", "http://localhost:9200", "Specify elasticsearch server to query")
//	fs.StringVar(&m.index, "index", "", "Default index to write to. Uses \"_index\" if found in input")
//	fs.BoolVar(&m.create, "create", false, "Fail if the document ID already exists")
//	fs.BoolVar(&m.raw, "raw", false, "Use the json as the _source directly, automatically generating ids. Expects \"--index\" to be present")
//	fs.IntVar(&m.bulkActions, "bulk-actions", 500, "Max bulk actions when indexing")
//	fs.DurationVar(&m.flushInterval, "flush-interval", 5 * time.Second, "Max interval duration between two bulk requests")
//	fs.IntVar(&m.bulkSize, "bulk-size", 10 << 20 /* 10MiB*/, "Max bulk size in bytes when indexing")
//}
//
//type WriteElasticsearchRawMessage struct {
//	Timestamp time.Time `json:"@timestamp"`
//	Message string `json:"message"`
//}
//
//type WriteElasticsearchInput struct {
//	Id string `json:"_id"`
//	Index string `json:"_index"`
//	Source *json.RawMessage `json:"_source"`
//}
//
//func WriteElasticsearchAfterFunc(outc MessageChannel) elastic.BulkAfterFunc {
//	return func(executionId int64, requests []elastic.BulkableRequest, response *elastic.BulkResponse, err error) {
//		if err != nil {
//			log.Printf("Found error in after: %s\n", err.Error())
//			return
//		}
//
//		for _, item := range response.Items {
//			payload, err := json.Marshal(&item)
//			if err != nil {
//				err = errors.Wrapf(err, "Un-expected error marshaling %T\n", item)
//				log.Println(err.Error())
//				continue
//			}
//
//			outc <- payload
//		}
//	}
//}
//
//func WriteElasticsearchFlushCloseFunc(processor *elastic.BulkProcessor) {
//	var e error
//
//	err := processor.Flush()
//	if err != nil {
//		e = errors.Wrap(err, "Error flushing the processor")
//	}
//
//	err = processor.Close()
//	if err != nil {
//		e = errors.Wrap(err, "Error closing the processor")
//	}
//
//	if e != nil {
//		log.Println(e.Error())
//	}
//}
