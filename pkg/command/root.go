package command

import (
	"context"
	"flag"

	"diogogmt.com/hbd/pkg/hbclient"
	"github.com/peterbourgon/ff/v2/ffcli"
)

// RootCmd wraps the  config and a ffcli.Command
type RootCmd struct {
	Conf *RootConfig

	*ffcli.Command
}

// RootConfig has the config for the root command
type RootConfig struct {
	JWTCookie string
	Verbose   bool
	HBClient  *hbclient.HBDClient
}

// RootConfigOption defines the signature for functional options to be applied to the root command
type RootConfigOption = func(c *RootConfig)

// NewRootCmd creates a new RootCmd
func NewRootCmd(opts ...RootConfigOption) *RootCmd {
	fs := flag.NewFlagSet("hbd", flag.ExitOnError)

	conf := RootConfig{}
	for _, opt := range opts {
		opt(&conf)
	}

	cmd := RootCmd{
		Conf: &conf,
	}
	cmd.Command = &ffcli.Command{
		Name:       "hbd",
		ShortUsage: "hbd [flags] <subcommand>",
		ShortHelp:  "Interact with humble bundle API",
		FlagSet:    fs,
		Exec:       cmd.Exec,
	}
	cmd.RegisterFlags(fs)

	return &cmd
}

// RegisterFlags registers a set of flags for the root command
func (c *RootCmd) RegisterFlags(fs *flag.FlagSet) {
	fs.StringVar(&c.Conf.JWTCookie, "jwt", "", "humblebundle dashboard JWT _simpleauth_sess cookie")
	fs.BoolVar(&c.Conf.Verbose, "v", false, "log verbose output")
}

// Exec executes the root command
func (c *RootCmd) Exec(ctx context.Context, args []string) error {
	return nil
}

// WithHBClient sets a hbClient in the root command
func WithHBClient(hbClient *hbclient.HBDClient) RootConfigOption {
	return func(c *RootConfig) {
		c.HBClient = hbClient
	}
}
