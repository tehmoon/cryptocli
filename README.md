# cryptocli
A modern tool to replace dd and openssl cli

## Motivation
I use decoding/encoding tools, dd and openssl all the time. It was getting a little bit annoying to have to use shell tricks to get what I wanted.

Pull requests are of course welcome.

## Futur

  - download x509 certificates from https
  - http/https proxy
  - http/https servers
  - cleanup the code
  - file types:
    - tls://\<addr>
    - tcp://\<addr> `read/write to tcp connection`
    - socket://\<path> `read/write to socket file`
    - fifo://\<path> `read/write to fifo file on filesystem`
  - commands
    - nacl
    - ec
    - hmac
  - codecs
    - aes-256-cbc[:`env password`]
    - aes-512-cbc[:`env password`]
    - base58
    - decimal
    - uint
    - octal

## Usage

`cryptocli <command> [<options>] [<arguments>]`

```
Usage: ./cryptocli [<Options>] 

Options:
  -chomp
    	Get rid of the last \n when not in pipe
  -decoders string
    	Set a list of codecs separated by ',' to decode input that will be process in the order given (default "binary")
  -encoders string
    	Set a list of codecs separated by ',' to encode output that will be process in the order given (default "binary")
  -from-byte-in string
    	Skip the first x bytes of stdin. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10
  -from-byte-out string
    	Skip the first x bytes of stdout. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10
  -in string
    	Input <fileType> method
  -out string
    	Output <fileType> method
  -tee string
    	Copy the output of -output to <fileType>
  -to-byte-in string
    	Stop at byte x of stdin.  Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10. If you add a '+' at the begining, the value will be added to -from-byte-in
  -to-byte-out string
    	Stop at byte x of stdout. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10. If you add a '+' at the begining, the value will be added to -from-byte-out

Codecs:
  hex
	hex encode output and hex decode input
  binary
	Do nothing in input and nothing in output
  binary_string
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
```

## Examples

Get the last 32 byte of a sha512 hash function from a hex string to base64 without last \n

`echo -n 'DEADBEEF' | cryptocli dgst -decoder hex -encoder base64 -from-byte-out 32 -to-byte-out +32 -chomp sha512`

Transform stdin to binary string

`echo -n toto | cryptocli dd -encoders binary_string`

Gzip stdin then base64 it

`echo -n toto | cryptocli dd -encoders gzip,base64`

Get rid of the first 2 bytes

`echo -n toto | cryptocli dd -from-byte-in 2`

Output the base64 hash of stdin to file

`echo -n toto | cryptocli dgst -encoders base64 -out file://./toto.txt sha512`

Decode base64 from file to stdout in hex

`cryptocli dd -decoders base64 -encoders hex -in ./toto.txt`

Gzip input, write it to file and write its sha512 checksum in hex format to another file

`echo toto | cryptocli dd -encoders gzip -tee pipe:"cryptocli dgst -encoders hex -out ./checksum.txt" -out ./file.gz`

SHA512 an https web page then POST the result to http server:

`cryptocli dgst -in https://www.google.com -encoders hex sha512 -out http://localhost:12345/`
