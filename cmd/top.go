package cmd

import (
	"context"
	"fmt"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/tools/clientcmd"
	metricsapi "k8s.io/metrics/pkg/apis/metrics"
	metricsv1beta1api "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

type TopComand struct {
	BaseCommand
}

var (
	SortBy             string
	NoHeaders          bool
	UseProtocolBuffers bool
	ShowCapacity       bool
	MetricsClient      metricsclientset.Interface
)

func (tc *TopComand) Init() {
	tc.command = &cobra.Command{
		Use:   "top",
		Short: "top node and pod",
		Long:  `top node and pod`,
		Run: func(cmd *cobra.Command, args []string) {
			err := tc.runTop(cmd, args)
			if err != nil {
				fmt.Println(err)
			}
		},
	}
	tc.command.Flags().StringVarP(&SortBy, "sort-by", "d", "", "If non-empty, sort nodes list using specified field. The field can be either 'cpu' or 'memory'.")
	tc.command.Flags().BoolVar(&NoHeaders, "no-headers", false, "If present, print output without headers")
	tc.command.Flags().BoolVar(&UseProtocolBuffers, "use-protocol-buffers", false, "Enables using protocol-buffers to access Metrics API.")
	tc.command.Flags().BoolVar(&ShowCapacity, "show-capacity", false, "Print node resources based on Capacity instead of Allocatable(default) of the nodes.")
}

func (tc *TopComand) runTop(cmd *cobra.Command, args []string) error {
	msg, _ = tc.command.Flags().GetString("msg")
	bot, _ = tc.command.Flags().GetString("bot")
	dep, _ = tc.command.Flags().GetString("dep")

	// 获取当前ns
	config, _ := clientcmd.LoadFromFile(cfgFile)
	currentContext := config.CurrentContext
	contNs := config.Contexts[currentContext].Namespace
	//configN, err := clientcmd.BuildConfigFromFlags("", cfgFile)
	//if err != nil {
	//	return err
	//}
	//clientSet, err := kubernetes.NewForConfig(configN)
	//if err != nil {
	//	return err
	//}
	//ToRESTConfig()
	//MetricsClient, err = metricsclientset.NewForConfig(rest * config(clientSet.RESTClient()))
	//MetricsClient = metricsClientset
	api, err := getMetricsFromMetricsAPI(MetricsClient, contNs, "", false, nil, nil)
	fmt.Println(api)
	return err
}

func getMetricsFromMetricsAPI(metricsClient metricsclientset.Interface, namespace, resourceName string, allNamespaces bool, labelSelector labels.Selector, fieldSelector fields.Selector) (*metricsapi.PodMetricsList, error) {
	var err error
	ns := metav1.NamespaceAll
	if !allNamespaces {
		ns = namespace
	}
	versionedMetrics := &metricsv1beta1api.PodMetricsList{}
	if resourceName != "" {
		m, err := metricsClient.MetricsV1beta1().PodMetricses(ns).Get(context.TODO(), resourceName, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		versionedMetrics.Items = []metricsv1beta1api.PodMetrics{*m}
	} else {
		versionedMetrics, err = metricsClient.MetricsV1beta1().PodMetricses(ns).List(context.TODO(), metav1.ListOptions{LabelSelector: labelSelector.String(), FieldSelector: fieldSelector.String()})
		if err != nil {
			return nil, err
		}
	}
	metrics := &metricsapi.PodMetricsList{}
	err = metricsv1beta1api.Convert_v1beta1_PodMetricsList_To_metrics_PodMetricsList(versionedMetrics, metrics, nil)
	if err != nil {
		return nil, err
	}
	return metrics, nil
}
