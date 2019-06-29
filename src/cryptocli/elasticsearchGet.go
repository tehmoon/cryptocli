package main

import (
	"time"
	"strconv"
	"encoding/json"
	"log"
	"github.com/tehmoon/errors"
	"github.com/spf13/pflag"
	"sync"
	elastic5 "gopkg.in/olivere/elastic.v5"
	"io"
	"context"
)

func init() {
	MODULELIST.Register("elasticsearch-get", "Query elasticsearch and output json on each line", NewElasticsearchGet)
}

type ElasticsearchGet struct {
	in chan *Message
	out chan *Message
	sync *sync.WaitGroup
	fs *pflag.FlagSet
	client *elastic5.Client
	stdin io.WriteCloser
	stdout io.ReadCloser
	flags *ElasticsearchGetFlags

	// Set to true only once by only one goroutine if querying needs to stop 
	close bool
}

type ElasticsearchGetFlags struct {
	Version int
	QueryStringQuery string
	Server string
	Index string
	From string
	To string
	Size int
	Asc bool
	CountOnly bool
	Sort string
	ScrollSize int
	TimestampField string
	Aggregation string
	Tail bool
	SortFields []string
}

func (m *ElasticsearchGet) SetFlagSet(fs *pflag.FlagSet) {
	m.flags = &ElasticsearchGetFlags{}

	fs.IntVar(&m.flags.Version, "version", 5, "Set the elasticsearch library version")
	fs.StringVar(&m.flags.From, "from", "now-15m", "Elasticsearch date for gte")
	fs.StringVar(&m.flags.To, "to", "now", "Elasticsearch date for lt. Has not effect when \"--tail\" is used")
	fs.BoolVar(&m.flags.Asc, "asc", false, "Sort by asc")
	fs.StringVar(&m.flags.Sort, "sort", "@timestamp", "Sort field")
	fs.StringVar(&m.flags.TimestampField, "timestamp-field", "@timestamp", "Timestamp field")
	fs.IntVar(&m.flags.Size, "size", 0, "Overall number of results to display, does not change the scroll size")
	fs.IntVar(&m.flags.ScrollSize, "scroll-size", 500, "Document to return between each scroll")
	fs.StringVar(&m.flags.QueryStringQuery, "query", "*", "Elasticsearch query string query")
	fs.StringVar(&m.flags.Server, "server", "http://localhost:9200", "Specify elasticsearch server to query")
	fs.StringVar(&m.flags.Index, "index", "", "Specify the elasticsearch index to query")
	fs.BoolVar(&m.flags.CountOnly, "count-only", false, "Only displays the match number")
	fs.StringVar(&m.flags.Aggregation, "aggregation", "", "Elastic Aggregation query")
	fs.BoolVar(&m.flags.Tail, "tail", false, "Query Elasticsearch in tail -f style. Deactivate the flag \"--to\"")
	fs.StringArrayVar(&m.flags.SortFields, "sort-field", make([]string, 0), "Additional fields to sort on")
}

func (m *ElasticsearchGet) In(in chan *Message) (chan *Message) {
	m.in = in

	return in
}

func (m *ElasticsearchGet) Out(out chan *Message) (chan *Message) {
	m.out = out

	return out
}

func (m *ElasticsearchGet) Init(global *GlobalFlags) (error) {
	if m.flags.Index == "" {
		return errors.Errorf("Flag %q is required", "--index")
	}

	if m.flags.TimestampField == "" {
		return errors.Errorf("Flag %q cannot be empty", "--timestamp-field")
	}

	if m.flags.To == "" {
		return errors.Errorf("Flag %q cannot be empty", "--to")
	}

	if m.flags.From == "" {
		return errors.Errorf("Flag %q cannot be empty", "--from")
	}

	if m.flags.Size < 0 {
		return errors.Errorf("Flag %q cannot be negative", "--size")
	}

	if m.flags.ScrollSize < 1 {
		return errors.Errorf("Flag %q cannot be less than 1", "--scroll-size")
	}

	if m.flags.ScrollSize > m.flags.Size && m.flags.Size != 0 {
		m.flags.ScrollSize = m.flags.Size
	}

	setURL := elastic5.SetURL(m.flags.Server)

	var err error

	switch version := m.flags.Version; version {
		case 5:
			m.client, err = elastic5.NewClient(setURL, elastic5.SetSniff(false))
			if err != nil {
				return errors.Wrapf(err, "Err creating connection to server %s", m.flags.Server)
			}
		default:
			return errors.Errorf("Version %d is not supported", version)
	}


	return nil
}

