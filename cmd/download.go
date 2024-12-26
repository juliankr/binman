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

var downloadCmd = &cobra.Command{
	Use:   "download [binaries...]",
	Short: "Download binaries specified in binman.yaml",
	Long: `Download binaries listed in the binman.yaml configuration file.
You can specify which binaries to download by providing their names as arguments.
If no arguments are provided, all binaries in the configuration will be downloaded.`,
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

	filename := filepath.Base(url)
	outPath := filepath.Join(downloadDir, filename)

	if err := saveToFile(outPath, resp.Body); err != nil {
		fmt.Printf("Error saving %s: %v\n", key, err)
		return
	}

	if strings.HasSuffix(outPath, ".tar.gz") || strings.HasSuffix(outPath, ".tgz") {
		extractDir := filepath.Join(downloadDir, key+"_extracted")
		if err := untar(outPath, extractDir); err != nil {
			fmt.Printf("Error extracting %s: %v\n", key, err)
			return
		}
		os.Remove(outPath)

		originalName := bin.OriginalName
		if originalName == "" {
			originalName = key
		}

		extractedFile := filepath.Join(extractDir, replacePlaceholders(originalName, bin.Version, system, cpu))
		finalPath := filepath.Join(downloadDir, key)
		if _, err := os.Stat(finalPath); err == nil {
			if err := os.Remove(finalPath); err != nil {
				fmt.Printf("Error removing existing binary %s: %v\n", key, err)
				return
			}
		}
		if err := os.Rename(extractedFile, finalPath); err != nil {
			fmt.Printf("Error renaming %s to %s: %v\n", extractedFile, finalPath)
			return
		}
		outPath = finalPath
		os.RemoveAll(extractDir)
	}

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
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

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
			return err
		}

		target := filepath.Join(dest, header.Name)

		if err := os.MkdirAll(filepath.Dir(target), os.ModePerm); err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.ModePerm); err != nil {
				return err
			}
		case tar.TypeReg:
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		default:
			return fmt.Errorf("unknown type: %v in %s", header.Typeflag, header.Name)
		}
	}

	return nil
}
