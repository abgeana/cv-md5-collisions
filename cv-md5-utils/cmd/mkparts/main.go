package main

import (
	"cv-md5-utils/internal/common"
	"encoding/binary"
	"flag"
	"github.com/golang/glog"
	"io/ioutil"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

var flagNibble int
var flagPartFrom int
var flagPartTo int
var flagColor string

func runPocNo(prefixFile string) error {
	glog.Info("executing poc_no.sh")
	defer glog.Info("poc_no.sh finished")

	/* copy the file to simply "prefix"
	 * needed becasue poc_no.sh does not play well with spaces in path names
	 */
	common.CopyFile(prefixFile, "prefix")
	defer os.RemoveAll("prefix")

	cmd := exec.Command("poc_no.sh", "prefix")
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// time out after 15 minutes if poc_no.sh has not finished
	time.AfterFunc(15*time.Minute, func() {
		syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
	})

	return cmd.Run()
}

func generateJfifSegments(segmentsPath, fileName string) []byte {
	files, err := ioutil.ReadDir(segmentsPath)
	if err != nil {
		glog.Fatalf("could not read directory with jfif segments: %s", segmentsPath)
	}

	ignoredFiles := []string{
		// names of the original image files
		"image.jpeg",
		"image.xcf",
		// names of the common segments not used by the
		"00 Start Of Image",
		"01 Application0",
	}

	segmentsData := make([]byte, 0)
	for _, file := range files {
		isIgnored := false
		for _, ignored := range ignoredFiles {
			if file.Name() == ignored {
				isIgnored = true
			}
		}

		if !isIgnored {
			segmentsData = append(
				segmentsData,
				common.ReadFile(filepath.Join(segmentsPath, file.Name()))...,
			)
		}
	}

	common.WriteFile(fileName, segmentsData)
	return segmentsData
}

func mkPart(part int) error {
	workdir := common.PathToPart(part)
	os.Mkdir(workdir, 0775)

	// change the working directory when making the current part
	cwd, _ := os.Getwd()
	os.Chdir(workdir)
	defer os.Chdir(cwd)

	if part == 1 {
		// the input for the first part is the pdf prefix and common jfif sections
		// the common jfif sections are SOI and APP0
		common.CatToFile(
			"01 starting prefix",
			common.PathToPDFPrefix(),
			common.PathToOriginalSegment(0, "00 Start Of Image"),
			common.PathToOriginalSegment(0, "01 Application0"),
		)
	} else {
		// the input for subsequent parts is "08 jfif short" from the previous part
		common.CopyFile(
			filepath.Join(
				common.PathToPart(part-1),
				"08 jfif short",
			),
			"01 starting prefix",
		)
	}

	if part < 16 {
		// read the "01 starting prefix" file
		startingPrefix := common.ReadFile("01 starting prefix")

		// calculate the size of the comment needed to pad to length 7 mod 64
		startingPrefixLen := len(startingPrefix)
		commentSectionLen := int(math.Ceil(float64(startingPrefixLen)/64)*64+7) - startingPrefixLen
		if commentSectionLen < 4 {
			// we need to add at least 4 characters, if this is not enough then add one more 64 byte block in the comment section
			commentSectionLen = int(math.Ceil(float64(startingPrefixLen+64)/64)*64+7) - startingPrefixLen
		}

		/* create the "02 comment" file
		 * this file contains one the comment section which pads the "01 starting prefix" file to length 7 mod 64
		 */
		commentData := make([]byte, commentSectionLen)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(commentSectionLen-2))
		copy(commentData[0:2], []byte{0xff, 0xfe})
		copy(commentData[2:4], lengthBytes)
		common.WriteFile("02 comment", commentData)

		/* create the "03 collision prefix" file
				 * this is the combination of "01 starting prefix" + "02 comment" + "ff fe 01 00 00"
		         * the last part brings the length of the prefix to 12 mod 64, which after trial and
		         * error is the sweet spot such that the poc_no.sh script does not alter the prefix
		*/
		collisionPrefix := common.ReadFile("01 starting prefix")
		collisionPrefix = append(collisionPrefix, commentData...)
		collisionPrefix = append(collisionPrefix, []byte{0xff, 0xfe, 0x01, 0x00, 0x00}...)
		common.WriteFile("03 collision prefix", collisionPrefix)

		// run the "poc_no.sh" script
		err := runPocNo("03 collision prefix")

		// check whether the script exited cleanly
		if err != nil {
			// if not, then bail out
			return err
		}

		// rename collision1 and collision2
		// collision1 has the comment length set to 0x100 (the short comment)
		// collision1 has the comment length set to 0x200 (the long comment)
		os.Rename("collision1.bin", "04 collision1 short com")
		os.Rename("collision2.bin", "04 collision2 long com")

		// clean up files generated by "poc_no.sh"
		os.RemoveAll("data")
		os.RemoveAll("logs")
		os.RemoveAll("prefix")
		os.RemoveAll("upper_1_640000")

		// read the collision binaries generated via `poc_no.sh`
		collision1Data := common.ReadFile("04 collision1 short com")
		collision2Data := common.ReadFile("04 collision2 long com")

		/* pad both files until the short comment ends
		 * this data is not interpreted in either of the collision files
		 * the amount of padding is
		 * 137 = 0x100 (length of segment data) - 0x79 (amount of bytes in segment with collision data) + 2
		 */
		padding := make([]byte, 137)
		collision1Data = append(collision1Data, padding...)
		collision2Data = append(collision2Data, padding...)
		common.WriteFile("05 collision1 short com fill", collision1Data)
		common.WriteFile("05 collision2 long com fill", collision2Data)

		// create the "06 jfif segments" file
		segmentsData := generateJfifSegments(
			common.PathToOriginal(part-1),
			"06 jfif segments",
		)

		/* create the "07 comment" file
		 * this is the comment which covers all the jfif segments
		 * in the "short com" file, this comment is parsed and thus the parser skips all segments
		 * in the "long com" file, this comment is not parsed, since it is in the 2nd half of the 0x200 bytes comment (generated by poc_no.sh)
		 */
		commentData = make([]byte, 0x100)
		lengthBytes = make([]byte, 2)
		/* the 2 bytes segment length, not the complete data length
		 * length of segmentsData + 0x100 bytes of the actual comment segment data - 2 bytes for FF FE
		 */
		commentSegmentLen := len(segmentsData) + 0x100 - 2
		binary.BigEndian.PutUint16(lengthBytes, uint16(commentSegmentLen))
		copy(commentData[0:2], []byte{0xff, 0xfe})
		copy(commentData[2:4], lengthBytes)
		common.WriteFile("07 comment", commentData)

		// create the "08 jfif short" and "08 jfif long" files
		jfifShortData := append(collision1Data, commentData...)
		jfifShortData = append(jfifShortData, segmentsData...)
		common.WriteFile("08 jfif short", jfifShortData)
		jfifLongData := append(collision2Data, commentData...)
		jfifLongData = append(jfifLongData, segmentsData...)
		common.WriteFile("08 jfif long", jfifLongData)
	} else {
		/* no collisions are needed for the final part
		 * for this part, we only add all the segments to the starting prefix
		 */
		startingPrefix := common.ReadFile("01 starting prefix")
		segmentsData := generateJfifSegments(
			common.PathToOriginal(part-1),
			"02 jfif segments",
		)
		common.WriteFile(
			"03 final",
			append(startingPrefix, segmentsData...),
		)
	}

	return nil
}

func main() {
	flag.IntVar(&flagNibble, "nibble", 1, "")
	flag.IntVar(&flagPartFrom, "part-from", 1, "")
	flag.IntVar(&flagPartTo, "part-to", 16, "")
	flag.StringVar(&flagColor, "color", "white on blue", "")
	flag.Parse()

	common.PathSetColor(flagColor)
	common.PathSetNibble(flagNibble)

	_, err := os.Stat(common.PathToCurrentNibble())
	if err != nil {
		// this most probably means that the nibble directory does not exist
		os.Mkdir(common.PathToCurrentNibble(), 0775)
	}

	for part := flagPartFrom; part <= flagPartTo; part++ {
		glog.Infof("making nibble %02d, part %x\n", flagNibble, part)
		done := false
		for done == false {
			if err := mkPart(part); err != nil {
				glog.Info(err.Error())
				os.RemoveAll(common.PathToPart(part))
			} else {
				done = true
			}
		}
	}
}
