/*
Copyright © 2022 xiandan HERE xiandan-erizo@outlook.com
*/
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-resty/resty/v2"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"strings"
	"time"
)

var (
	dep string
)

type StatsComand struct {
	BaseCommand
}

func (sc *StatsComand) Init() {
	sc.command = &cobra.Command{
		Use:   "stats",
		Short: "循环读取deployments状态,直至成功并发送消息",
		Long:  `循环读取当前命名空间deployments状态,对比副本数,直至成功并发送飞书消息`,
		Run: func(cmd *cobra.Command, args []string) {
			err := sc.runStats(cmd, args)
			if err != nil {
				fmt.Println(err)
			}
		},
	}

	sc.command.Flags().StringVarP(&dep, "deployment", "d", "", "需要检测的deployment")
}

func (sc *StatsComand) runStats(cmd *cobra.Command, args []string) error {
	msg, _ = sc.command.Flags().GetString("msg")
	bot, _ = sc.command.Flags().GetString("bot")
	dep, _ = sc.command.Flags().GetString("dep")

	// 获取当前ns
	config, _ := clientcmd.LoadFromFile(cfgFile)
	currentContext := config.CurrentContext
	contNs := config.Contexts[currentContext].Namespace
	configN, err := clientcmd.BuildConfigFromFlags("", cfgFile)
	if err != nil {
		return err
	}
	clientSet, err := kubernetes.NewForConfig(configN)
	if err != nil {
		return err
	}
	// TODO 输入deployment进行检测
	if dep != "" {
		deploymentList := strings.Split(dep, " ")
		fmt.Println(deploymentList)
	} else {

		count := 0
		for true {
			needSend := true
			deploymentList, err := clientSet.AppsV1().Deployments(contNs).List(context.TODO(), metav1.ListOptions{})
			// 根据不同的类型 返回一个map
			//daemonSetsList, err := clientSet.AppsV1().DaemonSets(contNs).List(context.TODO(), metav1.ListOptions{})
			//list.Items[1].Name

			if err != nil {
			}
			items := deploymentList.Items
			for i := 0; i < len(items); i++ {
				status := items[i].Status
				if status.Replicas == status.AvailableReplicas {

				} else {
					if count > 1 {
						fmt.Println("服务: ", items[i].Name, "依然部署中...")
					}
					needSend = false
				}
			}

			if needSend {
				sendMessage()
				break
			}
			count += 1
			// 每个十秒检查一次
			fmt.Println("-----------------------")
			time.Sleep(10000 * time.Millisecond)

		}

	}
	return err
}

type ResMsg struct {
	StatusMessage string
	StatusCode    int
}

func sendMessage() {
	data := make(map[string]interface{})
	data["msg_type"] = "text"
	data["content"] = map[string]string{"text": msg}
	bytesData, _ := json.Marshal(data)

	c := resty.New()
	result := &ResMsg{}
	_, _ = c.R().SetResult(result).SetBody(bytes.NewBuffer([]byte(bytesData))).
		SetHeader("Accept", "application/json").Get(bot)

	if result.StatusCode != 0 {
		fmt.Printf("发送失败: %+v", result)
	}

	fmt.Println("执行完毕")
}
