package command

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"flag"
	"fmt"
	"hash"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

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

func HashCheckOne(fpath string, hash hash.Hash, label string, expected string) error {

	f, err := os.Open(fpath)
	if err != nil {
		return errors.Errorf("error reading file: %v for: %s", err, fpath)
	}
	defer f.Close()

	if _, err := io.Copy(hash, f); err != nil {
		return errors.Errorf("error calculating %s: %v for: %s", label, err, fpath)
	}
	bs := hash.Sum(nil)
	if expected != fmt.Sprintf("%x", bs) {
		return errors.Errorf("%s checksum failed for %s -- expected %s but got %x", label, fpath, expected, bs)
	}
	return nil
}

func HashCheck(fpath string, asset *hbclient.DownloadType) error {
	// Note:  I've seen files where the md5 passed but the sha1 failed
	if asset.MD5 != "" {
		return HashCheckOne(fpath, md5.New(), "MD5", asset.MD5)
	} else if asset.SHA1 != "" {
		return HashCheckOne(fpath, sha1.New(), "SHA1", asset.SHA1)
	}
	return nil
}

// Check results of Get or Head and parse Last-Modified
func HttpChecksAndTime(resp *http.Response, err error) (*http.Response, *time.Time, error) {

	if err != nil {
		return resp, nil, errors.Wrapf(err, "%s book %s", resp.Request.Method, resp.Request.URL.String())
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return resp, nil, errors.Errorf("invalid response status code %d", resp.StatusCode)
	}

	bookLastmodTime, err := http.ParseTime(resp.Header.Get("Last-Modified"))
	if err != nil {
		return resp, nil, errors.Wrapf(err, "http.ParseTime last-modified header %s", resp.Header.Get("Last-Modified"))
	}
	return resp, &bookLastmodTime, nil
}

// downloadAsset downlads the assets of a bundle
func (c *DownloadCmd) downloadAsset(ctx context.Context, asset *hbclient.DownloadType) error {
	filename := fmt.Sprintf("%s.%s", asset.HumanName, strings.ToLower(strings.TrimPrefix(asset.Name, ".")))
	downloadURL := asset.URL.Web

	// fix filename
	filename = strings.ReplaceAll(filename, "/", "_")
	if strings.HasSuffix(filename, ".supplement") {
		filename = strings.TrimSuffix(filename, ".supplement") + "_supplement.zip"
	}
	if strings.HasSuffix(filename, ".download") {
		filename = strings.TrimSuffix(filename, ".download") + "_video.zip"
	}

	fpath := fmt.Sprintf("%s/%s", c.Conf.Dest, filename)

	if HashCheck(fpath, asset) == nil {
		fmt.Println("Already exists:", fpath)
		resp, bookLastmodTime, err := HttpChecksAndTime(http.Head(downloadURL))
		if resp != nil {
			defer resp.Body.Close()
		}
		if err != nil {
			return err
		}

		if err = os.Chtimes(fpath, *bookLastmodTime, *bookLastmodTime); err != nil {
			return errors.Wrap(err, "os.Chtimes")
		}
		return nil
	}

	resp, bookLastmodTime, err := HttpChecksAndTime(http.Get(downloadURL))
	if resp != nil {
		defer resp.Body.Close()
	}
	if err != nil {
		return err
	}

	bookFile, err := os.Create(fpath)
	if err != nil {
		return errors.Wrapf(err, "os.Create %s", fpath)
	}
	defer bookFile.Close()

	fmt.Println("Downloading:", fpath)
	_, err = io.Copy(bookFile, resp.Body)
	if err != nil {
		return errors.Errorf("error copying response body to file (%s): %v", fpath, err)
	}

	if err := os.Chtimes(fpath, *bookLastmodTime, *bookLastmodTime); err != nil {
		return errors.Wrap(err, "os.Chtimes")
	}

	err = HashCheck(fpath, asset)
	if err != nil {
		return err
	}
	fmt.Println("Download complete:", fpath)

	return nil
}
