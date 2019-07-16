# Cryptocli v2

IT IS BACK WITH A NEW POWERFUL DESIGN!!!

Cryptocli is a modern swiss army knife to data manipulation within complex pipelines.

It is often really annoying to have to use some tools on unix, others on windows to do simple things.

This drastically helps by setting up a pipeline where data flows from sources to sinks or gets modified.

Use cryptocli on multiple platform thanks to the pure Golang implementation of modules.

## CAVEATS

- Elasticsearch modules are compatible > 7.x

## WIP

This is still a work in progress as I am migrating modules one by one with new features along the way!

## Examples

### Stdin -> tcp-server -> stdout with line buffering

```
cryptocli --std -- \
  tcp-server --listen :8080
```

### tcp-server -> tcp-server: chain both tcp servers togethers

```
cryptocli -- \
  tcp-server --listen :8080 -- \
  tcp-server --listen :8081
```

### http -> file: Get a webpage then uppercase ascii and send the result to a file

```
cryptocli -- \
  http --url https://google.com -- \
  upper -- \
  file --write --path /tmp/google.com --mode 0600
```

### tcp-server -> http: proxy tcp to http

```
cryptocli -- \
  tcp-server --listen :8081 -- \
  http --url http://localhost:8080 --data
```

### tcp-server -> tcp --tls: tcp proxy to tls

```
cryptocli \
  -- tcp-server \
    --listen :8080 \
  -- tcp \
    --tls www.google.com \
    --addr www.google.com:443
```

```
curl \
  http://localhost:8080 \
  -HHost:\ www.google.com \
  -v
```

### http-server with tls -> http: proxy http with tls to http

```
openssl \
  req \
  -new \
  -newkey rsa:2048 \
  -sha256 \
  -days 365 \
  -nodes \
  -x509 \
  -keyout key.pem \
  -out crt.pem \
  -subj "/C=US/ST=MA/L=here/O=org/OU=test/CN=test.nonexistant"
```

```
cryptocli \
  -- tcp-server \
    --key key.pem \
    --certificate crt.pem \
    --listen :8080 \
  -- http \
    --url http://www.perdu.com
```

```
curl \
  -k \
  https://localhost:8080
```

### http -> ( tee -> dgst -> hex -> stdout ) , file

```
cryptocli -- \
  http --url https://google.com -- \
  tee --pipe "dgst --algo sha256 -- hex --encode -- stdout" -- \
  file --path /tmp/blih --write
```

### stdin -> aes-gcm encrypt -> tcp-server -> aes-gcm decryt -> stdout: setup a tcp-server with aes-gcm encryption

```
pwd=DEADBEEF ./cryptocli --std  -- \
  aes-gcm --encrypt --password-in "env --var pwd" -- \
  tcp-server --listen :8080 -- \
  aes-gcm --decrypt --password-in "env --var pwd"
```

### stdin -> line -> elasticsearch-put -> stdout: save each line in elasticsearch

Creates a JSON data structure like this:

```
{
  "@timestamp": "2019-06-16T13:38:01.488366-04:00",
  "message": "9"
}
```

This is ordered by `@timestamp`. Since elasticsearch does not have a timestamp nanosecond precision, when generating the timestamp, it'll wait one millisecond if the last generated timestamp is too close.


```
seq 1 10 | \
cryptocli \
  -- stdin \
  -- line \
  -- elasticsearch-put \
    --index bluh \
    --type blah \
    --server http://localhost:9200 \
    --raw \
  -- stdout
```

### elasticsearch-get -> fork -> elasticsearch-put -> stdout: query elasticsearch for the last 15 minutes then stream and save the data to another custer with new \_id, \_type  and \_index

```
cryptocli \
  -- elasticsearch-get \
    --index beats \
    --server http://localhost:9201 \
    --query 'fields.type: "pv-logs-s3"' \
    --tail \
  -- fork jq \
    --unbuffered \
    -cr '
      del(._id) |
      del(._type) |
      del(._index)' \
  -- elasticsearch-put \
    --server http://localhost:9200 \
    --type blah \
    --index blih \
  -- stdout
```

Remove the `fork` module or change it if you want to copy the index to another cluster without any changes.

## Usage

By setting the help flags to each module:

```
cryptocli -- http -h -- tcp
```

It will stop and show the help until there are no help flags remaining. 

### Cryptocli

```
Usage of ./src/cryptocli/cryptocli: [options] -- <module> [options] -- <module> [options] -- ...
      --std   Read from stdin and writes to stdout instead of setting both modules
List of all modules:
  websocket-server: Create an websocket webserver
  line: Produce messages per lines
  s3: Downloads or uploads a file from s3
  stdin: Reads from stdin
  stdout: Writes to stdout
  tcp: Connects to TCP
  tcp-server: Listens TCP and wait for a single connection to complete
  upper: Uppercase all ascii characters
  elasticsearch-get: Query elasticsearch and output json on each line
  gunzip: Gunzip de-compress
  hex: Hex de-compress
  http-server: Create an http web webserver
  null: Discard all incoming data
  tee: Create a new one way pipeline to copy the data over
  dgst: Dgst decode or encode
  env: Read an environment variable
  file: Reads from a file or write to a file.
  gzip: Gzip compress
  http: Connects to an HTTP webserver
  lower: Lowercase all ascii characters
  aes-gcm: AES-GCM encryption/decryption
  base64: Base64 decode or encode
  elasticsearch-put: Insert to elasticsearch from JSON
  fork: Start a program and attach stdin and stdout to the pipeline
  unzip: Buffer the zip file to disk and read selected file patterns.
  websocket: Connects to a websocket webserver
```

### Modules

