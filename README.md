# cryptocli
A modern tool to replace dd and openssl cli using a pipeline type data flow to move bytes around.

You can read from many sources, write to many sources, decode, encode, stop a byte X, read from byte X, redirect data to many sources from different point on the pipeline and perform some command.

It'll be your next swiss army knife for sure!

Read the Usage for more explanations.

## Motivation
I use decoding/encoding tools, dd and openssl all the time. It was getting a little bit annoying to have to use shell tricks to get what I wanted.

Pull requests are of course welcome.

## Commands:

```
Commands:
  dd:  Copy input to output like the dd tool.
  dgst:  Hash the content of stdin
  pbkdf2:  Derive a key from input using the PBKDF2 algorithm
  scrypt:  Derive a key from input using the scrypt algorithm
  pipe:  Execute a command and attach stdin and stdout to the pipeline
  get-certs:  Establish tls connection and get the certificates. Doesn't use any input.
```

## Usage

```
cryptocli <command> [<options>] [<arguments>]
```

```
Usage: ./cryptocli [<Options>] 

Options:
  -chomp
    	Get rid of the last \n when not in pipe
  -decoders string
    	Set a list of codecs separated by ',' to decode input that will be process in the order given (default "binary")
  -encoders string
    	Set a list of codecs separated by ',' to encode output that will be process in the order given (default "binary")
  -filters-cmd-in string
    	List of <filter> in URL format that filters data right after -decoders
  -filters-cmd-out string
    	List of <filter> in URL format that filters data right before -encoders
  -filters-in string
    	List of <filter> in URL format that filters data right before -decoders
  -filters-out string
    	List of <filter> in URL format that filters data right after -encoders
  -from-byte-in string
    	Skip the first x bytes of stdin. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10
  -from-byte-out string
    	Skip the first x bytes of stdout. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10
  -in string
    	Input <fileType> method
  -out string
    	Output <fileType> method
  -tee-cmd-in string
    	Copy output after -decoders and before <command> to <fileType>
  -tee-cmd-out string
    	Copy output after <command> and before -encoders to <fileType>
  -tee-in string
    	Copy output before -encoders to <fileType>
  -tee-out string
    	Copy output after -encoders to <fileType>
  -to-byte-in string
    	Stop at byte x of stdin.  Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10. If you add a '+' at the begining, the value will be added to -from-byte-in
  -to-byte-out string
    	Stop at byte x of stdout. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10. If you add a '+' at the begining, the value will be added to -from-byte-out

Codecs:
  hex
	hex encode output and hex decode input
  binary
	Do nothing in input and nothing in output
  binary-string
	Take ascii string of 1 and 0 in input and decode it to binary. A byte is always 8 characters number. Does the opposite for output
  base64
	base64 decode input and base64 encode output
  gzip
	gzip compress output and gzip decompress input
  hexdump
	Encode output to hexdump -c. Doesn't support decoding

FileTypes:
  file://
	Read from a file or write to a file. Default when no <filetype> is specified. Truncate output file unless OUTFILENOTRUNC=1 in environment variable.
  pipe:
	Run a command in a sub shell. Either write to the command's stdin or read from its stdout.
  https://
	Get https url or post the output to https. Use INHTTPSNOVERIFY=1 and/or OUTHTTPSNOVERIFY=1 environment variables to disable certificate check. Max redirects count is 3. Will fail if scheme changes.
  http://
	Get http url or post the output to https. Max redirects count is 3. Will fail if scheme changes.
  env:
	Read and unset environment variable. Doesn't work for output
  readline:
	Read lines from stdin until WORD is reached.
  s3://
	Either upload or download from s3.
  null:
	Behaves like /dev/null on *nix system
  hex:
	Decode hex value and use it for input. Doesn't work for output
  ascii:
	Decode ascii value and use it for input. Doesn't work for output
  rand:
	Rand is a global, shared instance of a cryptographically strong pseudo-random generator. On Linux, Rand uses getrandom(2) if available, /dev/urandom otherwise. On OpenBSD, Rand uses getentropy(2). On other Unix-like systems, Rand reads from /dev/urandom. On Windows systems, Rand uses the CryptGenRandom API. Doesn't work with output.

Filters:
  pem
	Filter PEM objects. Options: type=<PEM type> start-at=<number> stop-at=<number>. Type will filter only PEM objects with this type. Start-at will discard the first <number> PEM objects. Stop-at will stop at PEM object <number>.
```

