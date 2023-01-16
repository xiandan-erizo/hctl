/*
Copyright © 2022 xiandan HERE xiandan-erizo@outlook.com
*/
package cmd

import (
	"bytes"
	"fmt"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/remotecommand"
	"os"
	"path/filepath"
	"strings"
)

func CheckAndTransformFilePath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		path = filepath.Join(homeDir(), path[2:])
	}
	_, err := os.Stat(path) //os.Stat获取文件信息
	if err != nil {
		return "", err
	}
	return path, nil
}

func ExecCommandPod(podName string, cmdList []string) (string, string, error) {
	cmdExec := []string{"/bin/sh", "-c"}
	cmdExec = append(cmdExec, cmdList...)
	config, _ := clientcmd.LoadFromFile(cfgFile)
	currentContext := config.CurrentContext
	contNs := config.Contexts[currentContext].Namespace
	configN, err := clientcmd.BuildConfigFromFlags("", cfgFile)
	if err != nil {
		return "", "", err
	}
	clientSet, err := kubernetes.NewForConfig(configN)

	buf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	request := clientSet.CoreV1().RESTClient().
		Post().
		Namespace(contNs).
		Resource("pods").
		Name(podName).
		SubResource("exec").
		VersionedParams(&v1.PodExecOptions{
			Command: cmdExec,
			Stdin:   false,
			Stdout:  true,
			Stderr:  true,
			TTY:     true,
		}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(configN, "POST", request.URL())
	err = exec.Stream(remotecommand.StreamOptions{
		Stdout: buf,
		Stderr: errBuf,
	})

	fmt.Println(buf.String())

	fmt.Println(errBuf.String())

	return buf.String(), errBuf.String(), err
}
