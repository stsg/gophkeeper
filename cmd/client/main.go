// Package main contains all application logic
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/umputun/go-flags"

	"github.com/stsg/gophkeeper/pkg/client"
)

type Client interface {
	Run(ctx context.Context) error
	Register() error
	List() error
	AddCredentials() error
	GetCredentials() error
	AddText() error
	GetText() error
	AddFile() error
	GetFile() error
	AddCard() error
	GetCard() error
	Delete() error
}

var revision = "unknown"

var opts struct {
	URL     string        `short:"s" long:"server" env:"SERVER" default:"localhost:8080" description:"server connection address"`
	Command string        `short:"c" long:"command" env:"COMMAND" default:"list" description:"command to execute"`
	Timeout time.Duration `short:"t" long:"timeout" env:"TIMEOUT" default:"10s" description:"connection timeout"`
	Dbg     bool          `long:"dbg" env:"DEBUG" description:"show debug info"`
}

func main() {
	fmt.Printf("gophkeeper client %s\n", revision)

	p := flags.NewParser(&opts, flags.PassDoubleDash|flags.HelpFlag)
	if _, err := p.Parse(); err != nil {
		if err.(*flags.Error).Type != flags.ErrHelp {
			fmt.Printf("%s\n", err)
			os.Exit(1)
		}
		p.WriteHelp(os.Stderr)
		os.Exit(2)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cli := client.NewClient(opts)
	err := cli.Run(ctx)
	if err != nil {
		fmt.Printf("[ERROR] failed to run client: %v", err)
		os.Exit(1)
	}

}
