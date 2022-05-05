package common

import (
	"github.com/golang/glog"
	"io/ioutil"
)

func ReadFile(path string) []byte {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		glog.Fatalf("could not read file \"%s\"", path)
	}
	return data
}

func WriteFile(path string, data []byte) {
	err := ioutil.WriteFile(path, data, 0644)
	if err != nil {
		glog.Fatalf("could not write file \"%s\"", path)
	}
}

func CopyFile(src, dst string) {
	input := ReadFile(src)
	WriteFile(dst, input)
}
