# cryptocli
A modern tool to replace dd and openssl cli

## Motivation
I use decoding/encoding tools, dd and openssl all the time. It was getting a little bit annoying to have to use shell tricks to get what I wanted.

Pull requests are of course welcome.

## Futur

  - cleanup the code
  - commands
    - aes
    - nacl
    - ec
  - codecs
    - base58
    - decimal
    - octal

## Usage

`cryptocli <command> [<options>] [<arguments>]`

```
Commands:
  dd:  Copy input to output like the dd tool.
  dgst:  Hash the content of stdin

Options:
  -chomp
        Get rid of the last \n when not in pipe
  -decoder string
        Specify the decoder codec of input (default "binary")
  -encoder string
        Specify the encoder codec of output (default "binary")
  -from-byte-in string
        Skip the first x bytes of stdin. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10
  -from-byte-out string
        Skip the first x bytes of stdout. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base 10
  -to-byte-in string
        Stop at byte x of stdin.  Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base10. If you add a '+' at the begining, the value will be added to -from-byte-in (default "+0")
  -to-byte-out string
        Stop at byte x of stdout. Use 0X/0x for base 16, 0b/0B for base 2, 0 for base8 otherwise base10. If you add a '+' at the begining, the value will be added to -from-byte-out (default "+0")

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

`echo -n toto | cryptocli dd -encoder binary_string`

Gzip stdin then base64 it

`echo -n toto | cryptocli dd -encoder gzip  | cryptocli dd -encoder base64`

Get rid of the first 2 bytes

`echo -n toto | cryptocli dd -from-byte-in 2`
