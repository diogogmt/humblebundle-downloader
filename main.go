package main

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"flag"
	"fmt"
	"hash"
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
	types   = flags.String("types", "", "types: show download types")
	downloadTypes []string
	excludeTypes []string
	includeTypes []string
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

func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func check_checksum(filename string, checksum string, checksum_type string) {
	if checksum == "" {
		log.Println("checksum is empty, nothing to check")
		return
	}

    var hash hash.Hash

	switch checksum_type {
	case "md5sum ":
		hash = md5.New()
	case "sha1sum":
	    hash = sha1.New()
	default:
		log.Printf(
			"checksum type is not valid, we have \"%s\", but it needs to be \"%s\" or \"%s\"",
			checksum_type, "md5sum ", "sha1sum")
		return
	}

	f, err := os.Open(filename)
	if err != nil {
		log.Printf("error reading file: %v \tfor: %s", err, filename)
		return
	}
	defer f.Close()

	if _, err := io.Copy(hash, f); err != nil {
		log.Printf("error caluclating %s: %v for: %s", checksum_type, err, filename)
	}
	bs := hash.Sum(nil)
	if checksum != fmt.Sprintf("%x", bs) {
		log.Printf("%s checksum failed  for %120s -- expected %s but got %x", checksum_type, filename, checksum, bs)
	} else {
		log.Printf("%s checksum is good for %120s -- %x", checksum_type, filename, bs)
	}
}

func setup_exclude_fileTypes() {
	// get stat of file (if it exists)
	_, err := os.Stat(".exclude")
	if err != nil {
		fmt.Println("No download types to exclude...\n")
	} else {
		fileBytes, err := ioutil.ReadFile(".exclude")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		excludeTypes = strings.Split(string(fileBytes), "\n")
		// remove empty element (last one added)
		excludeTypes = excludeTypes[:len(excludeTypes)-1]
		// Show slices
		// for x := 0; x < len(excludeTypes); x++ {
		// 	fmt.Printf(".exclude type: ~%s~\n",  excludeTypes[x])
		// }
		if len(excludeTypes) > 0 {
			fmt.Printf("Excluding File Types: ~%s~\n", excludeTypes)
		} else {
			fmt.Println(".exclude file is empty")
		}
	}
}

func setup_include_fileTypes() {
	// get stat of file (if it exists)
	_, err := os.Stat(".include")
	if err != nil {
		fmt.Println("No download types to include...\n")
	} else {
		fileBytes, err := ioutil.ReadFile(".include")
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		includeTypes = strings.Split(string(fileBytes), "\n")
		// remove empty element (last one added)
		includeTypes = includeTypes[:len(includeTypes)-1]
		// Show slices
		// for x := 0; x < len(includeTypes); x++ {
		// 	fmt.Printf(".include type: ~%s~\n",  includeTypes[x])
		// }
		if len(includeTypes) > 0 {
			fmt.Printf("Including File Types: ~%s~\n", includeTypes)
		} else {
			fmt.Println(".include file is empty")
		}
	}
}


func get_download_types(order HumbleBundleOrder) {

	fmt.Println("Building list of download types\n")

	// 1 - Iterate through all products
	for i := 0; i < len(order.Products); i++ {
		prod := order.Products[i]
		// 2 - Iterate through the product downloads
		for j := 0; j < len(prod.Downloads); j++ {
			download := prod.Downloads[j]
			// 3 - Iterate through download types (PDF,EPUB,MOBI, ...)
			for x := 0; x < len(download.DownloadTypes); x++ {
				downloadType := download.DownloadTypes[x]
				if Contains(downloadTypes, downloadType.Name) == false {
					fmt.Printf("   New download type found: ~%s~\n", downloadType.Name)
					downloadTypes = append(downloadTypes,  downloadType.Name)
				}
			}
		}
	}
}

