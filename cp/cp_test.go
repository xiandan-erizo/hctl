package cpFilePod2Pod

import (
	"log"
	"testing"
)

// go test -race -test.run TestCpPod2Pod  切到该目录执行该测试
func TestCpPod2Pod(t *testing.T) {
	log.Printf("开始测试")
	CpPod2Pod("", "", "")
}
