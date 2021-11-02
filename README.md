# action-package

This action builds packages from your code using [fpm](https://github.com/jordansissel/fpm)

Your packages must be specified in a file called packages.yml in your project root. 
The fields that can be provided in packages.yaml correspond to fpm command line flags and all given values will be
appended to the fpm command.

For possible fields refer to the usage example below.

## usage example

The example below contains all flag values that can be passed to fpm as of now

```yaml
# key packages contains an array of packages to build
# this key is required but it can be empty
packages:
  # example of a .deb package build from local directory
  - name: example

    # source of the package - specifies how to gather sources
    source:
      # using mode "dir" to collect files from local directories
      #
      # each source mode needs specific arguments
      mode: dir
      # excludes is a list of patterns or subdirectories that should not be included in the
      # resulting package
      excludes:
        - .git/

    # target of the package - specifies how the "source" files will be packaged
    target:
      # using mode deb
      mode: deb

      # version of the package
      # this field is required for deb packages and will be checked for
      version:      1.0


      # the following metadata fields serve information purposes
      # they exist to be displayed by package managers like aptly and are all optional

      # maintainer should contain an email address and will be displayed by apt *optional*
      maintainer:   max.mustermann@example.com
      # vendor of the package *optional*
      vendor:       example AG
      # an URL to display along with the package information *optional*
      url:          www.example.com
      # refer to a license here - any string is accepted *optional*
      license:      apache 2.0
      # description to display in aptly *optional*
      description:  |
        This is an example package.
        Files are taken from local directory bla and packaged as example_1.0_amd64.deb


      # the following metadata fields provide additional information about files
      # contained in the package

      # directories that are explicitly owned by the package can be added to this array
      directories:
        - /opt/example
        - /opt/example/conf
      # config_files that need to be preserved across updates
      config_files:
        - /opt/example/conf/example.conf
      # systemd units that cone with the package
      systemd:
        - lib/systemd/example.service


      # the following metadata fields provides information on how the package interacts
      # with other deb packages

      # dependencies of the package - those need to be installed
      dependencies:
        # require a package name
        - php7.2
        # require a specific minimal version
        - nodejs >= 12.10

      # suggested package to go along with the installation - those do not need to be installed
      suggests:
        - example-utils

      # set no_auto_depends to prevent fpm from automatically guessing and adding dependencies
      no_auto_depends: true


      # the following metadata fields attach shell scripts to specific installation events

      # scripts for handling package installation
      before_install: before-install.sh
      after_install:  after-install.sh
      # scripts for handling package removal
      before_remove:  before-remove.sh
      after_remove:   after-remove.sh
      # scripts for handling package upgrades
      before_upgrade: before-upgrade.sh
      after_upgrade:  after-upgrade.sh


      # the following metadata fields specify how to handle systemd units
      # they apply to all units specified in "systemd" key above

      # enable units after installation
      systemd_enable: true
      # start units after installation
      systemd_auto_start: true
      # re-start units after upgrade
      systemd_restart_after_upgrade: true

    # arguments will be appended to the fpm command (as arguments i.e. with no preceding flag)
    paths:
      # since we used target mode dir the first argument will be interpreted as a path
      - bla
```

## environment variables

You can use environment variables in the packages.yaml:

```yaml
packages:
  - name: ${ENV_NAME}
    .
    .
    .
```

## version from github
A common use case is using git tags as version numbers.
Tags are accessible to the GitHub action via the GITHUB_REF environment variable.

```yaml
packages:
    .
    .
    .
    version: ${GITHUB_REF}
    .
    .
    .
```

### tags
When a tag is build, GITHUB_REF contains the string "refs/tags/<tag_name>". If such a string appears as the version <tag_name> is extracted and used as version instead.

### branches
When a branch is build, GITHUB_REF contains the string "refs/heads/<branch_name>". If such a string appears as the version <branch_name> is extracted and the resulting
version is <branch_name>.<github_run_number>.

