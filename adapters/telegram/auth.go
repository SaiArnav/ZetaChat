package telegram

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
)

// interactiveAuth is a UserAuthenticator that routes prompts through the
// adapter's Prompter so the TUI (or CLI) can collect inputs.
type interactiveAuth struct {
	a *Adapter
}

func (ia interactiveAuth) Phone(ctx context.Context) (string, error) {
	return ia.a.prompter.AskPhone()
}

func (ia interactiveAuth) Code(ctx context.Context, sentCode *tg.AuthSentCode) (string, error) {
	return ia.a.prompter.AskCode()
}

func (ia interactiveAuth) Password(ctx context.Context) (string, error) {
	return ia.a.prompter.AskPassword()
}

func (ia interactiveAuth) AcceptTermsOfService(ctx context.Context, tos tg.HelpTermsOfService) error {
	ok, err := ia.a.prompter.AskTerms("Terms of Service", tos.Text)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("telegram: terms of service not accepted")
	}
	return nil
}

func (ia interactiveAuth) SignUp(ctx context.Context) (auth.UserInfo, error) {
	return auth.UserInfo{}, errors.New("telegram: sign-up is not supported")
}

// stdinPrompter prompts on the terminal for the CLI mode.
type stdinPrompter struct{}

func (stdinPrompter) prompt(label string) (string, error) {
	fmt.Printf("%s: ", label)
	line, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

func (s stdinPrompter) AskPhone() (string, error) {
	return s.prompt("Enter phone number (e.g. +15551234567)")
}

func (s stdinPrompter) AskCode() (string, error) {
	return s.prompt("Enter the login code")
}

func (s stdinPrompter) AskPassword() (string, error) {
	return s.prompt("Enter your two-factor password")
}

func (s stdinPrompter) AskTerms(title, text string) (bool, error) {
	fmt.Printf("--- %s ---\n%s\n", title, text)
	ans, err := s.prompt("Accept terms? [y/N]")
	if err != nil {
		return false, err
	}
	return strings.EqualFold(ans, "y"), nil
}
