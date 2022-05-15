package main

import (
	"cv-md5-utils/internal/common"
	"flag"
	"fmt"
	"github.com/golang/glog"
	"os"
	"path/filepath"
)

func main() {
	var flagColor string
	var flagNibble int
	flag.StringVar(&flagColor, "color", "white on blue", "")
	flag.IntVar(&flagNibble, "nibble", 1, "")

	common.PathSetColor(flagColor)
	common.PathSetNibble(flagNibble)

	jpegsDir := filepath.Join(
		common.PathToCurrentNibble(),
		"jpegs",
	)
	os.Mkdir(jpegsDir, 0775)

	/* get the size of the initial PDF prefix file
	 * which must be discarded from various part files to create
	 * single JPEG files (i.e. without the PDF prefix)
	 */
	pdfPrefixStat, err := os.Stat(common.PathToPDFPrefix())
	if err != nil {
		glog.Fatal("could not get the size of the PDF prefix file")
	}
	pdfPrefixSize := pdfPrefixStat.Size()

	// create the last JPEG for hex digit f from "part 16/03 final"
	finalData := common.ReadFile(filepath.Join(
		common.PathToPart(16),
		"03 final",
	))
	fJpeg := finalData[pdfPrefixSize:]
	common.WriteFile(
		filepath.Join(jpegsDir, "0.jpeg"),
		fJpeg,
	)

	// create the first 15 JPEGs for hex digits 0 to f
	for i := 1; i <= 15; i++ {
		/* read the jfif file with the long comment
		 * in this file, the comment generated via the "poc_no.sh" file is 0x200 bytes long
		 * and the JFIF segments are interpreted
		 */
		jpeg := common.ReadFile(filepath.Join(
			common.PathToPart(i),
			"08 jfif long",
		))
		// discard the PDF prefix
		jpeg = jpeg[pdfPrefixSize:]
		/* append data from "part 16/03 final" required for the collision chaining
		 * this data is not interpreted anymore since the previously read "08 jfif long"
		 * ends with a "End of Image" JFIF segment
		 */
		jpeg = append(jpeg, fJpeg[len(jpeg):]...)
		common.WriteFile(
			filepath.Join(
				jpegsDir,
				fmt.Sprintf("%x.jpeg", i-1),
			),
			jpeg,
		)
	}
}
