# humble bundle downloader
Download your favourite bundles from [humblebundle](https://www.humblebundle.com/) via the command line.

<a href="http://golang.org"><img alt="Go package" src="https://golang.org/doc/gopher/appenginegophercolor.jpg" width="20%" /></a>
<a href="http://trello.com"><img src="http://www.gamasutra.com/db_area/images/news/2017/Feb/291206/humblebundle128.jpg"></a>

[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg?style=for-the-badge)](https://godoc.org/github.com/diogogmt/humblebundle-downloader)


- [humble bundle downloader](#humble-bundle-downloader)
  - [Installation](#installation)
      - [Binary](#binary)
      - [Go](#go)
      - [Homebrew](#homebrew)
  - [Usage](#usage)
    - [Examples](#examples)
  - [Contributing](#contributing)
      - [Makefile](#makefile)
  - [TODO](#todo)

## Installation

#### Binary

For installation instructions from binaries please visit the [Releases Page](https://diogogmt/humblebundle-downloader/releases).

#### Go

```bash
$ go get diogogmt/humblebundle-downloader/cmd
```

#### Homebrew

TODO

## Usage

```bash
$ hbd -h
USAGE
  hbd [flags] <subcommand>

SUBCOMMANDS
  download  Download assets from bundle

FLAGS
  -jwt ...  humblebundle dashboard JWT cookie
  -v false  log verbose output
```

```bash
$ hbd download
USAGE
  hbd download

FLAGS
  -dest ...   directory to download all bundle assets
  -key ...    purchase key
  -types all  which file types to download, eg; pdf, epub, mobi, etc...
```

### Examples

```bash
# download all pdf assets from bundle xxx into the bundle-pdf directory
$ hbd download -key xxx -types pdf -dest ./bundle-pdf

# download all assets using JWT _simpleauth_sess cookie for bundles linked to an account
$ hbd download -jwt=eyJ1... -key xxx -types pdf -dest ./bundle-pdf
```

## Contributing

#### Makefile

```bash
$ make help
Usage: 

  build         builds hbd binary
  imports       runs goimports
  lint          runs golint
  test          runs go test
  vet           runs go vet
  staticcheck   runs staticcheck
  vendor        updates vendored dependencies
  help          prints this help message
```

## TODO

* print all bundle assets before downloading
* add gh
* interactive GUI to select which assets to download