```
Usage of module "websocket-server":
      --addr string                Listen on an address
      --close-timeout duration     Duration to wait to read the close message (default 5s)
      --connect-timeout duration   Duration to wait for a websocket connection (default 15s)
```
```
Usage of module "line":
      --new-line   Append a new line to each message
```
```
Usage of module "s3":
      --bucket string   Specify the bucket name
      --path string     Object path
      --read            Read from s3
      --write           Write to s3
```
```
Usage of module "stdin":
```
```
Usage of module "stdout":
```
```
Usage of module "tcp":
      --addr string   Tcp address to connect to
      --insecure      Don't verify certificate chain when "--tls" is set
      --tls string    Use TLS with servername in client hello
```
```
Usage of module "tcp-server":
      --certificate string         Path to certificate in PEM format
      --connect-timeout duration   Max amount of time to wait for a potential connection when pipeline is closing (default 30s)
      --key string                 Path to private key in PEM format
      --listen string              Listen on addr:port. If port is 0, random port will be assigned
```
```
Usage of module "upper":
```
```
Usage of module "elasticsearch-get":
      --aggregation string       Elastic Aggregation query
      --asc                      Sort by asc
      --count-only               Only displays the match number
      --from string              Elasticsearch date for gte (default "now-15m")
      --index string             Specify the elasticsearch index to query
      --query string             Elasticsearch query string query (default "*")
      --scroll-size int          Document to return between each scroll (default 500)
      --server string            Specify elasticsearch server to query (default "http://localhost:9200")
      --size int                 Overall number of results to display, does not change the scroll size
      --sort string              Sort field (default "@timestamp")
      --sort-field stringArray   Additional fields to sort on
      --tail                     Query Elasticsearch in tail -f style. Deactivate the flag "--to"
      --timestamp-field string   Timestamp field (default "@timestamp")
      --to string                Elasticsearch date for lt. Has not effect when "--tail" is used (default "now")
```
```
Usage of module "gunzip":
```
```
Usage of module "hex":
      --decode   Hexadecimal decode
      --encode   Hexadecimal encode
```
```
Usage of module "http-server":
      --addr string                Listen on an address
      --connect-timeout duration   Max amount of time to wait for a potential connection when pipeline is closing (default 30s)
```
```
Usage of module "null":
```
```
Usage of module "tee":
      --pipe string   Pipeline definition
```
```
Usage of module "dgst":
      --algo string   Hash algorithm to use: md5, sha1, sha256, sha512, sha3_224, sha3_256, sha3_384, sha3_512, blake2s_256, blake2b_256, blake2b_384, blake2b_512, ripemd160
```
```
Usage of module "env":
      --var string   Variable to read from
```
```
Usage of module "file":
      --append        Append data instead of truncating when writting
      --mode uint32   Set file's mode if created when writting (default 416)
      --path string   File's path
      --read          Read from a file
      --write         Write to a file
```
```
Usage of module "gzip":
```
```
Usage of module "http":
      --data            Send data from the pipeline to the server
      --insecure        Don't valid the TLS certificate chain
      --method string   Set the method to query (default "GET")
      --url string      HTTP url to query
```
```
Usage of module "lower":
```
```
Usage of module "aes-gcm":
      --128                  128 bits key (default true)
      --256                  256 bits key
      --decrypt              Decrypt
      --encrypt              Encrypt
      --password-in string   Pipeline definition to set the password
```
```
Usage of module "base64":
      --decode   Base64 decode
      --encode   Base64 encode
```
```
Usage of module "elasticsearch-put":
      --bulk-actions int          Max bulk actions when indexing (default 500)
      --bulk-size int             Max bulk size in bytes when indexing (default 10485760)
      --create                    Fail if the document ID already exists
      --flush-interval duration   Max interval duration between two bulk requests (default 5s)
      --index string              Default index to write to. Uses "_index" if found in input
      --raw                       Use the json as the _source directly, automatically generating ids. Expects "--index" and "--type" to be present
      --server string             Specify elasticsearch server to query (default "http://localhost:9200")
      --type string               Default type to use. Uses "_type" if found in input
```
```
Usage of module "fork":
```
```
Usage of module "unzip":
      --pattern stringArray   Read the file each time it matches a pattern. (default [.*])
```
```
Usage of module "websocket":
      --close-timeout duration   Duration to wait to read the close message (default 5s)
      --insecure                 Don't valid the TLS certificate chain
      --url string               HTTP url to query
```
## Design

### Pipeline

Cryptocli passes data through step in a pipeline fashion.

The main building block is a `pipeline`. It has a `in` and a `out`.

Data flows from `in` and loops back from `out`.

```
+------------------------+
|                        |
|    +--------------+    |
|    |              |    |
+--> |   Pipeline   +----+
  in |              | out
     +--------------+

```

Inside of the pipeline, there are modules.

Modules are what handles data.

Each Module's `out` is patched to the next Module's `in`.

```
        +------------+          +------------+
        |            |          |            |
+-----> |   Module   +--------> |   Module   +---->
     in |            | out   in |            | out
        +------------+          +------------+

```

Below is an example of how a module looks like.

Let's grab `tcp-server` for example. It listens on a `addr:port`, then accepts connections.

`tcp-server` will accept only one connection. Data that comes from `in` will get witten to the socket, then data that are read from the socket will get written to `out`.

```
    +----------------+
    |                |
    |   tcp-server   |
    |                |
+--------+     +---------->
 in |    |     |     | out
    |    |     |     |
    |    |     |     |
    |    v     +     |
    | write   read   |
    |     socket     |
    |                |
    +----------------+
```

## TODO:

List of modules:

  * aes
  * count bytes
  * pem
  * http serve file

Feature ideas:

  * tags
  * add raw tty signal for stdin when tty
