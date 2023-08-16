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
	"github.com/spf13/cobra"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	crdClientset "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"sort"
	"strings"
	"time"
)

var (
	rstype string
	rslist string
	contNs string
	client discovery.DiscoveryInterface
)

const (
	deployment = "deployment"
	cloneset   = "cloneset"
	knative    = "knative todo"
)

const (
	clonesetCrd = "clonesets.apps.kruise.io"
	knativeCrd  = ""
)

var rstypeMap = map[string]string{
	"dep":        deployment,
	"deployment": deployment,
	"clo":        cloneset,
	"cloneset":   cloneset,
	"ksvc":       knative,
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

	sc.command.Flags().StringVarP(&rstype, "rstype", "t", "deployment", "控制器类型")
	sc.command.Flags().StringVarP(&rslist, "rslist", "l", "", "具体名称列表,空格分割")

}

func (sc *StatsComand) runStats(cmd *cobra.Command, args []string) error {
	msg, _ = sc.command.Flags().GetString("msg")
	bot, _ = sc.command.Flags().GetString("bot")
	rstype, _ = sc.command.Flags().GetString("rstype")
	rslist, _ = sc.command.Flags().GetString("rslist")

	// 处理rslist
	var rs_split []string
	if rslist == "" {
		rs_split = []string{}
	} else {
		rs_split = strings.Split(rslist, ",")
	}

	// 获取当前ns
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
	//count := 0
	writer := uilive.New()
	writer.Start()
	checkNew(cfgFile, rstypeResult, writer, rs_split, namespace)
	return nil
}

func checkNew(config string, rstypeResult string, writer *uilive.Writer, rsSplit []string, n string) {
	if rstypeResult == deployment {
		contNs, clientSet, _, _ := getClient(config, rstypeResult, namespace)
		k8sClientSet, _ := clientSet.(*kubernetes.Clientset)
		checkDeployment(k8sClientSet, contNs, writer, rsSplit)
	} else if rstypeResult == cloneset {
		contNs, _, clientSet, dynamicClient := getClient(config, rstypeResult, namespace)
		crd := getCrd(clientSet, clonesetCrd)
		checkCloneset(crd, dynamicClient, contNs, writer, rsSplit)
	} else if rstypeResult == knative {

	}

}

func getCrd(crdClientset *crdClientset.Clientset, crdName string) *v1.CustomResourceDefinition {
	crd, err := crdClientset.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), crdName, metav1.GetOptions{})
	if err != nil {
		panic(err)
	}
	return crd
}

// 检查deployment的状态
func checkDeployment(k8sClientSet *kubernetes.Clientset, contNs string, writer *uilive.Writer, rsSplit []string) {
	deploymentList, _ := k8sClientSet.AppsV1().Deployments(contNs).List(context.TODO(), metav1.ListOptions{})
	// todo 改成使用watch机制
	sort.Strings(rsSplit)
	var unavaliable []string
	for _, dep := range deploymentList.Items {
		if len(rsSplit) > 0 {
			index := sort.SearchStrings(rsSplit, dep.Name)
			if index < len(rsSplit) && rsSplit[index] == dep.Name {
				if dep.Status.AvailableReplicas == *dep.Spec.Replicas {
					continue
				} else {
					unavaliable = append(unavaliable, dep.Name)
				}
			}
		} else {
			if dep.Status.AvailableReplicas == *dep.Spec.Replicas {
				continue
			} else {
				unavaliable = append(unavaliable, dep.Name)
			}
		}

	}
	if len(unavaliable) == 0 {
		sendMessage()
		return
	}
	printMessage(writer, unavaliable)
	checkDeployment(k8sClientSet, contNs, writer, unavaliable)
}

// cloneset状态
func checkCloneset(crd *v1.CustomResourceDefinition, dynamicClient *dynamic.DynamicClient, contNs string, writer *uilive.Writer, rsSplit []string) {
	clonesetList, err := dynamicClient.Resource(schema.GroupVersionResource{
		Group:    crd.Spec.Group,
		Version:  crd.Spec.Versions[0].Name,
		Resource: strings.ToLower(crd.Spec.Names.Plural),
	}).Namespace(contNs).List(context.TODO(), metav1.ListOptions{})

	if err != nil {
		panic(err)
	}

	var unavaliable []string
	for _, clo := range clonesetList.Items {
		status := clo.Object["status"].(map[string]interface{})
		if len(rsSplit) > 0 {
			index := sort.SearchStrings(rsSplit, clo.GetName())
			if index < len(rsSplit) && rsSplit[index] == clo.GetName() {
				if status["availableReplicas"] == status["replicas"] {
					continue
				} else {
					unavaliable = append(unavaliable, clo.GetName())
				}
			}
		} else {
			if status["availableReplicas"] == status["replicas"] {
				continue
			} else {
				unavaliable = append(unavaliable, clo.GetName())
			}
		}

	}
	if len(unavaliable) == 0 {
		sendMessage()
		return
	}
	printMessage(writer, unavaliable)
	checkCloneset(crd, dynamicClient, contNs, writer, unavaliable)

}

// knative状态
func checkKnative() {

}
func printMessage(writer *uilive.Writer, serviceList []string) {
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02 15:04:05")
	// 每个十秒检查一次
	_, _ = fmt.Fprintf(writer, "%s\r\n依然部署中...\r\n%s ", formattedTime, strings.Join(serviceList, ""))
	err := writer.Flush()
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(10 * time.Second)
}

type ResMsg struct {
	StatusMessage string
	StatusCode    int
}

func sendMessage() {
	data := make(map[string]interface{})
	data["msg_type"] = "text"

	if msg == "" {
		data["content"] = map[string]string{"text": hctlconfig.Feishu.Msg}
	} else {
		data["content"] = map[string]string{"text": msg}
	}

	bytesData, _ := json.Marshal(data)

	if bot == "" {
		bot = hctlconfig.Feishu.Url
	}

	c := resty.New()
	result := &ResMsg{}
	_, _ = c.R().SetResult(result).SetBody(bytes.NewBuffer([]byte(bytesData))).
		SetHeader("Accept", "application/json").Get(bot)

	if result.StatusCode != 0 {
		fmt.Printf("发送失败: %+v", result)
	}

	fmt.Println("执行完毕")
}
