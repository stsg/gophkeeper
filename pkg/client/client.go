package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	postgres "github.com/stsg/gophkeeper/pkg/store"
)

type Client struct {
	Opts    options
	Store   *postgres.Storage
	HClient *http.Client
}

type options struct {
	URL     string        `short:"s" long:"server" env:"SERVER" default:"localhost:8080" description:"server connection address"`
	Command string        `short:"c" long:"command" env:"COMMAND" default:"list" description:"command to execute"`
	DBURI   string        `short:"d" long:"dburi" env:"DBURI" default:"postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable" description:"database connection string"`
	Timeout time.Duration `short:"t" long:"timeout" env:"TIMEOUT" default:"10s" description:"connection timeout"`
	Dbg     bool          `long:"dbg" env:"DEBUG" description:"show debug info"`
}

func (c *Client) Run(ctx context.Context) error {
	fmt.Printf("gophkeeper client command %s\n", c.Opts.Command)

	switch c.Opts.Command {
	case "list":
		return c.List(ctx)
	// case "add-credentials":
	// 	return c.AddCredentials()
	// case "get-credentials":
	// 	return c.GetCredentials()
	// case "add-text":
	// 	return c.AddText()
	// case "get-text":
	// 	return c.GetText()
	// case "add-file":
	// 	return c.AddFile()
	// case "get-file":
	// 	return c.GetFile()
	// case "add-card":
	// 	return c.AddCard()
	// case "get-card":
	// 	return c.GetCard()
	case "register":
		return c.Register(ctx)
	}

	fmt.Printf("unknown command: %s\n", c.Opts.Command)
	return nil
}
