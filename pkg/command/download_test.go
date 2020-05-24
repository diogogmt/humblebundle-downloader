package command

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"diogogmt.com/hbd/pkg/hbclient"
	"github.com/peterbourgon/ff/v2/ffcli"
	"github.com/pkg/errors"
)

func TestDownloadAsset(t *testing.T) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(fmt.Errorf("net.Listen: %s", err))
	}

	addrParts := strings.Split(listener.Addr().String(), ":")
	apiURL := fmt.Sprintf("http://localhost:%s", addrParts[len(addrParts)-1])
	hbClient := hbclient.NewClient(hbclient.WithAPIURL(apiURL))

	srv := http.Server{}
	go func() {
		if err := srv.Serve(listener); err != nil {
			panic(fmt.Errorf("srv.ListenAndServe: %s", err))
		}
	}()

	var (
		validOrder       hbclient.Order
		invalidMD5Order  hbclient.Order
		invalidSHA1Order hbclient.Order
	)

	// valid order
	{
		uid := "XTWV64DX7R8TQ"

		md5Hash := md5.New()
		md5Hash.Write([]byte(uid))
		md5Checksum := fmt.Sprintf("%x", md5Hash.Sum(nil))

		sha1Hash := sha1.New()
		sha1Hash.Write([]byte(uid))
		sha1Checksum := fmt.Sprintf("%x", sha1Hash.Sum(nil))

		validOrder = hbclient.Order{
			UID:     uid,
			GameKey: "game-key",
			Product: &hbclient.Product{
				HumanName: "Humble Book Bundle: Cybersecurity presented by Wiley",
			},
			Products: []*hbclient.Product{
				&hbclient.Product{
					HumanName: "Security/Social Engineering: The Art of Human Hacking",
					Downloads: []*hbclient.Download{
						&hbclient.Download{
							Platform: "ebook",
							Types: []*hbclient.DownloadType{
								&hbclient.DownloadType{
									Name:      "PDF",
									HumanName: "Security/Social Engineering: The Art of Human Hacking",
									MD5:       md5Checksum,
									SHA1:      sha1Checksum,
									URL: hbclient.DownloadTypeURL{
										Web: fmt.Sprintf("%s/%s", apiURL, uid),
									},
								},
								&hbclient.DownloadType{
									Name:      "EPUB",
									HumanName: "Security/Social Engineering: The Art of Human Hacking",
									MD5:       md5Checksum,
									SHA1:      sha1Checksum,
									URL: hbclient.DownloadTypeURL{
										Web: fmt.Sprintf("%s/%s", apiURL, uid),
									},
								},
								&hbclient.DownloadType{
									Name:      "PRC",
									HumanName: "Security/Social Engineering: The Art of Human Hacking",
									MD5:       md5Checksum,
									SHA1:      sha1Checksum,
									URL: hbclient.DownloadTypeURL{
										Web: fmt.Sprintf("%s/%s", apiURL, uid),
									},
								},
							},
						},
					},
				},
			},
		}
	}

	// invalid md5 checksum
	{
		uid := "58673926E155X"
		invalidMD5Order = hbclient.Order{
			UID:     uid,
			GameKey: "game-key",
			Product: &hbclient.Product{
				HumanName: "Humble Book Bundle: Cybersecurity presented by Wiley",
			},
			Products: []*hbclient.Product{
				&hbclient.Product{
					HumanName: "Social Engineering: The Art of Human Hacking",
					Downloads: []*hbclient.Download{
						&hbclient.Download{
							Platform: "ebook",
							Types: []*hbclient.DownloadType{
								&hbclient.DownloadType{
									Name:      "PDF",
									HumanName: "Social Engineering: The Art of Human Hacking",
									MD5:       "INVALID",
									URL: hbclient.DownloadTypeURL{
										Web: fmt.Sprintf("%s/%s", apiURL, uid),
									},
								},
							},
						},
					},
				},
			},
		}
	}

	// invalid sha1 checksum
	{
		uid := "BC1BC812BD49Z"
		invalidSHA1Order = hbclient.Order{
			UID:     uid,
			GameKey: "game-key",
			Product: &hbclient.Product{
				HumanName: "Humble Book Bundle: Cybersecurity presented by Wiley",
			},
			Products: []*hbclient.Product{
				&hbclient.Product{
					HumanName: "Social Engineering: The Art of Human Hacking",
					Downloads: []*hbclient.Download{
						&hbclient.Download{
							Platform: "ebook",
							Types: []*hbclient.DownloadType{
								&hbclient.DownloadType{
									Name:      "PDF",
									HumanName: "Social Engineering: The Art of Human Hacking",
									SHA1:      "INVALID",
									URL: hbclient.DownloadTypeURL{
										Web: fmt.Sprintf("%s/%s", apiURL, uid),
									},
								},
							},
						},
					},
				},
			},
		}
	}

	setupHandlers(t, validOrder)
	setupHandlers(t, invalidMD5Order)
	setupHandlers(t, invalidSHA1Order)

	dd := []struct {
		name      string
		order     hbclient.Order
		types     []string
		expectErr bool
	}{
		{
			name:  "valid-all",
			order: validOrder,
			types: []string{"all"},
		},
		{
			name:      "invalid-md5",
			order:     invalidMD5Order,
			types:     []string{"all"},
			expectErr: true,
		},
		{
			name:      "invalid-sha1",
			order:     invalidSHA1Order,
			types:     []string{"all"},
			expectErr: true,
		},
		{
			name:  "valid-pdf",
			order: validOrder,
			types: []string{"pdf"},
		},
		{
			name:  "valid-pdf-epub",
			order: validOrder,
			types: []string{"pdf", "epub"},
		},
	}
	for _, d := range dd {
		order := d.order
		tempDir, err := ioutil.TempDir("/tmp", "hbd.")
		if err != nil {
			t.Fatalf("ioutil.TempDir: %s", err)
		}

		rootCmd := NewRootCmd()
		rootCmd.Conf.HBClient = hbClient
		downloadCmd := NewDownloadCmd(rootCmd.Conf)
		rootCmd.Subcommands = []*ffcli.Command{
			downloadCmd.Command,
		}
		if err := rootCmd.Parse([]string{"download", "-key", order.UID, "-dest", tempDir, "-types", strings.Join(d.types, ",")}); err != nil {
			t.Fatalf("%s: rootCmd.Parse: %v", d.name, err)
		}

		err = downloadCmd.Exec(context.Background(), []string{})
		if !d.expectErr && err != nil {
			t.Errorf("%s: downloadCmd.Exec: %v", d.name, err)
		} else if d.expectErr && err == nil {
			t.Errorf("%s: downloadCmd.Exec: expected error but got nil", d.name)
		}

		if d.expectErr {
			continue
		}

		typesMap := map[string]struct{}{}
		for _, ty := range d.types {
			typesMap[ty] = struct{}{}
		}

		// Check if the downloaded files from the order match the original content
		for i := 0; i < len(order.Products); i++ {
			prod := order.Products[i]
			for j := 0; j < len(prod.Downloads); j++ {
				download := prod.Downloads[j]
				for x := 0; x < len(download.Types); x++ {
					asset := download.Types[x]
					if _, ok := typesMap[strings.ToLower(asset.Name)]; !ok {
						continue
					}
					filename := fmt.Sprintf("%s.%s", asset.HumanName, strings.ToLower(strings.TrimPrefix(asset.Name, ".")))
					filename = strings.ReplaceAll(filename, "/", "_")
					by, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", tempDir, filename))
					if err != nil {
						t.Errorf("%s: ioutil.ReadFile: %v", d.name, err)
					}
					if order.UID != string(by) {
						t.Errorf("%s: expected file content to be %q but got %q", d.name, order.UID, string(by))
					}
				}
			}
		}
	}

}

func setupHandlers(t *testing.T, order hbclient.Order) {
	t.Helper()

	http.HandleFunc(fmt.Sprintf("/order/%s", order.UID), func(w http.ResponseWriter, r *http.Request) {
		by, err := json.Marshal(&order)
		if err != nil {
			http.Error(w, errors.Wrap(err, "failed to marshal order").Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Add("content-type", "application/json")
		w.Write(by)
	})

	http.HandleFunc(fmt.Sprintf("/%s", order.UID), func(w http.ResponseWriter, r *http.Request) {
		by := []byte(order.UID)
		w.Header().Add("Content-Type", "application/octet-stream")
		w.Header().Add("Last-Modified", time.Now().Format(time.RFC850))
		w.Write(by)
	})

}