func ElasticsearchGetGenerateBoolQuery(flags *ElasticsearchGetFlags, gte bool) (bq *elastic5.BoolQuery) {
		qs := elastic5.NewQueryStringQuery(flags.QueryStringQuery)
		rq := elastic5.NewRangeQuery(flags.TimestampField)

		if ! flags.Tail {
			rq.Lt(flags.To)
		}

		if gte {
			rq.Gte(flags.From)
		} else {
			rq.Gt(flags.From)
		}

		return elastic5.NewBoolQuery().Must(qs, rq)
}

func ElasticsearchGetDo(args *ElasticsearchGetFuncArgs, out chan *Message, close *bool) (ts string, err error) {
	if args.Flags.Aggregation == "" {
		ts, err = ElasticsearchGetDoSearch(args, out, close)
		if err != nil {
			return ts, errors.Wrap(err, "Error in search")
		}

		return ts, nil
	}

	ts, err = ElasticsearchGetDoAggregation(args, out)
	if err != nil {
		return ts, errors.Wrap(err, "Error in aggregation")
	}

	return ts, nil
}

func (m *ElasticsearchGet) Start() {
	go func() {
		defer close(m.out)

		args := &ElasticsearchGetFuncArgs{
			Client: m.client,
			Flags: m.flags,
			BoolQuery: ElasticsearchGetGenerateBoolQuery(m.flags, true),
		}

		ts, err := ElasticsearchGetDo(args, m.out, &m.close)
		if err != nil {
			log.Println(err.Error())
			return
		}

		if args.Flags.Tail {
			for {
				if m.close {
					return
				}

				args.Flags.From = ts
				args.BoolQuery = ElasticsearchGetGenerateBoolQuery(m.flags, false)

				// TODO: create variable
				time.Sleep(time.Second * 5)

				ts, err = ElasticsearchGetDo(args, m.out, &m.close)
				if err != nil {
					log.Println(err.Error())
					return
				}
			}
		}
	}()

	go func() {
		for range m.in {}
		m.close = true
	}()
}

func ElasticsearchGetParseTimestamp(field string, hits []*elastic5.SearchHit, asc bool) (ts string) {
	pos := 0
	if asc {
		pos = len(hits) - 1
	}

	payload := *hits[pos].Source

	var hit map[string]interface{}

	err := json.Unmarshal(payload, &hit)
	if err != nil {
		log.Println(errors.Wrap(err, "Un-expected unable to unmarshal source to json"))
		return ""
	}

	if timestamp, found := hit[field]; found {
		if timestamp, ok := timestamp.(string); ok {
			return timestamp
		}
	}

	return ""
}


