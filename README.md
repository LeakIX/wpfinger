# wpfinger

[![GitHub Release](https://img.shields.io/github/v/release/LeakIX/wpfinger)](https://github.com/LeakIX/wpfinger/releases)
[![Follow on Twitter](https://img.shields.io/twitter/follow/leak_ix.svg?logo=twitter)](https://twitter.com/leak_ix)

wpfinger is a red-team WordPress scanning tool.

![screenshot](https://media.leakix.net/wpfinger.gif)

## Features

- Core version detection
- Plugin scanning through fingerprinting
- Vulnerability output, using database from [Wordfence](https://www.wordfence.com/intelligence-documentation/v2-accessing-and-consuming-the-vulnerability-data-feed/) 

## Usage

### Update database

```
wpfinger update
```

Will update the database with the latest vulnerabilities and plugin fingerprint.

### Scan

```
wpfinger scan -u https://example.com
```

| Flag  | Description                                           |
|-------|-------------------------------------------------------|
| --all | Will scan for all plugins, default is vulnerable only |
| --url | Target WordPress URL                                  |

## Installation Instructions

### From Binary

The installation is easy. You can download the pre-built binaries for your platform from the [Releases](https://github.com/LeakIX/wpfinger/releases/) page.

```sh
▶ chmod +x wpfinger
▶ mv wpfinger /usr/local/bin/wpfinger
```

### From Source

```sh
▶ GO111MODULE=on go get -u -v github.com/LeakIX/wpfinger/cmd/l9explore
▶ ${GOPATH}/bin/wpfinger -h
```


## Acknowledgements

Vulnerability database is courtesy of [Wordfence](https://www.wordfence.com/intelligence-documentation/v2-accessing-and-consuming-the-vulnerability-data-feed/).
