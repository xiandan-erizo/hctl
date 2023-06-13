/*
Copyright © 2022 xiandan HERE xiandan-erizo@outlook.com
*/
package cmd

import (
	"bytes"
	"flag"
	"fmt"
	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	"gopkg.in/yaml.v2"
	hconfig "htl/config"
	"io/ioutil"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"
)

var (
	cfgFile   string
	msg       string
	bot       string
	htlconfig hconfig.HConfig
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
	flags.StringVarP(&bot, "bot", "b", "https://open.feishu.cn/open-apis/bot/v2/hook/daa4ff06-226a-4fdc-8c26-2e049e618ad5", "飞书机器人地址")
	flags.StringVarP(&msg, "msg", "m", "服务重启完成", "需要发送的消息")
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
	InitConfig()
	return cli
}
func InitConfig() {
	yamlFile, err := ioutil.ReadFile(filepath.Join(homeDir(), ".htl", "config.yaml"))
	fmt.Println("初始化配置文件")
	if err != nil {
		htlconfig.Feishu.Url = "https://open.feishu.cn/open-apis/bot/v2/hook/daa4ff06-226a-4fdc-8c26-2e049e618ad5"
		htlconfig.Feishu.Msg = "服务重启完成"
		htlconfig.Dump.OssPath = "/javadump/"
		htlconfig.Dump.OssRs = "centos"
	} else {
		err = yaml.Unmarshal(yamlFile, &htlconfig)
		if err != nil {
			fmt.Println(err.Error())
		}
		fmt.Printf("config.app: %#v\n", htlconfig.Feishu)
		fmt.Printf("config.log: %#v\n", htlconfig.Dump)
	}
}

func getClient(cfgFile string, restype string) (string, discovery.DiscoveryInterface) {
	config, _ := clientcmd.LoadFromFile(cfgFile)
	if client != nil {
		return contNs, client
	}
	currentContext := config.CurrentContext
	contNs := config.Contexts[currentContext].Namespace

	configN, err := clientcmd.BuildConfigFromFlags("", cfgFile)
	if err != nil {
		panic(err)
	}

	if restype == "deployment" {
		client, _ := kubernetes.NewForConfig(configN)
		return contNs, client
	}

	kruiseConfig, err := clientcmd.BuildConfigFromFlags("", cfgFile)
	if err != nil {
		panic(err.Error())
	}

	client = kruiseclientset.NewForConfigOrDie(kruiseConfig)
	return contNs, client
}
