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
   --loglevel value, -l value  Set the loglevel (default: "info")
   --help, -h                  show help
   --version, -v               print the version

COPYRIGHT:
   (c) 2017 Yahoo Inc.

---
NAME:
   meta get - Get a metadata with key

USAGE:
   meta get [command options] [arguments...]

OPTIONS:
   --external value, --last-successful value, -e value  MetaFile pipeline meta (default: "meta")
   --json-value, -j                                     Treat value as json
   --sd-token value, -t value                           Set the SD_TOKEN to use in SD API calls [$SD_TOKEN]
   --sd-api-url value, -u value                         Set the SD_API_URL to use in SD API calls (default: "https://api.screwdriver.cd/v4/") [$SD_API_URL]
   --sd-pipeline-id value, -p value                     Set the SD_PIPELINE_ID for job description (default: 0) [$SD_PIPELINE_ID]

---
NAME:
   meta set - Set a metadata with key and value

USAGE:
   meta set [command options] [arguments...]

OPTIONS:
   --json-value, -j  Treat value as json


$ ./meta set aaa bbb
$ ./meta get aaa
bbb
$ ./meta set foo[2].bar[1] baz
[null,null,{"bar":[null,"baz"]}]
$ ./meta set foo '{"bar": "baz", "buz": 123}' --json-value
$ ./meta get foo --json-value
{"bar":"baz","buz":123}
$ ./meta get foo.bar
baz
$ ./meta get foo.bar --json-value
"baz"
$ ./meta get meta --external sd@123:other-job
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
