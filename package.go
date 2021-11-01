package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"strings"
)

// fomConfig contains all configuration needed to create a package using fpm
type FPMConfig struct {
	Packages []struct {

		// the name of the target package
		Name string

		// section Source of the fpm config
		// defines where and how to source the contents of the package
		Source struct {
			// source mode specifies how to gather the files contained in the package
			//
			// "dir":
			// use mode dir to source files from a local directory
			// a valid configuration using "dir" needs at least one argument containing a path
			//
			// Mode is REQUIRED
			Mode string `yaml:"mode"`

			// Excludes is used with mode "dir"
			// paths to files that are explicitly not part of the packages source files
			Excludes []string `yaml:"excludes"`

			Chdir string `yaml:"chdir"`
		} `yaml:"source"`

		// section Target of the fpm config
		Target struct {
			// Mode specifies the kind of package to create *REQUIRED*
			//
			// "deb":
			// use mode "deb" to create a debian package
			// a valid configuration using "deb" needs flags "name"
			Mode string `yaml:"mode"`


			// package Version *REQUIRED*
			Version string `yaml:"version"`

			// Maintainer of the package *OPTIONAL*
			// should be an email address
			Maintainer string `yaml:"maintainer"`

			// Vendor of the package *OPTIONAL*
			Vendor string `yaml:"vendor"`

			// project URL *OPTIONAL*
			// will be displayed in the packages metadata alongside the description
			URL string `yaml:"url"`

			License string `yaml:"license"`

			Description string `yaml:"description"`

			// special file tags
			Directories []string `yaml:"directories"`
			ConfigFiles []string `yaml:"config_files"`
			Systemd     []string `yaml:"systemd"`

			// dependency management
			Depends       []string `yaml:"depends"`
			Suggests      []string `yaml:"suggests"`
			NoAutoDepends bool     `yaml:"no_auto_depends"`
			Conflicts     []string `yaml:"conflicts"`

			// script tags
			BeforeInstall string `yaml:"before_install"`
			AfterInstall  string `yaml:"after_install"`

			BeforeRemove string `yaml:"before_remove"`
			AfterRemove  string `yaml:"after_remove"`

			BeforeUpgrade string `yaml:"before_upgrade"`
			AfterUpgrade  string `yaml:"after_upgrade"`

			SystemdEnable              bool `yaml:"systemd_enable"`
			SystemdAutoStart           bool `yaml:"systemd_auto_start"`
			SystemdRestartAfterUpgrade bool `yaml:"systemd_restart_after_upgrade"`
		}

		Paths []string `yaml:"paths"`
	}
}

// function readFile accepts a file path and reads the fpm configuration from that file
func (c *FPMConfig) ReadFile(path string) error {

	// read the file from disk
	fileContents, err := ioutil.ReadFile(path)

	// use ExpandEnv and attempt to insert ${ENVIRONMENT_VARIABLES}
	fileContents = []byte(os.ExpandEnv(string(fileContents)))

	if err != nil {
		return err
	}

	if err := yaml.Unmarshal(fileContents, c); err != nil {
		return err
	}

	return nil
}

// function contains decides if a given slice contains a given string
// arguments are named h (for haystack) and n (for needle)
// this function is not provided by golang and will be used in the check function below
func contains(h []string, n string) bool {
	for _, s := range h {
		// test each string in h for equality
		if s == n {
			// return true if a match is found
			return true
		}
	}
	// return false since no string in h matched n
	return false
}

// configError to provide structure to the output of the check method
type ConfigError struct {
	packageEntry string
	field        string
	message      string
}

// method Error provides a message for the ConfigError (and implements the Error interface)
func (c ConfigError) Error() string {
	return fmt.Sprintf("error in package %s:\n  -> config field %s missing or invalid\n  -> %s\n",
		c.packageEntry, c.field, c.message)
}

// method check to validate the fpm config
//
// if no error is returned the config is valid
// if there is an error it indicates what exactly is wrong with the config
func (c *FPMConfig) check() error {

	if len(c.Packages) == 0 {
		fmt.Print("packages.yml specifies no packages to build\n")
	}

	// check all packages
	for i, p := range c.Packages {
		// every package needs a name
		if p.Name == "" {
			return ConfigError{
				field:   fmt.Sprintf("package[%d].name", i),
				message: "name is required",
			}
		}

		// check if source mode is set to a valid mode
		validSourceModes := []string{"dir"}
		if !contains(validSourceModes, p.Source.Mode) {
			return ConfigError{
				packageEntry: p.Name,
				field:        "source.mode",
				message: fmt.Sprintf(
					"source mode is required and may contain %s", strings.Join(validSourceModes, "|")),
			}
		}

		// checks for source mode "dir"
		if p.Source.Mode == "dir" {

			// check whether directories were provided
			if len(p.Paths) < 1 && p.Source.Chdir == "" {
				return ConfigError{
					packageEntry: p.Name,
					field:        "arguments",
					message:      "for target mode dir at least one argument is required (a directory to package)",
				}
			}
		}

		// check if target mode is set to a valid mode
		validTargetModes := []string{"deb"}
		if !contains(validTargetModes, p.Target.Mode) {
			return ConfigError{
				packageEntry: p.Name,
				field:        "target.mode",
				message: fmt.Sprintf(
					"target mode is required and may contain %s", strings.Join(validSourceModes, "|")),
			}
		}

		// checks for target mode "deb"
		if p.Target.Mode == "deb" {
			if p.Target.Version == "" {
				return ConfigError{
					packageEntry: p.Name,
					field:        "target.version",
					message:      "debian packages require a version",
				}
			}
		}

	}

	return nil
}

