package client

import (
	"context"
	"errors"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	postgres "github.com/stsg/gophkeeper/pkg/store"
)

type authModel struct {
	cancelled bool

	width, height int

	username textinput.Model
	password textinput.Model
}

func newAuthModel() authModel {
	var m = authModel{
		cancelled: false,
		username:  textinput.New(),
		password:  textinput.New(),
	}
	m.username.CharLimit = 32
	m.username.Prompt = "Username: "
	m.username.Placeholder = "type your username..."

	m.password.CharLimit = 32
	m.password.Prompt = "Password: "
	m.password.EchoMode = textinput.EchoPassword
	m.password.Placeholder = "type your password..."

	m.username.Focus()
	return m
}

// Init implements tea.Model.
func (a authModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update implements tea.Model.
func (a authModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		usernameCmd tea.Cmd
		passwordCmd tea.Cmd
	)
	a.username, usernameCmd = a.username.Update(msg)
	a.password, passwordCmd = a.password.Update(msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.height = msg.Height
		a.width = msg.Width
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			switch {
			case a.username.Focused():
				if len(a.username.Value()) < 1 {
					return a, textinput.Blink
				}
				a.username.Blur()
				a.password.Focus()
			case a.password.Focused():
				if len(a.password.Value()) < 1 {
					return a, textinput.Blink
				}
				a.password.Blur()
				return a, tea.Quit
			}
		case "ctrl+c", "esc":
			a.cancelled = true
			return a, tea.Quit
		}
	}
	return a, tea.Batch(usernameCmd, passwordCmd)
}

// View implements tea.Model.
func (a authModel) View() string {
	return form(
		a.width, a.height,
		"Authenticate to Gophkeeper",
		lipgloss.JoinVertical(
			lipgloss.Left,
			a.username.View(),
			strings.Repeat(" ", 64),
			a.password.View(),
		),
	)
}

func (cli *Client) authenticate(ctx context.Context) (postgres.Creds, error) {
	var m, err = tea.NewProgram(
		newAuthModel(),
		tea.WithAltScreen(),
		tea.WithContext(ctx),
	).Run()
	if err != nil {
		return postgres.Creds{}, err
	}
	if m.(authModel).cancelled {
		return postgres.Creds{}, errors.New("authentiation cancelled by user")
	}
	var credential = postgres.Creds{
		Login: m.(authModel).username.Value(),
		Passw: m.(authModel).password.Value(),
	}
	var token, tokenError = cli.Store.Authenticate(ctx, credential)
	if tokenError != nil {
		return postgres.Creds{}, tokenError
	}
	return cli.Store.Identity(ctx, token)
}
