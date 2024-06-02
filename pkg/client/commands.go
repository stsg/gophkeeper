package client

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"

	postgres "github.com/stsg/gophkeeper/pkg/store"
	"golang.org/x/term"
)

func (c *Client) List(ctx context.Context) error {
	cr, err := c.authenticate(ctx)
	if err != nil {
		return err
	}

	resources, err := c.Store.List(ctx, cr)
	if err != nil {
		return err
	}

	// TODO: print resources as bubble list
	fmt.Printf("resources: %+v", resources)
	return nil
}

func (c *Client) Register(ctx context.Context) error {
	input := bufio.NewReader(os.Stdin)

	fmt.Print("Type new identity's username: ")
	username, err := input.ReadString('\n')
	if err != nil {
		return err
	}
	username = strings.TrimSuffix(username, "\n")

	fmt.Print("Type new identity's password: ")
	pass1, err := term.ReadPassword((int)(syscall.Stdin))
	if err != nil {
		return err
	}

	fmt.Println()
	fmt.Print("Retype new identity's password: ")
	pass2, err := term.ReadPassword((int)(syscall.Stdin))
	if err != nil {
		return err
	}
	fmt.Println()

	if !bytes.Equal(pass1, pass2) {
		return errors.New("passwords do not match")
	}

	cr := postgres.Creds{
		Login: username,
		Passw: (string)(pass1),
	}
	if err := c.Store.Register(ctx, cr); err != nil {
		return err
	}

	return nil
}
