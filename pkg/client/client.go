package client

import (
	"context"
	"fmt"
	"time"
)

type Client struct {
	// TODO: implement me
	options options
}

type options struct {
	// TODO: implement me
	URL     string        `short:"s" long:"server" env:"SERVER" default:"localhost:8080" description:"server connection address"`
	Command string        `short:"c" long:"command" env:"COMMAND" default:"list" description:"command to execute"`
	Timeout time.Duration `short:"t" long:"timeout" env:"TIMEOUT" default:"10s" description:"connection timeout"`
	Dbg     bool          `long:"dbg" env:"DEBUG" description:"show debug info"`
}

func NewClient(opts options) *Client {
	return &Client{
		options: opts,
	}
}

func (c *Client) Run(ctx context.Context) error {
	// TODO: implement me
	fmt.Printf("gophkeeper client command %s\n", c.options.Command)
	return nil
}
