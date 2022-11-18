package cmd

import (
	"fmt"
	"strings"
	"testing"
)

func Test_dump(t *testing.T) {
	str := "Dumping heap to /javatmp/tpp-rest-server-7b7ff8cc95-cpxfs-1918.hprof ...\nHeap dump file created\n"
	split := strings.FieldsFunc(str, func(r rune) bool {
		return strings.ContainsRune("\r\n", r)
	})
	//s := strings.Split(str, "\r\n")
	fmt.Print(split[0])

}
