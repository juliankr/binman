# Binary Manager

Manage your team binaries and used versions through git. Use the same versions in CI/CD pipelines and on your local system.

## Overview

Binary Manager is a tool designed to help teams manage their binaries efficiently. It allows you to set versions, download binaries, and ensure consistency across different environments. The configuration is managed through a `binman.yaml` file, which contains the details of each binary, including the URL, version, original name, and any headers required for the download.

## Commands
### download
The download command downloads binaries specified in the binman.yaml configuration file. You can specify which binaries to download by providing their names as arguments. If no arguments are provided, all binaries in the configuration will be downloaded.

Usage
`bin-manager download [binaries...]`
Example
`bin-manager download`
### bootstrap
The bootstrap command initializes the Binary Manager by creating a folder for all the binaries managed by Binary Manager. It will download the binaries YAML from the provided git repository. If the git repository is not set up, you can do so within this command.

Usage
`bin-manager bootstrap --gitrepo <git-repo-url> --location <location>`

Example
`bin-manager bootstrap --gitrepo https://github.com/your-org/your-repo.git --location ~/.binman`
Installation
To install Binary Manager, you can download the latest release from the GitHub releases page: https://github.com/juliankr/binman/releases

## Configuration

The `binman.yaml` file should contain the configuration for each binary. The URL can contain placeholders for version, system, and CPU architecture, which will be replaced with the appropriate values.

### Example `binman.yaml`

## Keep binaries up to date
You can use renovate to keep your binaries up to date. To tell renovate, where it should fetch the versions, you need to add a comment above the version.
` # renovate: datasource=<DATASOURCE> depName=<DEPENDENCY_NAME>`
E.g for github releases you can use
`# renovate: datasource=github-releases depName=<REPO e.g. mikfeah/yq>`

The corresponding config for renovite is under renovate/config.js. 

