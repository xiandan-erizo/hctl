package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
)

var (
	podName string
	cmdType []string
)

type DumpCommand struct {
	BaseCommand
}

func (dc *DumpCommand) Init() {
	dc.command = &cobra.Command{
		Use:   "dump",
		Short: "抓取dump文件并输出",
		Long:  `选择pod,并制作dump`,
		Run: func(cmd *cobra.Command, args []string) {
			err := dc.runDump(cmd, args)
			if err != nil {
				fmt.Println(err)
			}
		},
	}

	dc.CobraCmd().Flags().StringP("pod", "p", "", "pod name to get dump")
	dc.CobraCmd().Flags().StringP("cmd", "c", "", "command to execute")
}

func (dc *DumpCommand) runDump(cmd *cobra.Command, args []string) error {
	//
	cmdType = append(cmdType, "/bin/bash", "-c")
	podName, _ = dc.command.Flags().GetString("pod")
	commandJmap := []string{"export TIME=`date -d 'today' +'%H%M'` && \\\njmap -dump:format=b,file=/javatmp/$HOSTNAME-$TIME.hprof 1"}
	result, errorout, err := ExecCommandPod(podName, commandJmap)
	if err != nil && errorout != "" {
		fmt.Println(result)

	}
	// 获取当前ns
	config, _ := clientcmd.LoadFromFile(cfgFile)
	currentContext := config.CurrentContext
	contNs := config.Contexts[currentContext].Namespace
	token := config.AuthInfos[currentContext].Token

	fmt.Println(contNs)
	fmt.Println(token)
	split := strings.FieldsFunc(result, func(r rune) bool {
		return strings.ContainsRune("\r\n", r)
	})
	filePath := strings.Split(split[0], " ")[4]
	// 创建一个web服务,
	fmt.Println(filePath)

	return err
}
