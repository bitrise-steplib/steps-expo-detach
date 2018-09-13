package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-steplib/steps-expo-detach/expo"
	"github.com/bitrise-tools/go-steputils/stepconf"
)

// Config ...
type Config struct {
	ProjectPath    string          `env:"project_path,dir"`
	ExpoCLIVersion string          `env:"expo_cli_verson,required"`
	UserName       string          `env:"user_name"`
	Password       stepconf.Secret `env:"password"`
	Logout         bool            `env:"logout,opt[true,false]"`
}

func main() {
	var cfg Config
	if err := stepconf.Parse(&cfg); err != nil {
		failf("Issue with input: %s", err)
	}

	if err := validateUserNameAndpassword(cfg.UserName, cfg.Password); err != nil {
		failf("Input validation error: %s", err)
	}

	//
	// Select the --eject-method
	ejectMethod := expo.Plain
	fmt.Println()
	log.Infof("Define --eject-method")
	{
		if cfg.UserName != "" {
			ejectMethod = expo.ExpoKit
			log.Printf("Expo account credentials have provided => Set the --eject-method to %s", ejectMethod)
		} else {
			log.Printf("Expo account credentials have not provided => Set the --eject-method to %s", ejectMethod)
		}
	}

	e := expo.New(cfg.ExpoCLIVersion, ejectMethod)

	//
	// Install expo-cli
	fmt.Println()
	log.Infof("Install Expo CLI version: %s", cfg.ExpoCLIVersion)
	{
		if err := e.InstallExpoCLI(); err != nil {
			failf("Failed to install the selected (%s) version for Expo CLI, error: %s", cfg.ExpoCLIVersion, err)
		}
	}

	//
	// Logging in the user to the Expo account
	fmt.Println()
	log.Infof("Login: %s", cfg.ExpoCLIVersion)
	{
		switch ejectMethod {
		case expo.ExpoKit:
			if err := e.Login(cfg.UserName, cfg.Password); err != nil {
				failf("Failed to log in to your provided Expo account, error: %s", err)
			}
		case expo.Plain:
			log.Printf("--eject-method has been set to plain => Skip...")
		}
	}

	//
	// Eject project via the Expo CLI
	fmt.Println()
	log.Infof("Eject project: %s", cfg.ExpoCLIVersion)
	{
		if err := e.Eject(); err != nil {
			failf("Failed to eject project (%s), error: %s", filepath.Base(cfg.ProjectPath), err)
		}

	}

	//
	// Logging out the user from the Expo accountc
	fmt.Println()
	log.Infof("Loging out from Expo")
	{
		if e.Method == expo.ExpoKit && cfg.Logout {
			if err := e.Logout(); err != nil {
				log.Warnf("Failed to log out from your Expo account, error: %s", err)
			}
		} else if e.Method == expo.ExpoKit {
			log.Printf("Logout input was set to false => Skip...")
		} else {
			log.Printf("You were not logged in => Skip...")
		}
	}

	fmt.Println()
	log.Donef("Successfully ejected your project")
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

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	log.Warnf("For more details you can enable the debug logs by turning on the verbose step input.")
	os.Exit(1)
}
