package cmd

import (
	"archive/tar"
	"archive/zip"
	"bin-manager/binary"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var downloadCmd = &cobra.Command{
	Use:   "download [binaries...]",
	Short: "Download binaries specified in binman.yaml",
	Run: func(cmd *cobra.Command, args []string) {
		binaries, err := readBinariesConfig()
		if (err != nil) {
			fmt.Printf("Error reading binaries config: %v\n", err)
			return
		}

		downloadDir, err := createDownloadDir()
		if err != nil {
			fmt.Printf("Error creating download directory: %v\n", err)
			return
		}

		if len(args) == 0 {
			for key, bin := range binaries {
				downloadBinary(key, bin, downloadDir)
			}
		} else {
			for _, key := range args {
				bin, exists := binaries[key]
				if !exists {
					fmt.Printf("Binary %s not found in configuration\n", key)
					continue
				}
				downloadBinary(key, bin, downloadDir)
			}
		}

		if err := createSourceFile(downloadDir, binaries); err != nil {
			fmt.Printf("Error creating .source file: %v\n", err)
			return
		}
	},
}

func init() {
	rootCmd.AddCommand(downloadCmd)
}

func readBinariesConfig() (map[string]binary.Binary, error) {
	bmanPath, err := binary.GetBmanPath()
	if err != nil {
		return nil, err
	}

	yamlFile, err := ioutil.ReadFile(filepath.Join(bmanPath, "binman.yaml"))
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
		downloadDir = exPath
	}

	if err := os.MkdirAll(downloadDir, os.ModePerm); err != nil {
		return "", fmt.Errorf("error creating download directory: %w", err)
	}
	return downloadDir, nil
}

func downloadBinary(key string, bin binary.Binary, downloadDir string) {
	url := bin.GetUrl()
	fmt.Printf("Downloading %s from %s\n", key, url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Printf("Error creating request for %s: %v\n", key, err)
		return
	}

	for _, header := range bin.Header {
		parts := strings.SplitN(header, ": ", 2)
		if len(parts) == 2 {
			value := os.ExpandEnv(parts[1])
			req.Header.Set(parts[0], value)
		}
	}

	resp, err := http.DefaultClient.Do(req)
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

	finalPath := filepath.Join(downloadDir, key)
	if bin.SubPath != "" {
		finalPath = filepath.Join(downloadDir, bin.SubPath, key)
	}

	if strings.HasSuffix(outPath, ".tar.gz") || strings.HasSuffix(outPath, ".tgz") {
		if err := extractAndRename(outPath, finalPath, bin, downloadDir, untar); err != nil {
			fmt.Printf("Error extracting and renaming %s: %v\n", key, err)
			return
		}
	} else if strings.HasSuffix(outPath, ".zip") {
		if err := extractAndRename(outPath, finalPath, bin, downloadDir, unzip); err != nil {
			fmt.Printf("Error extracting and renaming %s: %v\n", key, err)
			return
		}
	} else {
		if err := os.Rename(outPath, finalPath); err != nil {
			fmt.Printf("Error renaming %s: %v\n", key, err)
			return
		}
	}

	if fileInfo, err := os.Stat(finalPath); err == nil && !fileInfo.IsDir() {
		if err := os.Chmod(finalPath, 0755); err != nil {
			fmt.Printf("Error making %s executable: %v\n", key, err)
			return
		}
	}

	fmt.Printf("Successfully downloaded and made %s executable\n", key)
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

func extractAndRename(src, dest string, bin binary.Binary, downloadDir string, extractFunc func(string, string) error) error {
	extractDir := filepath.Join(downloadDir, filepath.Base(dest)+"_extracted")
	if err := extractFunc(src, extractDir); err != nil {
		return fmt.Errorf("error extracting %s: %w", src, err)
	}
	os.Remove(src)

	return renameExtracted(extractDir, dest, bin)
}

func renameExtracted(extractDir, dest string, bin binary.Binary) error {
	if bin.OriginalName == "" {
		if _, err := os.Stat(dest); err == nil {
			if err := os.RemoveAll(dest); err != nil {
				return fmt.Errorf("error removing existing directory %s: %w", dest, err)
			}
		}
		if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory %s: %w", filepath.Dir(dest), err)
		}
		if err := os.Rename(extractDir, dest); err != nil {
			return fmt.Errorf("error renaming %s to %s: %w", extractDir, dest, err)
		}
	} else {
		extractedFile := filepath.Join(extractDir, binary.ReplacePlaceholders(bin.OriginalName, bin.Version))
		if _, err := os.Stat(dest); err == nil {
			if err := os.RemoveAll(dest); err != nil {
				return fmt.Errorf("error removing existing binary %s: %w", dest, err)
			}
		}
		if err := os.MkdirAll(filepath.Dir(dest), os.ModePerm); err != nil {
			return fmt.Errorf("error creating directory %s: %w", filepath.Dir(dest), err)
		}
		if err := os.Rename(extractedFile, dest); err != nil {
			return fmt.Errorf("error renaming %s to %s: %w", extractedFile, dest, err)
		}
		os.RemoveAll(extractDir)
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

func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path: %s", fpath)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return err
		}

		outFile, err := os.Create(fpath)
		if err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		if _, err := io.Copy(outFile, rc); err != nil {
			outFile.Close()
			rc.Close()
			return err
		}

		outFile.Close()
		rc.Close()
	}
	return nil
}

func createSourceFile(location string, binaries map[string]binary.Binary) error {
	bmanPath, err := binary.GetBmanPath()
	if err != nil {
		return err
	}

	sourceFilePath := filepath.Join(bmanPath, ".source")
	if err := os.Remove(sourceFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error removing existing .source file: %w", err)
	}

	var sourceContent strings.Builder
	for _, bin := range binaries {
		for _, line := range bin.Source {
			sourceContent.WriteString(binary.ReplacePlaceholders(line, bin.Version) + "\n")
		}
	}
	return ioutil.WriteFile(sourceFilePath, []byte(sourceContent.String()), 0644)
}