/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"archive/tar"
	"bin-manager/binary"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:   "download [binaries...]",
	Short: "Download specified binaries from binman.yaml",
	Long: `Download all binaries specified in the binman.yaml. 
You can download only parts of the yaml by adding them as parameter.`,
	Run: func(cmd *cobra.Command, args []string) {
		binaries, err := readBinariesConfig()
		if err != nil {
			fmt.Printf("Error reading binaries config: %v\n", err)
			return
		}

		downloadDir, err := createDownloadDir()
		if err != nil {
			fmt.Printf("Error creating download directory: %v\n", err)
			return
		}

		system := runtime.GOOS
		cpu := runtime.GOARCH

		if len(args) == 0 {
			for key, bin := range binaries {
				downloadBinary(key, bin, downloadDir, system, cpu)
			}
		} else {
			for _, key := range args {
				bin, exists := binaries[key]
				if !exists {
					fmt.Printf("Binary %s not found in configuration\n", key)
					continue
				}
				downloadBinary(key, bin, downloadDir, system, cpu)
			}
		}
		fmt.Println("download called")
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}

func readBinariesConfig() (map[string]binary.Binary, error) {
	ex, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("error getting executable path: %w", err)
	}
	exPath := filepath.Dir(ex)

	yamlFile, err := ioutil.ReadFile(filepath.Join(exPath, "binman.yaml"))
	if err != nil && os.IsNotExist(err) {
		yamlFile, err = ioutil.ReadFile("binman.yaml")
	}
	if err != nil {
		return nil, fmt.Errorf("error reading binman.yaml: %w", err)
	}

	binaries := make(map[string]binary.Binary)
	if err := yaml.Unmarshal(yamlFile, &binaries); err != nil {
		return nil, fmt.Errorf("error unmarshalling binman.yaml: %w", err)
	}

	return binaries, nil
}

func createDownloadDir() (string, error) {
	bmanPath := os.Getenv("BMAN_PATH")
	var downloadDir string

	if bmanPath != "" {
		downloadDir = filepath.Join(bmanPath, "bin")
	} else {
		ex, err := os.Executable()
		if err != nil {
			return "", fmt.Errorf("error getting executable path: %w", err)
		}
		exPath := filepath.Dir(ex)
		downloadDir = filepath.Join(exPath, "bin")
	}

	if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("error creating download directory: %w", err)
	}
	return downloadDir, nil
}

func downloadBinary(key string, bin binary.Binary, downloadDir, system, cpu string) {
	url := replacePlaceholders(bin.Url, bin.Version, system, cpu)
	fmt.Printf("Downloading %s from %s\n", key, url)

	resp, err := http.Get(url)
	if err != nil {
		fmt.Printf("Error downloading %s: %v\n", key, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Printf("Error downloading %s: received status code %d\n", key, resp.StatusCode)
		return
	}

	// Extract the filename from the URL
	filename := filepath.Base(url)
	outPath := filepath.Join(downloadDir, filename)

	if err := saveToFile(outPath, resp.Body); err != nil {
		fmt.Printf("Error saving %s: %v\n", key, err)
		return
	}

	if strings.HasSuffix(outPath, ".tar.gz") || strings.HasSuffix(outPath, ".tgz") {
		fmt.Printf("Extracting archive %s\n", outPath)
		extractDir := filepath.Join(downloadDir, key+"_extracted")
		if err := untar(outPath, extractDir); err != nil {
			fmt.Printf("Error extracting %s: %v\n", key, err)
			return
		}
		fmt.Printf("Successfully extracted %s\n", key)
		os.Remove(outPath) // Remove the archive after extraction

		// Move the extracted file with OriginalName to the bin folder and delete the rest
		extractedFile := filepath.Join(extractDir, replacePlaceholders(bin.OriginalName, bin.Version, system, cpu))
		finalPath := filepath.Join(downloadDir, key)
		fmt.Printf("Renaming %s to %s\n", extractedFile, finalPath)
		if err := os.Rename(extractedFile, finalPath); err != nil {
			fmt.Printf("Error renaming %s to %s: %v\n", extractedFile, finalPath, err)
			return
		}
		outPath = finalPath

		// Clean up extracted directory
		os.RemoveAll(extractDir)
	}

	fmt.Printf("Setting executable permissions for %s\n", outPath)
	if err := os.Chmod(outPath, 0755); err != nil {
		fmt.Printf("Error making %s executable: %v\n", key, err)
		return
	}

	fmt.Printf("Successfully downloaded and made %s executable\n", key)
}

func replacePlaceholders(s, version, system, cpu string) string {
	s = strings.ReplaceAll(s, "${version}", version)
	s = strings.ReplaceAll(s, "${system}", system)
	s = strings.ReplaceAll(s, "${cpu}", cpu)
	return s
}

func saveToFile(path string, body io.Reader) error {
	fmt.Printf("Saving to file %s\n", path)
	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("error creating file: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, body); err != nil {
		return fmt.Errorf("error saving file: %w", err)
	}
	return nil
}

func untar(src, dest string) error {
	fmt.Printf("Opening archive %s\n", src)
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	fmt.Printf("Creating gzip reader for %s\n", src)
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error reading tar header: %v\n", err)
			return err
		}

		target := filepath.Join(dest, header.Name)
		fmt.Printf("Extracting %s to %s\n", header.Name, target)

		// Ensure the parent directory exists
		if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
			fmt.Printf("Error creating directory %s: %v\n", filepath.Dir(target), err)
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			fmt.Printf("Creating directory %s\n", target)
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				fmt.Printf("Error creating directory %s: %v\n", target, err)
				return err
			}
		case tar.TypeReg:
			fmt.Printf("Creating file %s\n", target)
			outFile, err := os.Create(target)
			if err != nil {
				fmt.Printf("Error creating file %s: %v\n", target, err)
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				fmt.Printf("Error copying to file %s: %v\n", target, err)
				return err
			}
			outFile.Close()
		default:
			fmt.Printf("Unknown type: %v in %s\n", header.Typeflag, header.Name)
			return fmt.Errorf("unknown type: %v in %s", header.Typeflag, header.Name)
		}
	}

	return nil
}
