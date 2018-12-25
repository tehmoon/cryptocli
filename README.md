# Cryptocli v2

IT IS BACK WITH A NEW POWERFUL DESIGN!!!

Cryptocli is a modern swiss army knife to data manipulation within complex pipelines.

It is often really annoying to have to use some tools on unix, others on windows to do simple things.

This drastically helps by setting up a pipeline where data flows from sources to sinks or gets modified.

Use cryptocli on multiple platform thanks to the pure Golang implementation of modules.

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
  tcp-server --listen :8080 \
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

## Usage

By setting the help flags to each module:

```
cryptocli -- http -h -- tcp
```

It will stop and show the help until there are no help flags remaining. 

### Cryptocli

```
Usage of cryptocli: [options] -- <module> [options] -- <module> [options] -- ...
      --std   Read from stdin and writes to stdout instead of setting both modules
List of all modules:
  fork: Start a program and attach stdin and stdout to the pipeline
  s3: Downloads or uploads a file from s3
  stdin: Reads from stdin
  stdout: Writes to stdout
  lower: Lowercase all ascii characters
  tcp-server: Listens TCP and wait for a single connection to complete
  upper: Uppercase all ascii characters
  base64: Base64 decode or encode
  file: Reads from a file or write to a file.
  gunzip: Gunzip de-compress
  gzip: Gzip compress
  hex: Hex de-compress
  http-server: Create an http web webserver
  tcp: Connects to TCP
  dgst: Dgst decode or encode
  http: Connects to an HTTP webserver
  null: Discard all incoming data
```

### Modules

```
Usage of module "s3":
      --bucket string   Specify the bucket name
      --path string     Object path
      --read            Read from s3
      --write           Write to s3
```
```
Usage of module "tcp":
      --addr string   Tcp address to connect to
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
Usage of module "http-server":
      --addr string   Listen on an address
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
Usage of module "http":
      --data            Send data from the pipeline to the server
      --insecure        Don't valid the TLS certificate chain
      --method string   Set the method to query (default "GET")
      --url string      HTTP url to query
```
```
Usage of module "stdin":
```
```
Usage of module "tcp-server":
      --listen string   Listen on addr:port. If port is 0, random port will be assigned
```
```
Usage of module "base64":
      --decode   Base64 decode
      --encode   Base64 encode
```
```
Usage of module "gzip":
```
```
Usage of module "null":
```
```
Usage of module "stdout":
```
```
Usage of module "upper":
```
```
Usage of module "fork":
```
```
Usage of module "gunzip":
```
```
Usage of module "lower":
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
  * tee
  * count bytes
  * pem
  * http serve file

Feature ideas:

  * tags
  * add raw tty signal for stdin when tty
