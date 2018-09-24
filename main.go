package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

// Config ...
type Config struct {
	ProjectPath    string          `env:"project_path,dir"`
	ExpoCLIVersion string          `env:"expo_cli_verson,required"`
	UserName       string          `env:"user_name"`
	Password       stepconf.Secret `env:"password"`
}

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

	log.Printf("$ " + cmd.PrintableCommandArgs())
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

	log.Printf("$ " + cmd.PrintableCommandArgs())
	return cmd.Run()
}

// Eject command creates Xcode and Android Studio projects for your app.
func (e Expo) eject() error {
	args := []string{"eject", "--non-interactive", "--eject-method", e.Method.String()}

	cmd := command.New("expo", args...)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)

	log.Printf("$ " + cmd.PrintableCommandArgs())
	return cmd.Run()
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	log.Warnf("For more details you can enable the debug logs by turning on the verbose step input.")
	os.Exit(1)
}

func validateUserNameAndpassword(userName string, password stepconf.Secret) error {
	if userName != "" && string(password) == "" {
		return fmt.Errorf("user name is specified but password is not provided")
	}

	if userName == "" && string(password) != "" {
		return fmt.Errorf("password is specified but is not provided user name")
	}
	return nil
}

func main() {
	var cfg Config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Issue with input: %s", err)
	}

	fmt.Println()
	stepconf.Print(cfg)

	if err := validateUserNameAndpassword(cfg.UserName, cfg.Password); err != nil {
		failf("Input validation error: %s", err)
	}

	//
	// Select the --eject-method
	ejectMethod := Plain
	fmt.Println()
	log.Infof("Define --eject-method")
	{
		if cfg.UserName != "" {
			ejectMethod = ExpoKit
			log.Printf("Expo account credentials have provided => Set the --eject-method to %s", ejectMethod)
		} else {
			log.Printf("Expo account credentials have not provided => Set the --eject-method to %s", ejectMethod)
		}
	}

	e := Expo{
		Version: cfg.ExpoCLIVersion,
		Method:  ejectMethod,
	}

	//
	// Install expo-cli
	fmt.Println()
	log.Infof("Install Expo CLI version: %s", cfg.ExpoCLIVersion)
	{
		if err := e.installExpoCLI(); err != nil {
			failf("Failed to install the selected (%s) version for Expo CLI, error: %s", cfg.ExpoCLIVersion, err)
		}
	}

	//
	// Logging in the user to the Expo account
	fmt.Println()
	log.Infof("Login to Expo")
	{
		switch ejectMethod {
		case ExpoKit:
			if err := e.login(cfg.UserName, cfg.Password); err != nil {
				failf("Failed to log in to your provided Expo account, error: %s", err)
			}
		case Plain:
			log.Printf("--eject-method has been set to plain => Skip...")
		}
	}

	//
	// Logging out the user from the Expo account at the end of the step (even if it fails)
	defer func() {
		fmt.Println()
		log.Infof("Logging out from Expo")
		{
			if e.Method == ExpoKit {
				if err := e.logout(); err != nil {
					log.Warnf("Failed to log out from your Expo account, error: %s", err)
				}
			} else if e.Method == ExpoKit {
				log.Printf("Logout input was set to false => Skip...")
			} else {
				log.Printf("You were not logged in => Skip...")
			}
		}
	}()

	//
	// Eject project via the Expo CLI
	fmt.Println()
	log.Infof("Eject project")
	{
		if err := e.eject(); err != nil {
			failf("Failed to eject project (%s), error: %s", filepath.Base(cfg.ProjectPath), err)
		}

	}

	fmt.Println()
	log.Donef("Successfully ejected your project")
}
