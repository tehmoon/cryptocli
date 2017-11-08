# cryptocli
A modern tool to replace dd and openssl cli

## Motivation
I use decoding/encoding tools, dd and openssl all the time. It was getting a little bit annoying to have to use shell tricks to get what I wanted.

Pull requests are of course welcome.

## Futur

  - download x509 certificates from https
  - cleanup the code
  - specify file types for:
    - in
    - out
    - tee
  - file types:
    - tls://<addr>
    - file://<path> #read/write to filesystem
    - http://<addr>/<path> #read/write to http endpoint
    - https://<addr>/<path> #read/write to https endpoint
    - tcp://<addr> #read/write to tcp connection
    - socket://<path> #read/write to socket file
    - fifo://<path> #read/write to fifo file on filesystem
  - commands
    - aes
    - nacl
    - ec
    - hmac
  - codecs
    - base58
    - decimal
    - octal
    - hexdump

## Usage

`cryptocli <command> [<options>] [<arguments>]`

```
Commands:
  dd:  Copy input to output like the dd tool.
  dgst:  Hash the content of stdin

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
  -to-byte-in string
    	Stop at byte x of stdin.  Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10. If you add a '+' at the begining, the value will be added to -from-byte-in
  -to-byte-out string
    	Stop at byte x of stdout. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10. If you add a '+' at the begining, the value will be added to -from-byte-out

Codecs:
  hex
  binary
  binary_string
  base64
  gzip
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
