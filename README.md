# Metadata CLI
[![Build Status][build-image]][build-url]
[![Latest Release][version-image]][version-url]
[![Go Report Card][goreport-image]][goreport-url]

> CLI for reading/writing project metadata

## Usage

```bash
$ go get github.com/screwdriver-cd/meta-cli
$ cd $GOPATH/src/github.com/screwdriver-cd/meta-cli
$ go build -a -o meta
$ ./meta --help
NAME:
   meta-cli - get or set metadata for Screwdriver build

USAGE:
   meta command arguments [options]

VERSION:
   0.0.0

COMMANDS:
     get      Get a metadata with key
     set      Set a metadata with key and value
     help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --meta-space value          Location of meta temporarily (default: "/sd/meta")
   --external value, -e value  External pipeline meta (default: "meta")
   --json-value, -j            Treat value as json
   --help, -h                  show help
   --version, -v               print the version

COPYRIGHT:
   (c) 2017-unknown Yahoo Inc.
$ ./meta set aaa bbb
$ ./meta get aaa
bbb
$ ./meta set foo[2].bar[1] baz
[null,null,{"bar":[null,"baz"]}]
$ ./meta set foo '{"bar": "baz", "buz": 123}' --json-value
$ ./meta-cli get foo --json-value
{"bar":"baz","buz":123}
$ ./meta get foo.bar
baz
$ ./meta get foo.bar --json-value
"baz"
```

## Testing

```bash
$ go get github.com/screwdriver-cd/meta-cli
$ go test -cover github.com/screwdriver-cd/meta-cli/...
```

## License

Code licensed under the BSD 3-Clause license. See LICENSE file for terms.

[version-image]: https://img.shields.io/github/tag/screwdriver-cd/meta-cli.svg
[version-url]: https://github.com/screwdriver-cd/meta-cli/releases
[build-image]: https://cd.screwdriver.cd/pipelines/67/badge
[build-url]: https://cd.screwdriver.cd/pipelines/67
[goreport-image]: https://goreportcard.com/badge/github.com/Screwdriver-cd/meta-cli
[goreport-url]: https://goreportcard.com/report/github.com/Screwdriver-cd/meta-cli