// method build will create the packages as specified in packages.yml
func (c *FPMConfig) build() error {
	for _, p := range c.Packages {
		fmt.Printf("building package %s...\n", p.Name)

		// set flags that are always required
		args := []string{
			"-s", p.Source.Mode,
			"-t", p.Target.Mode,
		}

		// set version from file
		//
		// a SPECIAL CASE applies here where we extract a version from the github actions variable GITHUB_REF
		// GITHUB_REF is set to either:
		//    * refs/heads/<name> if the build is triggered for a branch
		//    * refs/tags/<name> if the build is triggered for a tag
		var gitHubDetect = regexp.MustCompile("^refs/(tags|heads)/([0-9a-zA-Z-.]+)$")
		var version string
		matches := gitHubDetect.FindAllStringSubmatch(p.Target.Version, -1)

		// if the version matches GITHUB_REF format
		if len(matches) == 1 {
			if matches[0][1] == "tags" {
				// for a tag set the tag name as version
				version = matches[0][2]
			} else {
				// for a branch set the branch name as version
				// additionally use the GITHUB_RUN_NUMBER to always remember which package is the latest
				version = fmt.Sprintf("%s.%s", os.Getenv("GITHUB_RUN_NUMBER"), matches[0][2])
			}
		} else {
			// version does not match the GITHUB_REF format - just use it as its given
			version = p.Target.Version
		}

		args = append(args, "-v", version)

		// special flags for the "dir" source mode
		if p.Source.Mode == "dir" {
			// append all exclude patterns to the command
			for _, e := range p.Source.Excludes {
				args = append(args, fmt.Sprintf("-x %s", e))
			}

                        if p.Source.Chdir != "" {
                                args = append(args, "-C", p.Source.Chdir)
                        }
		}

		// special flags for the "deb" target mode
		if p.Target.Mode == "deb" {
			// set package name
			args = append(args, "-n", p.Name)

			// metadata flags
			if p.Target.Maintainer != "" {
				args = append(args, "-m", p.Target.Maintainer)
			}
			if p.Target.URL != "" {
				args = append(args, "--url", p.Target.URL)
			}
			if p.Target.Vendor != "" {
				args = append(args, "--vendor", p.Target.Vendor)
			}
			if p.Target.Vendor != "" {
				args = append(args, "--license", p.Target.License)
			}

			// tag important files
			for _, d := range p.Target.Directories {
				args = append(args, "--directories", d)
			}
			for _, c := range p.Target.ConfigFiles {
				args = append(args, "--deb-config", c)
			}
			for _, s := range p.Target.Systemd {
				args = append(args, "--deb-systemd", s)
			}

			// append dependencies, suggests and conflicts
			for _, d := range p.Target.Depends {
				args = append(args, "-d", d)
			}
			for _, s := range p.Target.Suggests {
				args = append(args, "--deb-suggests", s)
			}
			for _, c := range p.Target.Conflicts {
				args = append(args, "--conflicts", c)
			}

			// add scripts
			if p.Target.BeforeInstall != "" {
				args = append(args, "--before-install", p.Target.BeforeInstall)
			}
			if p.Target.AfterInstall != "" {
				args = append(args, "--after-install", p.Target.AfterInstall)
			}
			if p.Target.BeforeRemove != "" {
				args = append(args, "--before-remove", p.Target.BeforeRemove)
			}
			if p.Target.AfterRemove != "" {
				args = append(args, "--after-remove", p.Target.AfterRemove)
			}
			if p.Target.BeforeUpgrade != "" {
				args = append(args, "--before-upgrade", p.Target.BeforeUpgrade)
			}
			if p.Target.AfterUpgrade != "" {
				args = append(args, "--after-upgrade", p.Target.AfterUpgrade)
			}

			// handle systemd units
			if p.Target.SystemdEnable == true {
				args = append(args, "--deb-systemd-enable")
			}
			if p.Target.SystemdAutoStart == true {
				args = append(args, "--deb-systemd-auto-start")
			}
			if p.Target.SystemdRestartAfterUpgrade == true {
				args = append(args, "--deb-systemd-restart-after-upgrade")
			}

		}

		// append arguments
		for _, a := range p.Paths {
			args = append(args, a)
		}

		// create the actual command
		buildCommand := exec.Command("fpm", args...)

		output, err := buildCommand.CombinedOutput()
		fmt.Printf(string(output))

		// exit with non-zero exit code in case the fpm command fails
		if err != nil {
			fmt.Printf("FPM command failed\n")
			os.Exit(2)
		}

		// print newlines to separate next package
		fmt.Printf("\n\n")
	}
	return nil
}

// main method
func main() {
	c := FPMConfig{}

	if err := c.ReadFile("packages.yml"); err != nil {
		fmt.Printf(err.Error())
	}

	if err := c.check(); err != nil {
		fmt.Printf(err.Error())
		os.Exit(1)
	}

	if err := c.build(); err != nil {
		fmt.Printf(err.Error())
	}

}
