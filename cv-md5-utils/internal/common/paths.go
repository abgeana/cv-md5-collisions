package common

import (
	"fmt"
	"github.com/golang/glog"
	"os"
	"path/filepath"
	"strings"
)

var color string
var nibble int

func rootPath() string {
	path, _ := os.Getwd()
	for path != "/" {
		if strings.HasSuffix(path, "cv-md5-collisions") {
			return path
		}
		path = filepath.Join(path, "..")
	}
	glog.Fatalf("could not find the root directory for this project")
	return ""
}

func PathSetColor(c string) {
	color = c
}

func PathSetNibble(n int) {
	nibble = n
}

func PathToOriginal(digit int) string {
	return filepath.Join(
		rootPath(),
		"collisions",
		color,
		"original",
		fmt.Sprintf("%x", digit),
	)
}

func PathToOriginalSegment(digit int, name string) string {
	return filepath.Join(
		PathToOriginal(digit),
		name,
	)
}

func PathToNibble(n int) string {
	return filepath.Join(
		rootPath(),
		"collisions",
		color,
		"collisions",
		fmt.Sprintf("nibble %02d", n),
	)
}

func PathToCurrentNibble() string {
	return PathToNibble(nibble)
}

func PathToPart(part int) string {
	return filepath.Join(
		PathToCurrentNibble(),
		fmt.Sprintf("part %02d", part),
	)
}

func PathToPDFPrefix() string {
	return filepath.Join(
		rootPath(),
		"collisions",
		color,
		"pdf prefixes",
		fmt.Sprintf("nibble %02d", nibble),
		"prefix.bin",
	)
}
