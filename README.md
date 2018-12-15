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
cryptocli --std --line -- \
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

### tcp-server -> http: proxy tcp to http with line buffering

```
cryptocli --line -- \
  tcp-server --listen :8081 -- \
  http --url http://localhost:8080 --data
```

## Usage

By setting the help flags to each module:

```
cryptocli -- http -h -- tcp
```

It will stop and show the help until there are no help flags remaining. 

```
Usage of cryptocli [options] -- <module> [options] -- <module> [options] -- ...
      --line   Read buffer per line for all modules
      --std    Read from stdin and writes to stdout instead of setting both modules
List of all modules:
  file: Reads from a file or write to a file.
  s3: Downloads or uploads a file from s3
  stdin: Reads from stdin
  tcp: Connects to TCP
  upper: Uppercase all ascii characters
  http: Connects to an HTTP webserver
  lower: Lowercase all ascii characters
  stdout: Writes to stdout
  tcp-server: Listens TCP and wait for a single connection to complete
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
  * fork
  * count bytes
  * http-server
  * pem
  * dgst

Feature ideas:

  * tags
