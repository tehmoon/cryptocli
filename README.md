[![Docker Automated build](https://img.shields.io/docker/automated/tehmoon/cryptocli)](https://hub.docker.com/r/tehmoon/cryptocli)

# Cryptocli v3

IT IS BACK WITH A NEW POWERFUL DESIGN!!!

Cryptocli is the ultimate tool to data manipulation and transfer between protocols and data format.

It is often really annoying to have to use some tools on unix, other tools on windows to do simple things.

This drastically helps by setting up a pipeline where data flows from sources to sinks or gets modified.

Use cryptocli on multiple platform thanks to the pure Golang implementation of modules.

Cryptocli also support multi streams, meaning that you can now have multiple clients that will be
connected to separate pipelines.

## Features

  - Multi client support for servers
  - Module chaining with data pipeline approach
  - Feedback loop in pipeline
  - Multi OS support
  - Single executable without any dependencies
  - Lightweight Docker image

## Executables

Run it on docker:

```
docker run --rm tehmoon/cryptocli
```

Find executables in the [release page](https://github.com/tehmoon/cryptocli/releases)

## CAVEATS

- Elasticsearch modules are compatible > 7.x

## Contributing

All PR are welcome!

If you have an idea, feature request, bug, please file an issue!

## Examples

### Stdin -> tcp-server -> stdout with line buffering

```
cryptocli -- \
  -- stdin \
  -- tcp-server \
    --listen :8080 \
  -- stdout
```

### tcp-server -> tcp-server: chain both tcp servers togethers

```
cryptocli \
  --multi-streams \
  -- tcp-server --listen :8080 \
  -- tcp-server --listen :8081
```

### http -> file: Get a webpage then uppercase ascii and send the result to a file

```
cryptocli \
  -- http --url https://google.com \
  -- upper \
  -- write-file \
    --path /tmp/google.com \
    --mode 0600
```

### tcp-server -> http: proxy tcp to http

```
cryptocli \
  --multi-streams \
  -- tcp-server \
    --listen :8081 \
  -- http \
    --url http://localhost:8080
```

### tcp-server -> tcp --tls: tcp proxy to tls

```
cryptocli 
  --multi-streams \
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
cryptocli \
  -- http \
    --url https://google.com \
  -- tee \
    --pipe "dgst --algo sha256 -- hex --encode -- stdout" \
  -- file \
    --path /tmp/blih \
    --write
```

### stdin -> aes-gcm encrypt -> tcp-server -> aes-gcm decryt -> stdout: setup a tcp-server with aes-gcm encryption

```
pwd=DEADBEEF ./cryptocli \
  --multi-streams \
  -- stdin \
  -- aes-gcm \
    --encrypt \
    --password-in "env --var pwd" \
  -- tcp-server \
    --listen :8080 \
  -- aes-gcm \
    --decrypt \
    --password-in "env --var pwd" \
  -- stdou
```

### stdin -> byte -> elasticsearch-put -> stdout: save each line in elasticsearch

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
  -- byte \
    --delimiter $'\n'
  -- write-elasticsearch \
    --index bluh \
    --server http://localhost:9200 \
    --raw \
  -- stdout
```

### elasticsearch-get -> fork -> elasticsearch-put -> stdout: query elasticsearch for the last 15 minutes then stream and save the data to another custer with new \_id, \_type  and \_index

```
cryptocli \
  -- query-elasticsearch \
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
      --max-concurrent-streams int   Max number of concurrent streams. Highier increase bandwidth at the cost of memory and CPU. (default 25)
      --multi-streams                Enable multi streams modules. Warning, some modules might be blocked waiting for  input data that will never come
      --std                          Read from stdin and writes to stdout instead of setting both modules
List of all modules:
  env: Read an environment variable
  http: Makes HTTP requests
  read-s3: Read a file from s3
  tee: Create a new one way pipeline to copy the data over
  lower: Lowercase all ascii characters
  null: Discard all incoming data
  upper: Uppercase all ascii characters
  byte: Byte manipulation module
  stdin: Reads from stdin
  write-s3: uploads a file to s3
  fork: Start a program and attach stdin and stdout to the pipeline
  gzip: Gzip compress
  http-server: Create an http web webserver
  read-file: Read file from filesystem
  tcp: Connects to TCP
  aes-gcm: AES-GCM encryption/decryption
  query-elasticsearch: Send query to elasticsearch cluster and output result in json line
  stdout: Writes to stdout
  websocket: Connect using the websocket protocol
  write-file: Writes to a file.
  gunzip: Gunzip de-compress
  base64: Base64 decode or encode
  tcp-server: Listens TCP and wait for a single connection to complete
  unzip: Buffer the zip file to disk and read selected file patterns.
  dgst: Dgst decode or encode
  hex: Hex encoding/decoding
  websocket-server: Create an http websocket server
  write-elasticsearch: Insert to elasticsearch from JSON
```

### Modules

```
Usage of module "env":
      --var string   Variable to read from
```
```
Usage of module "http":
      --data                    Read data from the stream and send it before reading the response
      --header stringArray      Set header in the form of "header: value"
      --insecure                Don't verify the tls certificate chain
      --method string           HTTP Verb (default "GET")
      --read-timeout duration   Read timeout for the tcp connection (default 15s)
      --url string              HTTP server to connect to
```
```
Usage of module "read-s3":
      --bucket string   Specify the bucket name
      --path string     Object path
```
```
Usage of module "tee":
      --pipe string   Pipeline definition
```
```
Usage of module "lower":
```
```
Usage of module "null":
```
```
Usage of module "upper":
```
```
Usage of module "byte":
      --append string       Append string to messages
      --delimiter string    Split stream into messages delimited by specified by the regexp delimiter. Mutually exclusive with "--message-size"
      --max-messages int    Stream x messages after skipped messages
      --message-size int    Split stream into messages of byte length. Mutually exclusive with "--delimiter" (default 16384)
      --prepend string      Prepend string to messages
      --skip-messages int   Skip x messages after splitting
```
```
Usage of module "stdin":
```
```
Usage of module "write-s3":
      --bucket string   Specify the bucket name
      --path string     Object path
```
```
Usage of module "fork":
```
```
Usage of module "gzip":
```
```
Usage of module "http-server":
      --addr string                     Listen on an address
      --connect-timeout duration        Max amount of time to wait for a potential connection when pipeline is closing (default 30s)
      --iddle-timeout duration          IdleTimeout is the maximum amount of time to wait for the next request when keep-alives are enabled (default 5s)
      --read-headers-timeout duration   Set the amount of time allowed to read request headers. (default 15s)
      --read-timeout duration           Set the maximum duration for reading the entire request, including the body. (default 15s)
      --write-timeout duration          Set maximum duration before timing out writes of the response (default 15s)
```
```
Usage of module "read-file":
      --path string   File's path
```
```
Usage of module "tcp":
      --addr string             Tcp address to connect to
      --insecure                Don't verify certificate chain when "--servername" is set
      --read-timeout duration   Read timeout for the tcp connection (default 3s)
      --tls string              Use TLS with servername in client hello
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
Usage of module "query-elasticsearch":
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
      --tail-interval duration   Time to wait before querying elasticsearch again when using "--tail" (default 1s)
      --tail-max duration        Maximum time to wait before exiting the "--tail" loop (default 2562047h47m16.854775807s)
      --timestamp-field string   Timestamp field (default "@timestamp")
      --to string                Elasticsearch date for lt. Has not effect when "--tail" is used (default "now")
```
```
Usage of module "stdout":
```
```
Usage of module "websocket":
      --close-timeout duration   Timeout to wait for after sending the closure message (default 15s)
      --header stringArray       Set header in the form of "header: value"
      --insecure                 Don't verify the tls certificate chain
      --ping-interval duration   Interval of time between ping websocket messages (default 30s)
      --read-timeout duration    Read timeout for the websocket connection (default 15s)
      --text                     Set the websocket message's metadata to text
      --url string               Websocket server to connect to
```
```
Usage of module "write-file":
      --append        Append data instead of truncating when writting
      --mode uint32   Set file's mode if created when writting (default 416)
      --path string   File's path
```
```
Usage of module "gunzip":
```
```
Usage of module "base64":
      --decode   Base64 decode
      --encode   Base64 encode
```
```
Usage of module "tcp-server":
      --certificate string         Path to certificate in PEM format
      --connect-timeout duration   Max amount of time to wait for a potential connection when pipeline is closing (default 30s)
      --key string                 Path to private key in PEM format
      --listen string              Listen on addr:port. If port is 0, random port will be assigned
      --read-timeout duration      Amout of time to wait reading from the connection (default 15s)
```
```
Usage of module "unzip":
      --pattern stringArray   Read the file each time it matches a pattern. (default [.*])
```
```
Usage of module "dgst":
      --algo string   Hash algorithm to use: md5, sha1, sha256, sha512, sha3_224, sha3_256, sha3_384, sha3_512, blake2s_256, blake2b_256, blake2b_384, blake2b_512, ripemd160
```
```
Usage of module "hex":
      --decode   Hexadecimal decode
      --encode   Hexadecimal encode
```
```
Usage of module "websocket-server":
      --addr string                     Listen on an address
      --close-timeout duration          Timeout to wait for after sending the closure message (default 15s)
      --connect-timeout duration        Max amount of time to wait for a potential connection when pipeline is closing (default 30s)
      --read-headers-timeout duration   Set the amount of time allowed to read request headers. (default 15s)
      --read-timeout duration           Read timeout for the websocket connection (default 15s)
      --text                            Set the websocket message's metadata to text
```
```
Usage of module "write-elasticsearch":
      --bulk-actions int          Max bulk actions when indexing (default 500)
      --bulk-size int             Max bulk size in bytes when indexing (default 10485760)
      --create                    Fail if the document ID already exists
      --flush-interval duration   Max interval duration between two bulk requests (default 5s)
      --index string              Default index to write to. Uses "_index" if found in input
      --raw                       Use the json as the _source directly, automatically generating ids. Expects "--index" to be present
      --server string             Specify elasticsearch server to query (default "http://localhost:9200")
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
