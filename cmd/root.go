/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "htl",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func (cli *Cli) Execute() error {
	return cli.rootCmd.Execute()
}

func homeDir() string {
	u, err := user.Current()
	if nil == err {
		return u.HomeDir
	}
	// cross compile support
	if runtime.GOOS == "windows" {
		return homeWindows()
	}
	// Unix-like system, so just assume Unix
	return homeUnix()
}

func homeUnix() string {
	if home := os.Getenv("HOME"); home != "" {
		return home
	}
	var stdout bytes.Buffer
	cmd := exec.Command("sh", "-c", "eval echo ~$USER")
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		return ""
	}
	result := strings.TrimSpace(stdout.String())
	if result == "" {
		fmt.Println("blank output when reading home directory")
		os.Exit(0)
	}

	return result
}
func homeWindows() string {
	drive := os.Getenv("HOMEDRIVE")
	path := os.Getenv("HOMEPATH")
	home := drive + path
	if drive == "" || path == "" {
		home = os.Getenv("USERPROFILE")
	}
	if home == "" {
		fmt.Println("HOMEDRIVE, HOMEPATH, and USERPROFILE are blank")
		os.Exit(0)
	}

	return home
}

func (cli *Cli) setFlags() {
	kubeconfig := flag.String("kubeconfig", filepath.Join(homeDir(), ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	flags := cli.rootCmd.PersistentFlags()
	flags.StringVar(&cfgFile, "config", *kubeconfig, "path of kubeconfig")
}

// Cli cmd struct
type Cli struct {
	rootCmd *cobra.Command
}

// NewCli returns the cli instance used to register and execute command
func NewCli() *Cli {
	cli := &Cli{
		rootCmd: &cobra.Command{
			Use: "htl",
		},
	}
	cli.rootCmd.SetOut(os.Stdout)
	cli.rootCmd.SetErr(os.Stderr)
	cli.setFlags()
	cli.rootCmd.DisableAutoGenTag = true
	return cli
}
