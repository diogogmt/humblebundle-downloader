Humble Bundle File Downloader
-----------------------------

<a href="http://trello.com"><img src="http://www.gamasutra.com/db_area/images/news/2017/Feb/291206/humblebundle128.jpg"></a>
<a href="http://golang.org"><img alt="Go package" src="https://golang.org/doc/gopher/appenginegophercolor.jpg" width="20%" /></a>

## Why?

After hearding about the latest Humble Bundle [cyber security deal](https://www.humblebundle.com/books/cybersecurity-wiley) I wanted an easy way to download all the book files as well as extra content such as DVD examples.

When you access the Humble Bundle download page `https://www.humblebundle.com/downloads?key=yourDownloadKey` the pages makes a GET request to `https://www.humblebundle.com/api/v1/order/yourDownloadKey` to fetch all the available books as part of your order. The resonse that comes back from the order API is as follow:
```json
{
  "amount_spent": 0.0,
  "product": {
    "category": "bundle",
    "machine_name": "wiley_bookbundle",
    "post_purchase_text": "",
    "supports_canonical": false,
    "human_name": "Humble Book Bundle: Cybersecurity presented by Wiley",
    "automated_empty_tpkds": {},
    "partial_gift_enabled": true
  },
  "gamekey": "yourDownloadKey",
  "uid": "1234547R8DXTQ",
  "created": "2017-07-23T17:42:57.642320",
  "subproducts": [{
      "machine_name": "social_engineering_the_art_of_human_hacking",
      "url": "http://www.wiley.com/remtitle.cgi?0470639539",
      "downloads": [{
        "machine_name": "social_engineering_the_art_of_human_hacking_ebook",
        "platform": "ebook",
        "download_struct": [{
            "sha1": "361ea9355b7b0f26aa077fc57e9bed1d92a6131b",
            "name": "EPUB",
            "url": {
              "web": "https://dl.humble.com/social_engineering_the_art_of_human_hacking.epub?gamekey=yourDownloadKey&ttl=1500923594",
              "bittorrent": "https://dl.humble.com/torrents/social_engineering_the_art_of_human_hacking.epub.torrent?gamekey=yourDownloadKey&ttl=1500923594"
            },
            "human_size": "6 MB",
            "file_size": 6331444,
            "small": 1,
            "md5": "9a5cb066b06b9d14cacaaa4311ff1ecc"
          },
          {
            "sha1": "07982aaf7c3eb0f45a3642a1dcde3d37d583b345",
            "name": "PDF",
            "url": {
              "web": "https://dl.humble.com/social_engineering_the_art_of_human_hacking.pdf?gamekey=yourDownloadKey&ttl=1500923594",
              "bittorrent": "https://dl.humble.com/torrents/social_engineering_the_art_of_human_hacking.pdf.torrent?gamekey=yourDownloadKey&ttl=1500923594"
            },
            "human_size": "6 MB",
            "file_size": 6298426,
            "small": 1,
            "md5": "9b1cebe9825b405a57b04add0d743842"
          },
          {
            "sha1": "c23a01c249b5374e287e972d34ff275f06ee06a1",
            "name": "MOBI",
            "url": {
              "web": "https://dl.humble.com/social_engineering_the_art_of_human_hacking.prc?gamekey=yourDownloadKey&ttl=1500923594",
              "bittorrent": "https://dl.humble.com/torrents/social_engineering_the_art_of_human_hacking.prc.torrent?gamekey=yourDownloadKey&ttl=1500923594"
            },
            "human_size": "9.7 MB",
            "file_size": 10187642,
            "small": 1,
            "md5": "63496aab0b82c2d7b2df80fb4aa4462f"
          }
        ],
        "options_dict": {},
        "download_identifier": null,
        "android_app_only": false,
        "download_version_number": null
      }],
      "library_family_name": null,
      "payee": {
        "human_name": "Wiley",
        "machine_name": "wiley"
      },
      "human_name": "Social Engineering: The Art of Human Hacking",
      "custom_download_page_box_html": null,
      "icon": "https://humblebundle-a.akamaihd.net/misc/files/hashed/8f3a65315ed5c726ff581916f436d258e51b32d7.png"
    }
  ],
  "currency": "USD",
  "is_giftee": false,
  "has_wallet": false,
  "claimed": false,
  "total": 0.0,
  "wallet_credit": null,
  "path_ids": [
    "12345449980812964"
  ]
}
```
With that data structure the CLI iterates through all the book purchases and download all the listed files.

Added option to exclude download types using .ignore file

New shell script to determine download types to be added, if desired to the .ignore file

## How to compile

### Setup go environment

Example has golang installed at `/opt/go`

```shell
$ cat ~/.go
# Point to the local installation of golang.
export GOROOT=/opt/go

# Point to the location beneath which source and binaries are installed.
export GOPATH="${HOME}/go"

# Ensure that the binary-release is on your PATH.
export PATH="${PATH}:${GOROOT}/bin"

# Ensure that compiled binaries are also on your PATH.
export PATH="${PATH}:${GOPATH}/bin"
```

```shell
cd ~go
go build -o bin/humblebundle-downloader src/github.com/affinityv/humblebundle-downloader/main.go
```

## Examples

Download all books from your Humble Bundle Order

```shell
humblebundle-downloader -key=yourDownloadKey
```
