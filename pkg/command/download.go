package command

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"sync"

	"diogogmt.com/hbd/pkg/hbclient"
	"github.com/peterbourgon/ff/v2/ffcli"
	"github.com/pkg/errors"
)

// DownloadCmd wraps the download config and a ffcli.Command
type DownloadCmd struct {
	Conf *DownloadConfig

	*ffcli.Command
}

// DownloadConfig has the config for the download command and a reference to the root command config
type DownloadConfig struct {
	RootConf *RootConfig

	Key       string
	Dest      string
	Types     map[string]struct{}
	TypesFlag string
}

// NewDownloadCmd creates a new DownloadCmd
func NewDownloadCmd(rootConf *RootConfig) *DownloadCmd {
	conf := DownloadConfig{
		RootConf: rootConf,
		Types:    map[string]struct{}{},
	}
	cmd := DownloadCmd{
		Conf: &conf,
	}
	fs := flag.NewFlagSet("hbd download", flag.ExitOnError)
	cmd.RegisterFlags(fs)

	cmd.Command = &ffcli.Command{
		Name:       "download",
		ShortUsage: "hbd download",
		ShortHelp:  "Download assets from bundle",
		FlagSet:    fs,
		Exec:       cmd.Exec,
	}
	return &cmd
}

// RegisterFlags registers a set of flags for the download command
func (c *DownloadCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.Key, "key", "", "purchase key")
	fs.StringVar(&c.Conf.Dest, "dest", "", "directory to download all bundle assets")
	fs.StringVar(&c.Conf.TypesFlag, "types", "all", "comma separated list of file types, eg; pdf,epub,mobi")
}

// Exec executes the download command
func (c *DownloadCmd) Exec(ctx context.Context, args []string) error {
	if c.Conf.Key == "" {
		return errors.Errorf("missing key")
	}
	for _, t := range strings.Split(c.Conf.TypesFlag, ",") {
		c.Conf.Types[strings.ToLower(t)] = struct{}{}
	}

	order, err := c.Conf.RootConf.HBClient.GetOrder(c.Conf.Key)
	if err != nil {
		return errors.Wrap(err, "HBClient.GetOrder")
	}
	if c.Conf.Dest == "" {
		c.Conf.Dest = fmt.Sprintf("./%s", strings.ReplaceAll(order.Product.HumanName, "/", "_"))
	}
	if err := c.downloadBundle(ctx, order); err != nil {
		return errors.Wrap(err, "download bundle")
	}
	return nil
}

// downloadBundle fetches a bundle order and download all its assets
func (c *DownloadCmd) downloadBundle(ctx context.Context, order *hbclient.Order) error {
	_ = os.MkdirAll(c.Conf.Dest, 0777)
	downloadTypes := []*hbclient.DownloadType{}
	for i := 0; i < len(order.Products); i++ {
		prod := order.Products[i]
		for j := 0; j < len(prod.Downloads); j++ {
			download := prod.Downloads[j]
			for x := 0; x < len(download.Types); x++ {
				dt := download.Types[x]
				_, all := c.Conf.Types["all"]
				_, includeType := c.Conf.Types[strings.ToLower(dt.Name)]
				if !all && !includeType {
					continue
				}
				dt.HumanName = prod.HumanName
				downloadTypes = append(downloadTypes, dt)
			}
		}
	}

	var errs []string
	errCh := make(chan string)

	var group sync.WaitGroup
	for x := 0; x < len(downloadTypes); x++ {
		downloadType := downloadTypes[x]
		group.Add(1)
		go func() {
			defer group.Done()
			if err := c.downloadAsset(ctx, downloadType); err != nil {
				errCh <- errors.Wrapf(err, "downloadAsset %s.%s", downloadType.HumanName, downloadType.Name).Error()
			}
		}()
	}
	go func() {
		group.Wait()
		close(errCh)
	}()

	for {
		err, ok := <-errCh
		if !ok {
			break
		}
		errs = append(errs, err)
	}

	if len(errs) != 0 {
		return errors.Errorf(strings.Join(errs, " - "))
	}

	return nil
}

// downloadAsset downlads the assets of a bundle
func (c *DownloadCmd) downloadAsset(ctx context.Context, asset *hbclient.DownloadType) error {
	filename := fmt.Sprintf("%s.%s", asset.HumanName, strings.ToLower(strings.TrimPrefix(asset.Name, ".")))
	downloadURL := asset.URL.Web
	filename = strings.ReplaceAll(filename, "/", "_")
	resp, err := http.Get(downloadURL)
	if err != nil {
		return errors.Wrapf(err, "http.Get book %s", downloadURL)
	}
	defer resp.Body.Close()

	bookLastmodTime, err := http.ParseTime(resp.Header.Get("Last-Modified"))
	if err != nil {
		return errors.Wrapf(err, "http.ParseTime last-modified header %s", resp.Header.Get("Last-Modified"))
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.Errorf("invalid response status code %d", resp.StatusCode)
	}

	s, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "ioutil.ReadAll response body")
	}

	bookFile, err := os.Create(fmt.Sprintf("%s/%s", c.Conf.Dest, filename))
	if err != nil {
		return errors.Wrapf(err, "os.Create %s/%s", c.Conf.Dest, filename)
	}
	defer bookFile.Close()

	if _, err = bookFile.Write(s); err != nil {
		return errors.Wrap(err, "writting book file")
	}
	if err := os.Chtimes(fmt.Sprintf("%s/%s", c.Conf.Dest, filename), bookLastmodTime, bookLastmodTime); err != nil {
		return errors.Wrap(err, "os.Chtimes")
	}

	if asset.SHA1 != "" {
		hash := sha1.New()
		hash.Write([]byte(s))
		bs := hash.Sum(nil)
		if asset.SHA1 != fmt.Sprintf("%x", bs) {
			return errors.Errorf("SHA1 checksum failed for %s -- expected %s but got %x", filename, asset.SHA1, bs)
		}
	}
	if asset.MD5 != "" {
		hash := md5.New()
		hash.Write([]byte(s))
		md5Checksum := fmt.Sprintf("%x", hash.Sum(nil))
		if asset.MD5 != md5Checksum {
			return errors.Errorf("MD5 checksum failed for %s -- expected %s but got %s", filename, asset.MD5, md5Checksum)
		}
	}
	return nil
}
