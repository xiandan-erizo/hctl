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
	"github.com/gosuri/uilive"
	kruiseclientset "github.com/openkruise/kruise-api/client/clientset/versioned"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/clientcmd/api"
	"strings"
	"time"
)

var (
	rstype string
	rslist string
	contNs string
	client discovery.DiscoveryInterface
)

var rstypeMap = map[string]string{
	"dep":        "deployment",
	"deployment": "deployment",
	"clo":        "cloneset",
	"cloneset":   "cloneset",
}

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

	sc.command.Flags().StringVarP(&rstype, "rstype", "t", "", "控制器类型")
	sc.command.Flags().StringVarP(&rslist, "rslist", "l", "", "具体名称列表,空格分割")
}

func (sc *StatsComand) runStats(cmd *cobra.Command, args []string) error {
	msg, _ = sc.command.Flags().GetString("msg")
	bot, _ = sc.command.Flags().GetString("bot")
	rstype, _ = sc.command.Flags().GetString("rstype")
	rslist, _ = sc.command.Flags().GetString("rslist")
	// 获取当前ns
	config, _ := clientcmd.LoadFromFile(cfgFile)

	// TODO 输入deployment进行检测
	rstypeResult, ok := rstypeMap[rstype]
	if !ok {
		if rstype == "" {
			rstypeResult = "deployment"
		} else {
			panic(fmt.Sprintf("不支持的类型: %s,目前仅支持:%s", rstype, rstypeMap))
		}

	}

	// 开始检查
	// 发送信息
	count := 0
	writer := uilive.New()
	writer.Start()
	check(config, count, rstypeResult, writer)
	sendMessage()
	return nil
}

func check(config *api.Config, count int, rstypeResult string, writer *uilive.Writer) {
	needSend := true
	var serviceList []string
	if rstypeResult == "deployment" {
		contNs, clientSet := getClient(config, rstypeResult)
		k8sClientSet, _ := clientSet.(*kubernetes.Clientset)
		deploymentList, _ := k8sClientSet.AppsV1().Deployments(contNs).List(context.TODO(), metav1.ListOptions{})

		for _, item := range deploymentList.Items {
			status := item.Status
			if status.Replicas == status.AvailableReplicas {
				// 所有副本都可用，无需发送消息
			} else {
				// 存在不可用的副本，需要发送消息
				serviceList = append(serviceList, item.Name+"\r\n")
				needSend = false
			}
		}
	} else {
		contNs, clientSet := getClient(config, rstypeResult)
		kruiseclient, _ := clientSet.(*kruiseclientset.Clientset)
		clonesetList, _ := kruiseclient.AppsV1alpha1().CloneSets(contNs).List(context.TODO(), metav1.ListOptions{})
		for _, item := range clonesetList.Items {
			status_replicas := item.Status.Replicas
			replicas := item.Spec.Replicas
			if status_replicas == *replicas {
				// 所有副本都可用，无需发送消息
			} else {
				// 存在不可用的副本，需要发送消息
				serviceList = append(serviceList, item.Name+"\r\n")
				needSend = false
			}
		}
	}

	if needSend {
		return
	}
	count++
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02 15:04:05")
	// 每个十秒检查一次
	_, _ = fmt.Fprintf(writer, "%s\r\n依然部署中...\r\n%s ", formattedTime, strings.Join(serviceList, ""))
	err := writer.Flush()
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(10 * time.Second)
	check(config, count, rstypeResult, writer)
}

func getClient(config *api.Config, restype string) (string, discovery.DiscoveryInterface) {
	if client != nil {
		return contNs, client
	}
	currentContext := config.CurrentContext
	contNs := config.Contexts[currentContext].Namespace

	configN, err := clientcmd.BuildConfigFromFlags("", cfgFile)
	if err != nil {
		// Handle error
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
