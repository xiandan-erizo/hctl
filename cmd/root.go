/*
Copyright © 2022 xiandan HERE xiandan-erizo@outlook.com
*/
package cmd

import (
	"bytes"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	hconfig "hctl/config"
	"io/ioutil"
	crdClientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
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
	cfgFile    string
	msg        string
	bot        string
	namespace  string
	hctlconfig hconfig.HConfig
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "hctl",
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
	flags.StringVarP(&bot, "bot", "b", "", "飞书机器人地址")
	flags.StringVarP(&msg, "msg", "m", "服务重启完成", "需要发送的消息")
	flags.StringVarP(&namespace, "namespace", "n", "", "命名空间")
}

// Cli cmd struct
type Cli struct {
	rootCmd *cobra.Command
}

// NewCli returns the cli instance used to register and execute command
func NewCli() *Cli {
	cli := &Cli{
		rootCmd: &cobra.Command{
			Use: "hctl",
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
	yamlFile, err := ioutil.ReadFile(filepath.Join(homeDir(), ".hctl", "config.yaml"))
	if err != nil {
		hctlconfig.Feishu.Url = "https://open.feishu.cn/open-apis/bot/v2/hook/daa4ff06-226a-4fdc-8c26-2e049e618ad5"
		hctlconfig.Feishu.Msg = "服务重启完成"
		hctlconfig.Dump.OssPath = "/javadump/"
		hctlconfig.Dump.OssRs = "centos"
	} else {
		err = yaml.Unmarshal(yamlFile, &hctlconfig)
		if err != nil {
			fmt.Println(err.Error())
		}
	}
}

func getClient(cfgFile string, restype string, namespace string) (string, discovery.DiscoveryInterface, *crdClientset.Clientset, *dynamic.DynamicClient) {
	config, _ := clientcmd.LoadFromFile(cfgFile)
	if client != nil {
		return contNs, client, nil, nil
	}
	currentContext := config.CurrentContext
	if namespace != "" {
		contNs = namespace
	} else {
		contNs = config.Contexts[currentContext].Namespace
	}

	configN, err := clientcmd.BuildConfigFromFlags("", cfgFile)
	if err != nil {
		panic(err)
	}

	if restype == "deployment" {
		client, _ = kubernetes.NewForConfig(configN)
		return contNs, client, nil, nil
	} else
	//} else if restype == "cloneset" {
	//	kruiseConfig, err := clientcmd.BuildConfigFromFlags("", cfgFile)
	//	if err != nil {
	//		panic(err.Error())
	//	}
	//
	//	client = kruiseclientset.NewForConfigOrDie(kruiseConfig)
	//	return contNs, client, nil
	//}
	{
		// 创建dynamic
		crdClient := crdClientset.NewForConfigOrDie(configN)
		client := dynamic.NewForConfigOrDie(configN)
		return contNs, nil, crdClient, client
	}

}