func get_downloads(order HumbleBundleOrder) {
	// Download files from order
	var group sync.WaitGroup
	// 1 - Iterate through all products
	for i := 0; i < len(order.Products); i++ {
		prod := order.Products[i]
		// 2 - Iterate through the product downloads
		for j := 0; j < len(prod.Downloads); j++ {
			download := prod.Downloads[j]
			// 3 - Iterate through download types (PDF,EPUB,MOBI, ...)
			for x := 0; x < len(download.DownloadTypes); x++ {
				downloadType := download.DownloadTypes[x]
				expectedFileSize := download.DownloadTypes[x].FileSize
				if len(excludeTypes) > 0 {
					if Contains(excludeTypes, downloadType.Name) {
						continue
					}
				}
				if len(includeTypes) > 0 {
					if Contains(includeTypes, downloadType.Name) == false {
						continue
					}
				}
				group.Add(1)
				go func(filename, downloadURL string) {

					defer group.Done()

					filename = strings.Replace(filename, "/", "_", -1)
					filename = strings.Replace(filename, ".supplement", "_supplement.zip", 1)
					filename = strings.Replace(filename, ".download", "_video.zip", 1)
					filename = strings.Replace(filename, ".part 1", "_video_part_1.zip", 1)
					filename = strings.Replace(filename, ".part 2", "_video_part_2.zip", 1)

					// get fileInfo structure describing file (if it exists), it may exist from a previous run
					var pathedFilename string
					pathedFilename = fmt.Sprintf("%s/%s", *out, filename)
					fileInfo, err := os.Stat(pathedFilename)
					if err == nil {
						// file exists, check size against size in order
						// fmt.Printf("filename: %-120s\t%12d\t%12d\n", pathedFilename, fileInfo.Size(), expectedFileSize)
						if fileInfo.Size() == expectedFileSize {
							check_checksum(pathedFilename, downloadType.MD5,  "md5sum ")
							check_checksum(pathedFilename, downloadType.SHA1, "sha1sum")
							return
						} else {
							// to be done... perhaps, continue download ...
							fmt.Printf("filename: %-120s\t%12d\t%s\t%s\n", pathedFilename, expectedFileSize, "new download to be started....", downloadURL)
						}
					} else {
						fmt.Printf("filename: %-120s\t%12d\t%s\t%s\n", pathedFilename, expectedFileSize, "download started....", downloadURL)
					}

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

					bookFile, err := os.Create(fmt.Sprintf("%s", pathedFilename))
					if err != nil {
						log.Printf("error creating book file (%s): %v", pathedFilename, err)
						return
					}
					defer bookFile.Close()

					_, err = io.Copy(bookFile, resp.Body)
					if err != nil {
						log.Printf("error copying response body to file (%s): %v", pathedFilename, err)
						return
					}
					log.Printf("Finished saving file %s", pathedFilename)
					os.Chtimes(fmt.Sprintf("%s", pathedFilename), bookLastmodTime, bookLastmodTime)

					// log.Printf("TZ=UTC touch -d \"%s\" \"%s\"", strings.Replace(fmt.Sprintf("%s", bookLastmodTime), " UTC", "", 1), pathedFilename)
					// log.Printf("\t%-9d \"%s\"", resp.ContentLength, downloadURL)

					check_checksum(pathedFilename, downloadType.MD5,  "md5sum ")
					check_checksum(pathedFilename, downloadType.SHA1, "sha1sum")

				}(fmt.Sprintf("%s.%s", prod.HumanName, strings.ToLower(strings.TrimPrefix(downloadType.Name, "."))), downloadType.URL.Web)
			}
		}
	}
	group.Wait()
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

	if strings.ToLower(*types) == "y" {
		get_download_types(*order)
		os.Exit(1)
	}

	// setup or use existing directory for downloads
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

	// setup include and exclude requirements (based on download types)
	// use ".include" and ".exclude" files if they optionally exist
	setup_exclude_fileTypes()
	setup_include_fileTypes()

	// Get downloads
	get_downloads(*order)

}
