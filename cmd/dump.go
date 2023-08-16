package cmd

import (
	"fmt"
	"github.com/spf13/cobra"
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
	dc.CobraCmd().Flags().StringP("pd", "", "", "pod path to javadump")
	dc.CobraCmd().Flags().StringP("po", "", "", "oss path to javadump")
}

func (dc *DumpCommand) runDump(cmd *cobra.Command, args []string) error {
	//
	cmdType = append(cmdType, "/bin/bash", "-c")
	podName, _ = dc.command.Flags().GetString("pod")
	podPath, _ := dc.command.Flags().GetString("pd")
	if podPath == "" {
		podPath = hctlconfig.Dump.OssPod
	}

	// 进入目标pod 打dump
	commandJmap := []string{"export TIME=`date -d 'today' +'%H%M'` && \\\njmap -dump:format=b,file=" + podPath + "$HOSTNAME-$TIME.hprof 1"}
	result, errorout, err := ExecCommandPod(podName, commandJmap)
	if err != nil && errorout != "" {
		fmt.Println(result)
	}

	//TODO 进入oss pod 压缩并上传

	return err
}