func ElasticsearchGetDoSearch(args *ElasticsearchGetFuncArgs, out chan *Message, close *bool) (ts string, err error) {
	ts = args.Flags.From

	scroll := args.Client.Scroll(args.Flags.Index).
		Query(args.BoolQuery).
		Sort(args.Flags.Sort, args.Flags.Asc).
		Scroll("15s").
		Size(args.Flags.ScrollSize)

	for _, field := range args.Flags.SortFields {
		scroll.Sort(field, args.Flags.Asc)
	}

	res, err := scroll.Do(context.Background())
	if err != nil {
		if err != io.EOF {
			return ts, errors.Wrap(err, "Err querying elasticsearch")
		}
	}

	if res == nil || res.Hits.TotalHits == 0 {
		if args.Flags.CountOnly {
			SendMessageLine([]byte("0"), out)
		}

		return ts, nil
	}

	scrollId := res.ScrollId
	defer ElasticsearchGetClearScroll(args.Client, scrollId)

	ts = ElasticsearchGetParseTimestamp(args.Flags.TimestampField, res.Hits.Hits, args.Flags.Asc)

	if args.Flags.CountOnly {
		var totalHits int64 = 0

		totalHits = res.Hits.TotalHits

		SendMessageLine([]byte(strconv.FormatInt(totalHits, 10)), out)

		return ts, nil
	}

	counter := 0

	for i := 0; (counter != args.Flags.Size || counter == 0); i++ {
		if *close { return ts, nil }

		if i == len(res.Hits.Hits) {
			break
		}

		payload, err := json.Marshal(res.Hits.Hits[i])
		if err != nil {
			return ts, errors.Wrap(err, "Un-excepted error marshaling hit")
		}

		SendMessageLine(payload, out)

		counter++
	}

	if counter == args.Flags.Size && counter != 0 {
		return ts, nil
	}

	LOOP: for {
		if *close { return ts, nil }

		res, err := args.Client.Scroll(args.Flags.Index).
			Query(args.BoolQuery).
			Scroll("15s").
			ScrollId(scrollId).
			Do(context.Background())
		if err != nil {
			if err == io.EOF {
				break LOOP
			}

			return ts, errors.Wrap(err, "Err querying elasticsearch")
		}

		if args.Flags.Asc {
			ts = ElasticsearchGetParseTimestamp(args.Flags.TimestampField, res.Hits.Hits, args.Flags.Asc)
		}

		for i := 0; (counter != args.Flags.Size || counter == 0); i++ {
			if *close { return ts, nil }

			if i == len(res.Hits.Hits) {
				break
			}

			payload, err := json.Marshal(res.Hits.Hits[i])
			if err != nil {
				return ts, errors.Wrap(err, "Un-excepted error marshaling hit")
			}

			SendMessageLine(payload, out)

			counter++
		}

		scrollId = res.ScrollId

		if counter == args.Flags.Size && counter != 0 {
			break LOOP
		}
	}

	return ts, nil
}

func ElasticsearchGetClearScroll(client *elastic5.Client, id string) (err error) {
	_, err = client.ClearScroll(id).
		Do(context.Background())
	if err != nil {
		return errors.Wrapf(err, "Failed to clear the scrollid %s", id)
	}

	return nil
}

type ElasticsearchGetFuncArgs struct {
	Client *elastic5.Client
	Flags *ElasticsearchGetFlags
	BoolQuery *elastic5.BoolQuery
}

func (m *ElasticsearchGet) Wait() {
	for range m.in {}

	// This will trigger to stop next query loop
}

func NewElasticsearchGet() (Module) {
	return &ElasticsearchGet{
		sync: &sync.WaitGroup{},
		close: false,
	}
}

type ElasticsearchGetStringAggregation struct{
	body string
}

func (a ElasticsearchGetStringAggregation) Source() (v interface{}, err error) {
	err = json.Unmarshal([]byte(a.body), &v)

	return v, err
}

func ElasticsearchGetDoAggregation(args *ElasticsearchGetFuncArgs, out chan *Message) (ts string, err error) {
	aggregation := &ElasticsearchGetStringAggregation{
		body: args.Flags.Aggregation,
	}

	ts = args.Flags.From

	res, err := args.Client.Search(args.Flags.Index).
		Query(args.BoolQuery).
		Size(1).
		Sort(args.Flags.Sort, false).
		Aggregation("root", aggregation).
		Do(context.Background())
	if err != nil {
		if err != io.EOF {
			return ts, errors.Wrap(err, "Err querying elasticsearch")
		}
	}

	if res == nil {
		if args.Flags.CountOnly {
			SendMessageLine([]byte("0"), out)
		}

		return ts, nil
	}

	if len(res.Hits.Hits) == 1 {
		ts = ElasticsearchGetParseTimestamp(args.Flags.TimestampField, res.Hits.Hits, args.Flags.Asc)
	}

	if res.Aggregations != nil {
		if args.Flags.CountOnly {
			SendMessageLine([]byte("0"), out)

			return ts, nil
		}

		payload, err := json.Marshal(res.Aggregations)
		if err != nil {
			return ts, errors.Wrap(err, "Aggregations results are empty")
		}

		SendMessageLine(payload, out)
	}

	return ts, nil
}
