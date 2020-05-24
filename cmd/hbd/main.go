package main

import (
	"context"
	"fmt"
	"os"

	"diogogmt.com/hbd/pkg/command"
	"diogogmt.com/hbd/pkg/hbclient"
	"github.com/peterbourgon/ff/v2/ffcli"
)

func main() {
	rootCmd := command.NewRootCmd()
	downloadCmd := command.NewDownloadCmd(rootCmd.Conf)

	rootCmd.Subcommands = []*ffcli.Command{
		downloadCmd.Command,
	}

	if err := rootCmd.Parse(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "error during Parse: %v\n", err)
		os.Exit(1)
	}

	hbClient := hbclient.NewClient(hbclient.WithJWT(rootCmd.Conf.JWTCookie))

	command.WithHBClient(hbClient)(rootCmd.Conf)

	if err := rootCmd.Run(context.Background()); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
