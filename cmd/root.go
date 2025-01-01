/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "bin-manager",
	Short: "Managing binaries in a team",
	Long: `Managing your binaries. Set versions. Download binaries, etc.

The binman.yaml file should contain the configuration for each binary, including the URL, version, 
original name, and any headers required for the download. The URL can contain placeholders for 
version, system, and CPU architecture, which will be replaced with the appropriate values.

Example binman.yaml:

binman:
  url: https://github.com/juliankr/binman/releases/download/${version}/bin-manager-${system}-${cpu}
  version: 0.0.4
  originalName: bin-manager-${system}-${cpu}
  source:
   - "export PATH=${binman-path}/bin:$PATH"
   - "BMAN_PATH=${binman-path}"
yq:
  originalName: yq_${system}_${cpu}
  url: https://github.com/mikefarah/yq/releases/download/${version}/yq_${system}_${cpu}.tar.gz
  version: v4.44.6
kubectl:
  url: https://dl.k8s.io/release/${version}/bin/${system}/${cpu}/kubectl
  version: v1.25.0
kustomize:
  url: https://github.com/kubernetes-sigs/kustomize/releases/download/kustomize%2F${version}/kustomize_${version}_${system}_${cpu}.tar.gz
  version: v5.5.0
  originalName: kustomize
argocd:
  url: https://github.com/argoproj/argo-cd/releases/download/${version}/argocd-${system}-${cpu}
  version: v2.13.2
  originalName: argocd-${system}-${cpu}
private-release:
  url: https://api.github.com/repos/juliankr/private-release/releases/assets/
  urlPostfix:
    darwin-arm64: 217034482
    darwin-amd64: 217034481
    linux-arm64: 217034479
    linux-amd64: 217034480
  version: 1.0.1
  header:
    - "Authorization: token ${GITHUB_TOKEN}"
    - "Accept: application/octet-stream"
`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.bin-manager.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
