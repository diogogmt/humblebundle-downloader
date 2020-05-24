package hbclient

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/pkg/errors"
)

var (
	testOrder = Order{
		UID:     "XTWV64DX7R8TQ",
		GameKey: "game-key",
		Product: &Product{
			HumanName: "Humble Book Bundle: Cybersecurity presented by Wiley",
		},
		Products: []*Product{
			&Product{
				HumanName: "Social Engineering: The Art of Human Hacking",
				Downloads: []*Download{
					&Download{
						Platform: "ebook",
						Types: []*DownloadType{
							&DownloadType{
								Name: "EPUB",
								URL: DownloadTypeURL{
									Web: "https://dl.humble.com/social_engineering_the_art_of_human_hacking.epub",
								},
							},
							&DownloadType{
								Name: "PDF",
								URL: DownloadTypeURL{
									Web: "https://dl.humble.com/social_engineering_the_art_of_human_hacking.pdf",
								},
							},
						},
					},
				},
			},
		},
	}

	testOrderError = HBError{
		Message: "order does not exist",
		Status:  "unknown",
	}
)

func TestHBClient(t *testing.T) {
	http.HandleFunc(fmt.Sprintf("/order/%s", testOrder.UID), func(w http.ResponseWriter, r *http.Request) {
		by, err := json.Marshal(&testOrder)
		if err != nil {
			panic(err)
		}
		w.Header().Add("content-type", "application/json")
		w.Write(by)
	})

	http.HandleFunc("/order/error", func(w http.ResponseWriter, r *http.Request) {
		by, err := json.Marshal(&testOrderError)
		if err != nil {
			panic(err)
		}
		w.WriteHeader(404)
		w.Header().Add("content-type", "application/json")
		w.Write(by)
	})

	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(fmt.Errorf("net.Listen: %s", err))
	}

	srv := http.Server{}
	go func() {
		if err := srv.Serve(listener); err != nil {
			panic(fmt.Errorf("srv.ListenAndServe: %s", err))
		}
	}()

	addrParts := strings.Split(listener.Addr().String(), ":")
	hbClient := NewClient(WithAPIURL(fmt.Sprintf("http://localhost:%s", addrParts[len(addrParts)-1])))

	testOrderBy, _ := json.Marshal(testOrder)
	testOrderErrorBy := []byte(errors.Errorf("%s %s", testOrderError.Status, testOrderError.Message).Error())
	dd := []struct {
		In  string
		Out []byte
	}{
		{
			In:  testOrder.UID,
			Out: testOrderBy,
		},
		{
			In:  "error",
			Out: testOrderErrorBy,
		},
	}
	for _, d := range dd {
		o, err := hbClient.GetOrder(d.In)
		if o == nil && err != nil {
			if !reflect.DeepEqual([]byte(err.Error()), d.Out) {
				t.Errorf("%s - expected errors to match", d.In)
			}
		} else if err != nil {
			t.Fatalf("hbClient.GetOrder: %s", err)
		} else {
			oBy, _ := json.Marshal(o)
			if !reflect.DeepEqual(oBy, d.Out) {
				t.Errorf("%s - expected orders to match", d.In)
			}
		}
	}
}
