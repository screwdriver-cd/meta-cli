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
   --external value, -e value        MetaFile pipeline meta (default: "meta")
   --skip-fetch, -F                  Used with --external to skip fetching from lastSuccessfulMeta when not triggered by external job
   --json-value, -j                  Treat value as json. When false, set values are treated as string; get is value-dependent and strings are not json-escaped
   --sd-token value, -t value        Set the SD_TOKEN to use in SD API calls [$SD_TOKEN]
   --sd-api-url value, -u value      Set the SD_API_URL to use in SD API calls (default: "https://api.screwdriver.cd/v4/") [$SD_API_URL]
   --sd-pipeline-id value, -p value  Set the SD_PIPELINE_ID of the job for fetching last successful meta (default: 0) [$SD_PIPELINE_ID]
   --skip-store                      Used with --external to skip storing external metadata in the local meta
   --cache-local                     Used with external, this flag saves a copy of the key/value pair in the local meta

---
NAME:
   meta set - Set a metadata with key and value

USAGE:
   meta set [command options] [arguments...]

OPTIONS:
   --json-value, -j  Treat value as json. When false, set values are treated as string; get is value-dependent and strings are not json-escaped

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
$ # For scheduled jobs, e.g. that trigger things normally triggered by component:
  if [[ "$(./meta get -j meta)" == null ]]; then
      ./meta set -j meta "$(meta get meta -j --external component)"
  fi
---
NAME:
   meta lua - Run a lua script

USAGE:
   meta lua [command options] [arguments...]

OPTIONS:
   --evaluate value, -E value        lua text to evaluate; when not set, the first argument is treated as a filename
   --external value, -e value        MetaFile pipeline meta (default: "meta")
   --skip-fetch, -F                  Used with --external to skip fetching from lastSuccessfulMeta when not triggered by external job
   --json-value, -j                  Treat value as json. When false, set values are treated as string; get is value-dependent and strings are not json-escaped
   --sd-token value, -t value        Set the SD_TOKEN to use in SD API calls [$SD_TOKEN]
   --sd-api-url value, -u value      Set the SD_API_URL to use in SD API calls (default: "https://api.screwdriver.cd/v4/") [$SD_API_URL]
   --sd-pipeline-id value, -p value  Set the SD_PIPELINE_ID of the job for fetching last successful meta (default: 0) [$SD_PIPELINE_ID]
   --skip-store                      Used with --external to skip storing external metadata in the local meta
   --cache-local                     Used with external, this flag saves a copy of the key/value pair in the local meta

$ # Atomically increment a key that may or may not exist
$ meta lua -E 'meta.set("num", (meta.get("num") or 0) + 1)'

$ # Append image metadata and output the index
$ meta lua -E 'images = meta.get("images") or {}; print(#images); table.insert(images, {org="foo", repo="bar", tag="7.8.2-202101041200"}); meta.set("images", images)'
0
$ meta lua -E 'images = meta.get("images") or {}; print(#images); table.insert(images, {org="foo", repo="baz", tag="7.8.2-202101041200"}); meta.set("images", images)'
1
```

## Brew installation (experimental)
```bash
brew tap screwdriver-cd/meta-cli git@github.com:screwdriver-cd/meta-cli.git
brew install screwdriver-cd/meta-cli/meta
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
