package main

import (
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

// Expo ...
type Expo struct {
	Version string
	Workdir string
}

// installExpoCLI runs the install npm command to install the expo-cli
func (e Expo) installExpoCLI() error {
	args := []string{"install", "-g"}
	if e.Version != "latest" {
		args = append(args, "expo-cli@"+e.Version)
	} else {
		args = append(args, "expo-cli")
	}

	cmd := command.New("npm", args...)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	log.Donef("$ %s", cmd.PrintableCommandArgs())
	return cmd.Run()
}

// Login with your Expo account
func (e Expo) login(userName string, password stepconf.Secret) error {
	args := []string{"login", "--non-interactive", "-u", userName, "-p", string(password)}

	cmd := command.New("expo", args...)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	nonFilteredArgs := ("$ " + cmd.PrintableCommandArgs())
	fileredArgs := strings.Replace(nonFilteredArgs, string(password), "[REDACTED]", -1)
	log.Printf(fileredArgs)

	return cmd.Run()
}

// Logout from your Expo account
func (e Expo) logout() error {
	cmd := command.New("expo", "logout", "--non-interactive")
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	log.Donef("$ %s", cmd.PrintableCommandArgs())
	return cmd.Run()
}

// Eject command creates Xcode and Android Studio projects for your app.
func (e Expo) eject() error {
	args := []string{"eject", "--non-interactive"}

	cmd := command.New("expo", args...)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)
	if e.Workdir != "" {
		cmd.SetDir(e.Workdir)
	}

	log.Donef("$ %s", cmd.PrintableCommandArgs())
	return cmd.Run()
}

func (e Expo) publish() error {
	args := []string{"publish", "--non-interactive"}

	cmd := command.New("expo", args...)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)
	if e.Workdir != "" {
		cmd.SetDir(e.Workdir)
	}

	log.Donef("$ %s", cmd.PrintableCommandArgs())
	return cmd.Run()
}
