/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
	"bin-manager/binary" // Import the binary package
)

// bootstrapCmd represents the bootstrap command
var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Initialize the bin-manager",
	Long: `Create a folder for all the binaries managed by bin-manager.
	It will download the binaries yaml from the provided git repo. In case you 
	did not yet setup the git repo, you can do so within this command.`,
	Run: func(cmd *cobra.Command, args []string) {
		if !isGitInstalled() {
			fmt.Println("Error: git is not installed. Please install git and try again.")
			os.Exit(1)
		}

		gitrepo, _ := cmd.Flags().GetString("gitrepo")
		if gitrepo == "" {
			fmt.Print("Please enter the URL of the git repository containing the binaries yaml: ")
			fmt.Scanln(&gitrepo)
		}

		location, _ := cmd.Flags().GetString("location")
		if location == "" {
			defaultLocation := os.Getenv("HOME") + "/.binman"
			fmt.Printf("Please enter the location to create the folder for binaries (default: %s): ", defaultLocation)
			fmt.Scanln(&location)
			if location == "" {
				location = defaultLocation
			}
		}

		if err := createDirectory(location); err != nil {
			fmt.Printf("Failed to create directory: %s\n", err)
			return
		}

		if isGitRepo(location) {
			fmt.Printf("The folder %s is already a git repository.\n", location)
		} else {
			if err := cloneRepository(gitrepo, location); err != nil {
				fmt.Printf("Failed to clone repository: %s\n", err)
				return
			}
		}

		if err := createBinmanYaml(location); err != nil {
			fmt.Printf("Failed to create binman.yaml file: %s\n", err)
			return
		}

		if err := copyBinary(location); err != nil {
			fmt.Printf("Failed to copy binary: %s\n", err)
			return
		}

		if err := updateGitignore(location); err != nil {
			fmt.Printf("Failed to update .gitignore file: %s\n", err)
			return
		}

		if err := createSourceFile(location); err != nil {
			fmt.Printf("Failed to create .source file: %s\n", err)
			return
		}

		fmt.Println("")
		fmt.Println("")
		fmt.Printf("bootstrap called with gitrepo: %s and location: %s\n", gitrepo, location)
		fmt.Println("Please push the changes to your repository.")
		fmt.Printf("Add the following line to your shell configuration file to include %s/bin in your PATH:\n", location)
		fmt.Printf("For bash:\n  echo 'source %s/.source' >> ~/.bashrc\n", location)
		fmt.Printf("For zsh:\n  echo 'source %s/.source' >> ~/.zshrc\n", location)
	},
}

func isGitInstalled() bool {
	_, err := exec.LookPath("git")
	return err == nil
}

func createDirectory(location string) error {
	if _, err := os.Stat(location); os.IsNotExist(err) {
		return os.MkdirAll(location, os.ModePerm)
	}
	return nil
}

func isGitRepo(location string) bool {
	gitDir := filepath.Join(location, ".git")
	if _, err := os.Stat(gitDir); err == nil {
		return true
	}
	return false
}

func cloneRepository(gitrepo, location string) error {
	fmt.Printf("Cloning repository %s into %s\n", gitrepo, location)
	gitCmd := exec.Command("git", "clone", gitrepo, location)
	gitCmd.Stdout = os.Stdout
	gitCmd.Stderr = os.Stderr
	return gitCmd.Run()
}

func createBinmanYaml(location string) error {
	binmanFilePath := filepath.Join(location, "binman.yaml")
	if _, err := os.Stat(binmanFilePath); os.IsNotExist(err) {
		fmt.Printf("Creating binman.yaml file in %s\n", location)
		bin := binary.Binary{
			OriginalName: "bin-manager-${system}-${cpu}",
			Url:          "https://github.com/juliankr/binman/releases/download/0.0.1/bin-manager-${system}-${cpu}",
			Version:      "0.0.4",
		}
		binman := map[string]binary.Binary{
			"binman": bin,
		}
		content, err := yaml.Marshal(binman)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(binmanFilePath, content, 0644)
	}
	fmt.Printf("The folder %s already contains a binman.yaml file.\n", location)
	return nil
}

func copyBinary(location string) error {
	binmanBinaryPath := filepath.Join(location, "bin", "binman")
	if _, err := os.Stat(binmanBinaryPath); os.IsNotExist(err) {
		fmt.Printf("Copying current binary to %s\n", binmanBinaryPath)
		if err := os.MkdirAll(filepath.Dir(binmanBinaryPath), os.ModePerm); err != nil {
			return err
		}
		currentBinaryPath, err := os.Executable()
		if err != nil {
			return err
		}
		input, err := ioutil.ReadFile(currentBinaryPath)
		if err != nil {
			return err
		}
		return ioutil.WriteFile(binmanBinaryPath, input, 0755)
	}
	return nil
}

func updateGitignore(location string) error {
	gitignorePath := filepath.Join(location, ".gitignore")
	var gitignoreContent string
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		gitignoreContent = "bin\n.source\n"
	} else {
		content, err := ioutil.ReadFile(gitignorePath)
		if err != nil {
			return err
		}
		gitignoreContent = string(content)
		if !strings.Contains(gitignoreContent, "bin") {
			gitignoreContent += "\nbin\n"
		}
		if !strings.Contains(gitignoreContent, ".source") {
			gitignoreContent += "\n.source\n"
		}
	}
	return ioutil.WriteFile(gitignorePath, []byte(gitignoreContent), 0644)
}

func createSourceFile(location string) error {
	sourceFilePath := filepath.Join(location, ".source")
	sourceContent := fmt.Sprintf("export PATH=\"%s/bin:$PATH\"\n", location)
	return ioutil.WriteFile(sourceFilePath, []byte(sourceContent), 0644)
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
	bootstrapCmd.Flags().String("gitrepo", "", "URL of the git repository containing the binaries yaml")
	bootstrapCmd.Flags().String("location", "", "Location to create the folder for binaries")
}