## Examples

Get the last 32 byte of a sha512 hash function from a hex string to base64 without last \n

```
echo -n 'DEADBEEF' | cryptocli dgst -decoder hex -encoder base64 -from-byte-out 32 -to-byte-out +32 -chomp sha512
```

Transform stdin to binary string

```
echo -n toto | cryptocli dd -encoders binary-string
```

Gzip stdin then base64 it

```
echo -n toto | cryptocli dd -encoders gzip,base64
```

Get rid of the first 2 bytes

```
echo -n toto | cryptocli dd -from-byte-in 2
```

Output the base64 hash of stdin to file

```
echo -n toto | cryptocli dgst -encoders base64 -out file://./toto.txt sha512
```

Decode base64 from file to stdout in hex

```
cryptocli dd -decoders base64 -encoders hex -in ./toto.txt
```

Gzip input, write it to file and write its sha512 checksum in hex format to another file

```
echo toto | cryptocli dd -encoders gzip -tee-out pipe:"cryptocli dgst -encoders hex -out ./checksum.txt" -out ./file.gz
```

SHA512 an https web page then POST the result to http server:

```
cryptocli dgst -in https://www.google.com -encoders hex sha512 -out http://localhost:12345/
```

Generate 32 byte salt and derive a 32 bytes key from input to `derivated-key.txt` file.

```
echo -n toto | cryptocli pbkdf2 -encoders base64 -out derivated-key.txt
```

You should have the same result as in `derivated-key.txt` file

```
echo -n toto | cryptcli pbkdf2 -salt-in pipe:"cryptocli dd -decoders base64 -to-byte-in 32" -encoders base64
```

Read key from env then scrypt it

```
key=blah cryptocli scrypt -in env:key -encoders base64
```

Hash lines read from stdin

```
cryptocli dgst -in readline:WORD -encoders hex
```

Execute nc -l 12344 which opens a tcp server and base64 the output

```
cryptocli pipe -encoders base64 nc -l 12344
```

Download an s3 object in a streaming fashion then gunzip it

```
cryptocli dd -in s3://bucket/path/to/key -decoders gzip -out key
```

Upload an s3 object, gzip it and write checksum

```
cryptocli dd -in file -encoders gzip -tee-out pipe:"cryptocli dgst -encoders hex -out file.checksum" -out s3://bucket/path/to/file
```

Filter only PEM objects of type certificate

```
cryptocli pipe -filters-cmd-out pem:type=certificate openssl s_client -connect google.com:443
```

Get first cert from tls connection

```
cryptocli get-certs google.com:443
```

Generate random 32 bytes strings using crypto/rand lib

```
cryptocli dd -in rand: -to-byte-in 32 -encoders hex
```

Set salt in pbkdf2/scrypt from hex or from ascii

```
cryptocli pbkdf2 -salt-in hex:deadbeef -encoders hex
cryptocli scrypt -salt-in ascii:deadbeef -encoders hex
```


## Internal data flow

Input -> filters-in -> tee input -> decoders -> byte counter in -> filters-cmd-in -> tee command input -> command -> filters-cmd-out -> tee command output -> byte counter out -> encoders -> filters-out -> tee output -> output

## Futur

  - redo the README.md file
  - http/https/ssl-strip proxy
  - http/https/ws/wss servers
  - tcp/tls server
  - cleanup the code
  - code coverage
  - unit tests
  - go tool suite
  - options:
    - interval
    - timeout
    - loop
    - err `redirect error`
    - debug `redirect debug`
    - delimiter pipe `from input reset pipes everytime it hits the delimiter`
  - file types:
    - tls://\<addr>
    - tcp://\<addr> `read/write to tcp connection`
    - socket://\<path> `read/write to socket file`
    - ws://\<path> `read/write to http websocket`
    - wss://\<path> `read/write to htts websocket`
    - fifo://\<path> `read/write to fifo file on filesystem`
    - scp://\<path> `copy from/to sshv2 server`
    - kafka://\<host>/\<topic> `receive/send message to kafka`
  - commands
    - aes-256-cbc -key-in \<filetype> -derived-key-in \<filetype> -salt-pos 0 -salt-length 32 -salt-in \<filetype> -iv-in \<filetype> -iv-pos 32 -iv-length 32
    - nacl
    - ec
    - hmac
  - codecs
    - pem
    - delete-chars:`characters`
    - base58
    - decimal
    - uint
    - octal
