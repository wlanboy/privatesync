package store

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

/*Prompter asks the user for input; AskSecret must not echo the answer*/
type Prompter interface {
	Ask(question string) (string, error)
	AskSecret(question string) (string, error)
}

/*ErrNoInput is returned when stdin is closed before an answer was given*/
var ErrNoInput = errors.New("no input available")

/*TerminalPrompter reads answers from a reader and writes questions to out*/
type TerminalPrompter struct {
	in  *bufio.Reader
	out io.Writer
	fd  int
	tty bool
}

/*NewTerminalPrompter creates a prompter; secrets are read without echo if in is a terminal*/
func NewTerminalPrompter(in *os.File, out io.Writer) *TerminalPrompter {
	fd := int(in.Fd())
	return &TerminalPrompter{
		in:  bufio.NewReader(in),
		out: out,
		fd:  fd,
		tty: term.IsTerminal(fd),
	}
}

/*Ask reads one line of visible input*/
func (p *TerminalPrompter) Ask(question string) (string, error) {
	fmt.Fprintf(p.out, "Enter %s: ", question)
	return readLine(p.in)
}

/*AskSecret reads one line without echo when attached to a terminal*/
func (p *TerminalPrompter) AskSecret(question string) (string, error) {
	fmt.Fprintf(p.out, "Enter %s: ", question)
	if !p.tty {
		return readLine(p.in)
	}
	secret, err := term.ReadPassword(p.fd)
	fmt.Fprintln(p.out)
	if err != nil {
		return "", err
	}
	return string(secret), nil
}

func readLine(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && !(errors.Is(err, io.EOF) && line != "") {
		if errors.Is(err, io.EOF) {
			return "", ErrNoInput
		}
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}
