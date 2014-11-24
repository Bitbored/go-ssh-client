package main

import (
	"fmt"
	"github.com/bitbored/go-ansicon"
	"github.com/flynn-archive/go-crypto-ssh"
	"github.com/flynn-archive/go-crypto-ssh/terminal"
	"os"
	"os/user"
	"strings"
)

func main() {
	if len(os.Args) < 1 {
		printHelp()
	}
	args := strings.Split(os.Args[1], "@")
	var username, hostname string

	if len(args) < 2 {
		user, err := user.Current()
		if err != nil {
			fmt.Println("Failed to get current user")
			os.Exit(1)
		}
		username = user.Username
		hostname = args[0]
	} else {
		username = args[0]
		hostname = args[1]
	}

	fmt.Printf("%s@%s's password: ", username, hostname)
	password, err := terminal.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		fmt.Println("Could not read password")
		os.Exit(1)
	}

	err = openClient(username, hostname, string(password))
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
}

func printHelp() {
	fmt.Println("usage: ssh [user@]hostname")
	os.Exit(0)
}

func openClient(username, hostname, password string) error {
	config := &ssh.ClientConfig{
		User: username,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
	}

	sshClient, err := ssh.Dial("tcp", hostname+":22", config)
	if err != nil {
		return err
	}

	if err = shell(sshClient); err != nil {
		return err
	}

	return nil
}

func shell(client *ssh.Client) error {
	session, finalize, err := makeSession(client)
	if err != nil {
		return err
	}

	defer finalize()
	if err = session.Shell(); err != nil {
		return err
	}
	session.Wait()
	return nil
}

func makeSession(client *ssh.Client) (session *ssh.Session, finalize func(), err error) {
	session, err = client.NewSession()
	if err != nil {
		return
	}

	session.Stdout = ansicon.Convert(os.Stdout)
	session.Stderr = os.Stderr
	session.Stdin = os.Stdin

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

	fd := int(os.Stdin.Fd())
	if terminal.IsTerminal(fd) {

		var termWidth, termHeight int
		var oldState *terminal.State

		oldState, err = terminal.MakeRaw(fd)
		if err != nil {
			return
		}

		finalize = func() {
			session.Close()
			terminal.Restore(fd, oldState)
		}

		termWidth, termHeight, err = terminal.GetSize(fd)
		if err != nil {
			// Ignore this error and use default terminal size to support Windows
			termWidth = 80
			termHeight = 24
		}
		err = session.RequestPty("xterm-256color", termHeight, termWidth, modes)
	} else {
		finalize = func() {
			session.Close()
		}
	}
	return
}
