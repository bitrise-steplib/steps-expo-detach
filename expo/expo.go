package expo

import (
	"log"
	"os"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

// EjectMethod if the project is using Expo SDK and you choose the "plain" --eject-method those imports will stop working.
type EjectMethod string

// const ...
const (
	Plain   EjectMethod = "plain"
	ExpoKit EjectMethod = "expoKit"
)

func (m EjectMethod) String() string {
	return string(m)
}

// Expo CLI
type Expo struct {
	Version string
	Method  EjectMethod
}

// New return a new *Expo with the provided version and --eject-method
func New(version string, method EjectMethod) *Expo {
	return &Expo{
		Version: version,
		Method:  method,
	}
}

// InstallExpoCLI runs the install npm command to install the expo-cli
func (e *Expo) InstallExpoCLI() error {
	args := []string{"install", "-g"}
	if e.Version != "latest" {
		args = append(args, "expo-cli@"+e.Version)
	} else {
		args = append(args, "expo-cli")
	}

	cmd := command.New("npm", args...)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	log.Printf("$ " + cmd.PrintableCommandArgs())
	return cmd.Run()
}

// Login with your Expo account
func (e *Expo) Login(userName string, password stepconf.Secret) error {
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
func (e *Expo) Logout() error {
	cmd := command.New("expo", "logout", "--non-interactive")
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	log.Printf("$ " + cmd.PrintableCommandArgs())
	return cmd.Run()
}

// Eject command creates Xcode and Android Studio projects for your app.
func (e *Expo) Eject() error {
	args := []string{"eject", "--non-interactive", "--eject-method", e.Method.String()}

	cmd := command.New("expo", args...)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	log.Printf("$ " + cmd.PrintableCommandArgs())
	return cmd.Run()
}
