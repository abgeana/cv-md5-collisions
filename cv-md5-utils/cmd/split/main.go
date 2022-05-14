package main

import (
	"cv-md5-utils/internal/common"
	"cv-md5-utils/internal/jfif"
	"flag"
	"fmt"
)

func main() {
	var flagPath string

	flag.StringVar(&flagPath, "path", "", "")
	flag.Parse()

	data := common.ReadFile(flagPath)

	for idx, segment := range jfif.ParseJfif(data) {
		fileName := fmt.Sprintf("%02x %s", idx, segment.Name())
		common.WriteFile(fileName, segment.Data)
		if segment.Marker == jfif.MARKER_SOS {
			fileName = fmt.Sprintf("%02x %s (entropy-coded data)", idx, segment.Name())
			common.WriteFile(fileName, segment.ImageData)
		}
	}
}
