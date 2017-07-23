package main

import (
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
		HumanName   string
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

func main() {
	flags.Parse(os.Args[1:])
	if *gameKey == "" {
		log.Fatal("Missing key")
	}
	if *out == "" {
		pwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}
		*out = fmt.Sprintf("%s/books", pwd)
	}
	_ = os.MkdirAll(*out, 0777)
	log.Printf("Saving files into %s", *out)
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
		log.Fatalf("[ERROR] error downloading order information %s: %v", u, err)
	}
	defer resp.Body.Close()
	buf, err := ioutil.ReadAll(resp.Body)
	order := &HumbleBundleOrder{}
	err = json.Unmarshal(buf, order)
	if err != nil {
		log.Fatalf("[ERROR] error unmarshaling order", err)
	}

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
				go func(name, fileType, downloadURL string) {
					defer group.Done()
					resp, err := http.Get(downloadURL)
					if err != nil {
						log.Printf("[ERROR] error downloading file %s", downloadURL)
						return
					}
					defer resp.Body.Close()
					log.Printf("Download status: %d - %s", resp.StatusCode, downloadURL)
					if resp.StatusCode < 200 || resp.StatusCode > 299 {
						log.Printf("[ERROR] error status code %d", resp.StatusCode)
						return
					}

					bookFile, err := os.Create(fmt.Sprintf("%s/%s.%s", *out, name, fileType))
					if err != nil {
						log.Printf("[ERROR] error creating book file %s", err)
						return
					}
					defer bookFile.Close()

					log.Printf("Saving file %s", name)
					_, err = io.Copy(bookFile, resp.Body)
					if err != nil {
						log.Printf("[ERROR] error reading response body %s", err)
						return
					}
					log.Printf("Finished saving file %s.%s", name, fileType)
				}(prod.HumanName, strings.ToLower(strings.TrimPrefix(downloadType.Name, ".")), downloadType.URL.Web)
			}
		}
	}
	group.Wait()
}
