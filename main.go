package main

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
)

var (
	flags   = flag.NewFlagSet("humblebundle", flag.ExitOnError)
	baseURL = "https://www.humblebundle.com/api/v1"
	gameKey = flags.String("key", "", "key: Key listed in the URL params in the downloads page")
	out     = flags.String("out", "", "out: /path/to/save/books")
)

type HumbleBundleOrder struct {
	AmountSpent float64
	Product     struct {
		Category    string
		MachineName string
		HumanName   string `json:"human_name"`
	}
	GameKey  string `json:"gamekey"`
	UID      string `json:"uid"`
	Created  string `json:"created"`
	Products []struct {
		MachineName string `json:"machine_name"`
		HumanName   string `json:"human_name"`
		URL         string `json:"url"`
		Downloads   []struct {
			MachineName   string `json:"machine_name"`
			HumanName     string `json:"human_name"`
			Platform      string `json:"platform"`
			DownloadTypes []struct {
				SHA1 string `json:"sha1"`
				Name string `json:"name"`
				URL  struct {
					Web        string `json:"web"`
					BitTorrent string `json:"bittorrent"`
				} `json:"url"`
				HumanSize string `json:"human_size"`
				FileSize  int64  `json:"file_size"`
				MD5       string `json:"md5"`
			} `json:"download_struct"`
		} `json:"downloads"`
	} `json:"subproducts"`
}

type logger struct {
}

func (writer logger) Write(bytes []byte) (int, error) {
	return fmt.Print(string(bytes))
}

func main() {
	flags.Parse(os.Args[1:])
	if *gameKey == "" {
		log.Fatal("Missing key")
	}

	log.SetFlags(0)
	log.SetOutput(new(logger))

	// Build order endpoint URL
	u, err := url.Parse(baseURL)
	if err != nil {
		log.Fatal(err)
	}
	u.Path = path.Join(u.Path, "order")
	u.Path = path.Join(u.Path, *gameKey)
	// Fetch order information
	resp, err := http.Get(u.String())
	if err != nil {
		log.Fatalf("error downloading order information %s: %v", u, err)
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	order := &HumbleBundleOrder{}
	err = json.Unmarshal(buf, order)
	if err != nil {
		log.Fatalf("error unmarshaling order: %v", err)
	}
	if *out == "" {
		log.Printf("Human Name: %s", order.Product.HumanName)
		if order.Product.HumanName == "" {
			pwd, err := os.Getwd()
			if err != nil {
				log.Fatal(err)
			}
			*out = fmt.Sprintf("%s/books", pwd)
		} else {
			order.Product.HumanName = strings.Replace(order.Product.HumanName, "/", "_", -1)
			*out = order.Product.HumanName
		}
	}
	_ = os.MkdirAll(*out, 0777)
	log.Printf("Saving files into %s", *out)

	// Download all files from order
	var group sync.WaitGroup
	// 1 - Iterate through all products
	for i := 0; i < len(order.Products); i++ {
		prod := order.Products[i]
		// 2 - Iterate through the product downloads, currently only returns ebook platform download
		for j := 0; j < len(prod.Downloads); j++ {
			download := prod.Downloads[j]
			// 3 - Iterate through download types (PDF,EPUB,MOBI)
			for x := 0; x < len(download.DownloadTypes); x++ {
				downloadType := download.DownloadTypes[x]
				group.Add(1)
				go func(filename, downloadURL string) {
					filename = strings.Replace(filename, "/", "_", -1)
					filename = strings.Replace(filename, ".supplement", "_supplement.zip", 1)
					filename = strings.Replace(filename, ".download", "_video.zip", 1)
					defer group.Done()
					resp, err := http.Get(downloadURL)
					if err != nil {
						log.Printf("error downloading file %s", downloadURL)
						return
					}
					defer resp.Body.Close()

					bookLastmod := resp.Header.Get("Last-Modified")
					bookLastmodTime, err := http.ParseTime(bookLastmod)
					if err != nil {
						log.Printf("error reading Last-Modified header data: %v", err)
						return
					}

					// log.Printf("Last-Modified Header: ~%s~", bookLastmod)
					// log.Printf("Download status: %d - %s", resp.StatusCode, downloadURL)
					if resp.StatusCode < 200 || resp.StatusCode > 299 {
						log.Printf("error status code %d", resp.StatusCode)
						return
					}

					bookFile, err := os.Create(fmt.Sprintf("%s/%s", *out, filename))
					if err != nil {
						log.Printf("error creating book file (%s/%s): %v", *out, filename, err)
						return
					}
					defer bookFile.Close()

					_, err = io.Copy(bookFile, resp.Body)
					if err != nil {
						log.Printf("error copying response body to file (%s/%s): %v", *out, filename, err)
						return
					}
					log.Printf("Finished saving file %s/%s", *out, filename)
					os.Chtimes(fmt.Sprintf("%s/%s", *out, filename), bookLastmodTime, bookLastmodTime)

					// log.Printf("TZ=UTC touch -d \"%s\" \"%s/%s\"", strings.Replace(fmt.Sprintf("%s", bookLastmodTime), " UTC", "", 1), *out, filename)
					// log.Printf("\t%-9d \"%s\"", resp.ContentLength, downloadURL)

					if downloadType.SHA1 != "" {
						f, err := os.Open(fmt.Sprintf("%s/%s", *out, filename))
						if err != nil {
							log.Printf("error reading file: %v for: %s/%s", err, *out, filename)
							return
						}
						defer f.Close()

						hash := sha1.New()
						if _, err := io.Copy(hash, f); err != nil {
							log.Printf("error calculating sha1sum: %v for: %s/%s", err, *out, filename)
						}
						bs := hash.Sum(nil)
						if downloadType.SHA1 != fmt.Sprintf("%x", bs) {
							log.Printf("SHA1 checksum failed for %s -- expected %s but got %x", filename, downloadType.SHA1, bs)
						} // else {
						// 	log.Printf("SHA1 checksum is good for %s/%s -- %x", *out, filename, bs)
						// }
					}
					if downloadType.MD5 != "" {
						f, err := os.Open(fmt.Sprintf("%s/%s", *out, filename))
						if err != nil {
							log.Printf("error reading file: %v for: %s/%s", err, *out, filename)
							return
						}
						defer f.Close()

						hash := md5.New()
						if _, err := io.Copy(hash, f); err != nil {
							log.Printf("error calculating md5sum: %v for: %s/%s", err, *out, filename)
						}
						bs := hash.Sum(nil)
						if downloadType.MD5 != fmt.Sprintf("%x", bs) {
							log.Printf("MD5 checksum failed for %s -- expected %s but got %x", filename, downloadType.MD5, bs)
						} // else {
						// 	log.Printf("MD5 checksum is good for %s/%s -- %x", *out, filename, bs)
						// }
					}

				}(fmt.Sprintf("%s.%s", prod.HumanName, strings.ToLower(strings.TrimPrefix(downloadType.Name, "."))), downloadType.URL.Web)
			}
		}
	}
	group.Wait()
}
