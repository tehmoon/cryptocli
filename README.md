[![Docker Automated build](https://img.shields.io/docker/automated/tehmoon/cryptocli)](https://hub.docker.com/r/tehmoon/cryptocli)
[![GitHub stars](https://img.shields.io/github/stars/tehmoon/cryptocli?style=social)](https://github.com/tehmoon/cryptocli/stargazers)
[![GitHub Release Date](https://img.shields.io/github/release-date/tehmoon/cryptocli)](https://github.com/tehmoon/cryptocli/releases/tag/latest)
[![GitHub last commit](https://img.shields.io/github/last-commit/tehmoon/cryptocli)](https://github.com/tehmoon/cryptocli/commits/master)
[![GitHub issues](https://img.shields.io/github/issues/tehmoon/cryptocli)](https://github.com/tehmoon/cryptocli/issues)
[![Twitter Follow](https://img.shields.io/twitter/follow/moonbocal?style=social)](https://twitter.com/moonbocal)

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


## New pwn module!

This super exciting module enables scripting in Go!

It does this using [otto](https://github.com/robertkrimen/otto).

You will need a `start` function that is the main loop for the module. The pipeline stops when the function terminates.

Anytime the return value will be `undefined` it means that it failed.

Here's a list of builtins:

  - fromJSON (string) (object): json deserialize
  - toJSON (object) (string): json serialize
  - read (integer) (string): Read x bytes from input
  - readline () (string): Read line from input
  - write(string): Write string to the output 
  - sleep(integer): Sleep for x seconds
  - log(string): Log the string to stderr
  - readToPipe(string): The first argument is the pipeline definition. It returns a string that is all the data from that pipeline.
  - writeToPipe(string, string): The first argument is the pipeline definition, the second is the string to be written to that pipeline.
  - pipe(string, string): Start a new pipeline. The first argument is the pipeline definition. The second argument is the name of the callback that will be executed. In that callback, all attempt to read and write will do this on the new pipeline, not the parent. If the callback is omited, the pipe will patch the pipeline to the current pipeline, blocking execution until done. Note that after that, the pipeline will be empty and no read/write can be done
  - regexp(string, string, string): Use the go regexp package to perform regular expression and replace. The first parameter is the regexp. The second one is the string to read from, and the third one is the replace string. It returns a string that is the result of the replace.

### Example:

```
var user_id = ""
var url = ""

function start(metadata) {
  // Write to stdout
  writeToPipe("stdout", "What's your name: ")

  // Read the response from stdin and remove any new line.
  user_id = readFromPipe("stdin -- byte --max-messages 1")
  user_id = regexp("\n", user_id, "")
  
  // Start tcp server and pivot to the function name callback
  pipe("tcp-server --read-timeout 10h --listen :8080", "callback")

  // When callback is done executing, let me know what was the website
  log("visited " + url)
}

function callback(metadata) {
  // Display some metadata from the pipe() function
  log("metadata: " + toJSON(metadata))

  // Write to the tcp connection
  write(user_id + " wants to know what website you want to visit: ")

  // Read line from tcp connection, this will be the url to user
  url = readline()
  if (url === undefined) {
    return
  }

  // Start the http module to query the desired website
  var content = readFromPipe("http --url " + url)

  // Write the content back to the tcp connection
  write(content)

  // Pivot one last time and take control of the terminal
  pipe("stdout -- stdin")
}
```

As you can see, it's pretty awesome :)

## Metadata

Metadata is a new addition where modules when they send channels, add metadata about the current state.

It's using [go templates](https://godoc.org/text/template) and you can use it with some flags as long as the previous chained module will send some metadata. For example, it is possible to add a `path` template when writing a file so you can have one file per channel.

Right now, the implementation is really naive, you cannot get metadata for other modules than the previous one.

### Metadata Examples

HTTP redirect a client to download a file name writeFile.go on the filesystem (LFI NOT SECURE)
```
cryptocli \
  -- http-server --addr :8080 --redirect-to writeFile.go \
  -- read-file --path './{{ index . "redirect-to" }}'
```

Start an HTTP server and serve static files (LFI NOT SECURE)

```
cryptocli --multi-streams \
  -- http-server --addr :8080 \
  -- read-file --path './{{ index .url }}'
```

Start an HTTP server and write files depending on remote-addr

```
cryptocli --multi-streams \
  -- null \
  -- http-server --addr :8080 \
  -- write-file --path './blah/{{ index . "remote-addr" }}.txt'
```

Same as above but for S3

```
cryptocli  --multi-streams \
  -- null \
  -- http-server --addr :8080 \
  -- write-s3 --path '/{{ index . "remote-addr" }}' --bucket '{{ index .headers.Bucket 0 }}'
```

### Metadata Modules

#### http-server

```
"redirect-to": string
"url": string
"headers": []string
"host": string
"remote-addr": string
"request-uri": string
"addr": string
```

#### query-elasticsearch

```
"query": string
"index": string
"from": string
"to": string
"aggregation": string
"timestamp-field": string
```

#### read-file

```
"path": string
```

#### read-s3

```
"path": string
"bucket": string
```

#### tcp-server

```
"local-addr": string
"remote-addr": string
"addr": string
```

#### websocket-server

```
"url": string
"headers": []string
"host": string
"remote-addr": string
"request-uri": string
"addr": string
```

#### write-elasticsearch

```
"index": string
```

#### write-file

```
"path": string
```

#### write-s3

```
"path": string
"bucket": string
```

## Cryptocli Examples

### Websocket reverse shell

target:
```
cryptocli \
  --multi-streams \
  -- fork sh \
  -- websocket-server \
    --addr :8080
```

client:
```
cryptocli \
  -- stdin \
  -- websocket \
    --url ws://localhost:8080 \
    --read-timeout 10h 
  -- stdout
```

Tip: Add some `aes-gcm` module to make it end-to-end encrypted!

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

### Create a tcp server -> pwn module

See javascript file at the beginning of the documentation

```
  cryptocli \
    -- tcp-server \
      --listen :8080 \
      --read-timeout 10h \
    -- pwn \
      --file-pipe "read-file --path test.js"
```

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
      --version                      Show version and exits
List of all modules:
  read-file: Read file from filesystem
  http-server: Create an http web webserver
  http: Makes HTTP requests
  query-elasticsearch: Send query to elasticsearch cluster and output result in json line
  dgst: Dgst decode or encode
  pwn: Start a javascript VM to control input/output
  gunzip: Gunzip de-compress
  gzip: Gzip compress
  lower: Lowercase all ascii characters
  stdin: Reads from stdin
  upper: Uppercase all ascii characters
  write-s3: uploads a file to s3
  fork: Start a program and attach stdin and stdout to the pipeline
  tcp: Connects to TCP
  tee: Create a new one way pipeline to copy the data over
  websocket: Connect using the websocket protocol
  null: Discard all incoming data
  stdout: Writes to stdout
  unzip: Buffer the zip file to disk and read selected file patterns.
  websocket-server: Create an http websocket server
  aes-gcm: AES-GCM encryption/decryption
  hex: Hex encoding/decoding
  write-elasticsearch: Insert to elasticsearch from JSON
  base64: Base64 decode or encode
  env: Read an environment variable
  read-s3: Read a file from s3
  tcp-server: Listens TCP and wait for a single connection to complete
  write-file: Writes to a file.
  byte: Byte manipulation module
```

### Modules

```
Usage of module "read-file":
      --path string   File's path using templates
```
```
Usage of module "http-server":
      --addr string                Listen on an address
      --connect-timeout duration   Max amount of time to wait for a potential connection when pipeline is closing (default 30s)
      --file-upload                Serve a HTML page on GET / and a file upload endpoint on POST /
      --header stringArray         Set header in the form of "header: value"
      --password string            Specify the required password for basic auth
      --redirect-to string         Redirect the request to where the download can begin
      --show-client-headers        Show client headers in the logs
      --show-server-headers        Show server headers in the logs
      --user string                Specify the required user for basic auth
```
```
Usage of module "http":
      --data                    Read data from the stream and send it before reading the response
      --header stringArray      Set header in the form of "header: value"
      --insecure                Don't verify the tls certificate chain
      --method string           HTTP Verb (default "GET")
      --password string         Specify the required password for basic auth
      --read-timeout duration   Read timeout for the tcp connection (default 15s)
      --show-client-headers     Show client headers in the logs
      --show-server-headers     Show server headers in the logs
      --url string              HTTP server to connect to
      --user string             Specify the required user for basic auth
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
Usage of module "dgst":
      --algo string   Hash algorithm to use: md5, sha1, sha256, sha512, sha3_224, sha3_256, sha3_384, sha3_512, blake2s_256, blake2b_256, blake2b_384, blake2b_512, ripemd160
```
```
Usage of module "pwn":
      --file-pipe "read-file --path test.js"   Read content of the file from a pipeline. IE: "read-file --path test.js"
```
```
Usage of module "gunzip":
```
```
Usage of module "gzip":
```
```
Usage of module "lower":
```
```
Usage of module "stdin":
```
```
Usage of module "upper":
```
```
Usage of module "write-s3":
      --bucket string   Specify the bucket name using metadata
      --path string     Object path using metadata
```
```
Usage of module "fork":
```
```
Usage of module "tcp":
      --addr string             Tcp address to connect to
      --insecure                Don't verify certificate chain when "--servername" is set
      --read-timeout duration   Read timeout for the tcp connection (default 3s)
      --tls string              Use TLS with servername in client hello
```
```
Usage of module "tee":
      --pipe string   Pipeline definition
```
```
Usage of module "websocket":
      --close-timeout duration   Timeout to wait for after sending the closure message (default 15s)
      --header stringArray       Set header in the form of "header: value"
      --insecure                 Don't verify the tls certificate chain
      --ping-interval duration   Interval of time between ping websocket messages (default 30s)
      --read-timeout duration    Read timeout for the websocket connection (default 15s)
      --show-client-headers      Show client headers in the logs
      --show-server-headers      Show server headers in the logs
      --text                     Set the websocket message's metadata to text
      --url string               Websocket server to connect to
```
```
Usage of module "null":
```
```
Usage of module "stdout":
```
```
Usage of module "unzip":
      --pattern stringArray   Read the file each time it matches a pattern. (default [.*])
```
```
Usage of module "websocket-server":
      --addr string                     Listen on an address
      --close-timeout duration          Timeout to wait for after sending the closure message (default 15s)
      --connect-timeout duration        Max amount of time to wait for a potential connection when pipeline is closing (default 30s)
      --header stringArray              Set header in the form of "header: value"
      --read-headers-timeout duration   Set the amount of time allowed to read request headers. (default 15s)
      --read-timeout duration           Read timeout for the websocket connection (default 15s)
      --show-client-headers             Show client headers in the logs
      --show-server-headers             Show server headers in the logs
      --text                            Set the websocket message's metadata to text
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
Usage of module "hex":
      --decode   Hexadecimal decode
      --encode   Hexadecimal encode
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
```
Usage of module "base64":
      --decode   Base64 decode
      --encode   Base64 encode
```
```
Usage of module "env":
      --var string   Variable to read from
```
```
Usage of module "read-s3":
      --bucket string   Specify the bucket name using metadata
      --path string     Object path using metadata
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
Usage of module "write-file":
      --append        Append data instead of truncating when writting
      --mode uint32   Set file's mode if created when writting (default 416)
      --path string   Metadata template for file path
```
```
Usage of module "byte":
      --append string       Append string to messages
      --delimiter string    Split stream into messages delimited by specified by the regexp delimiter. Mutually exclusive with "--message-size"
      --max-messages int    Stream x messages after skipped messages
      --message-size int    Split stream into messages of byte length. Mutually exclusive with "--delimiter" (default -1)
      --prepend string      Prepend string to messages
      --skip-messages int   Skip x messages after splitting
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
