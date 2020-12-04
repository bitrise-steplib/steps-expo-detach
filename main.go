package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

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

	e := Expo{
		Version: cfg.ExpoCLIVersion,
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
		if err := runPublish(e, cfg); err != nil {
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

func runPublish(expo Expo, cfg Config) error {
	//
	// Logging in the user to the Expo account
	fmt.Println()
	log.Infof("Login to Expo")
	{
		if err := expo.login(cfg.UserName, cfg.Password); err != nil {
			return fmt.Errorf("failed to log in to your provided Expo account, error: %s", err)
		}
	}

	//
	// Logging out the user from the Expo account (even if it fails)
	defer func() {
		fmt.Println()
		log.Infof("Logging out from Expo")
		{
			if err := expo.logout(); err != nil {
				log.Warnf("Failed to log out from your Expo account, error: %s", err)
			}
		}
	}()

	fmt.Println()
	log.Infof("Running expo publish")

	// Running publish
	if err := expo.publish(); err != nil {
		return err
	}

	return nil
}
