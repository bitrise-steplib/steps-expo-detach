package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/errorutil"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-tools/go-steputils/stepconf"
	"github.com/bitrise-tools/xcode-project/serialized"
)

// Config ...
type Config struct {
	Workdir                    string          `env:"project_path,dir"`
	ExpoCLIVersion             string          `env:"expo_cli_verson,required"`
	UserName                   string          `env:"user_name"`
	Password                   stepconf.Secret `env:"password"`
	RunPublish                 string          `env:"run_publish"`
	OverrideReactNativeVersion string          `env:"override_react_native_version"`
}

// EjectMethod if the project is using Expo SDK and you choose the "plain" --eject-method those imports will stop working.
type EjectMethod string

// const ...
const (
	Plain   EjectMethod = "plain"
	ExpoKit EjectMethod = "expoKit"
)

// Expo CLI
type Expo struct {
	Version string
	Method  EjectMethod
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

	log.Donef("$ " + cmd.PrintableCommandArgs())
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
	args := []string{"eject", "--non-interactive", "--eject-method", string(e.Method)}

	cmd := command.New("expo", args...)
	cmd.SetStdout(os.Stdout)
	cmd.SetStderr(os.Stderr)
	if e.Workdir != "" {
		cmd.SetDir(e.Workdir)
	}

	log.Donef("$ " + cmd.PrintableCommandArgs())
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

	log.Donef("\n$ " + cmd.PrintableCommandArgs())
	return cmd.Run()
}

func parsePackageJSON(pth string) (serialized.Object, error) {
	b, err := fileutil.ReadBytesFromFile(pth)
	if err != nil {
		return nil, fmt.Errorf("Failed to read package.json file: %s", err)
	}

	var packages serialized.Object
	if err := json.Unmarshal(b, &packages); err != nil {
		return nil, fmt.Errorf("Failed to parse package.json file: %s", err)
	}
	return packages, nil
}

func savePackageJSON(packages serialized.Object, pth string) error {
	b, err := json.MarshalIndent(packages, "", "  ")
	if err != nil {
		return fmt.Errorf("Failed to serialize modified package.json file: %s", err)
	}

	if err := fileutil.WriteBytesToFile(pth, b); err != nil {
		return fmt.Errorf("Failed to write modified package.json file: %s", err)
	}
	return nil
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
	os.Exit(1)
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
		Workdir: cfg.Workdir,
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
			failf("Failed to eject project: %s", err)
		}

	}

	fmt.Println()
	log.Donef("Successfully ejected your project")

	if cfg.RunPublish == "yes" {
		fmt.Println()
		log.Infof("Running expo publish")

		if err := e.publish(); err != nil {
			failf("Failed to publish project: %s", err)
		}
	}

	if cfg.OverrideReactNativeVersion != "" {
		//
		// Force certain version of React Native in package.json file
		fmt.Println()
		log.Infof("Set react-native dependency version: %s", cfg.OverrideReactNativeVersion)

		packageJSONPth := filepath.Join(cfg.Workdir, "package.json")
		packages, err := parsePackageJSON(packageJSONPth)
		if err != nil {
			failf(err.Error())
		}

		deps, err := packages.Object("dependencies")
		if err != nil {
			failf("Failed to parse dependencies from package.json file: %s", err)
		}

		deps["react-native"] = cfg.OverrideReactNativeVersion
		packages["dependencies"] = deps

		if err := savePackageJSON(packages, packageJSONPth); err != nil {
			failf(err.Error())
		}

		//
		// Install new node dependencies
		log.Printf("install new node dependencies")

		nodeDepManager := "npm"
		if exist, err := pathutil.IsPathExists(filepath.Join(cfg.Workdir, "yarn.lock")); err != nil {
			log.Warnf("Failed to check if yarn.lock file exists in the workdir: %s", err)
		} else if exist {
			nodeDepManager = "yarn"
		}

		cmd := command.New(nodeDepManager, "install")
		out, err := cmd.RunAndReturnTrimmedCombinedOutput()
		if err != nil {
			if errorutil.IsExitStatusError(err) {
				failf("%s failed: %s", cmd.PrintableCommandArgs(), out)
			}
			failf("%s failed: %s", cmd.PrintableCommandArgs(), err)
		}
	}
}
