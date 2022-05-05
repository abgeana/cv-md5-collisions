package jfif

import (
	"fmt"
	"github.com/golang/glog"
)

type JfifSegment struct {
	Marker    byte
	Length    int
	Data      []byte
	ImageData []byte // ImageData contains the entropy-coded segment which comes immediately after an SOS segment.
}

const (
	/* For a good reference regarding segment markers also check these pages
	 * https://en.wikipedia.org/wiki/JPEG#Syntax_and_structure
	 * https://dev.exiv2.org/projects/exiv2/wiki/The_Metadata_in_JPEG_files
	 */
	// Start Of Image
	MARKER_SOI = 0xd8
	// Application0
	MARKER_APP0 = 0xe0
	// Define Quantization Table
	MARKER_DQT = 0xdb
	// Start Of Frame (Baseline DCT)
	MARKER_SOF0 = 0xc0
	// Define Huffman Table
	MARKER_DHT = 0xc4
	// Start Of Scan
	MARKER_SOS = 0xda
	// End Of Image
	MARKER_EOI = 0xd9
	// Comment
	MARKER_COM = 0xfe
)

func SegmentName(marker byte) string {
	switch marker {
	case MARKER_SOI:
		return "Start Of Image"
	case MARKER_APP0:
		return "Application0"
	case MARKER_DQT:
		return "Define Quantization Table"
	case MARKER_SOF0:
		return "Start Of Frame"
	case MARKER_DHT:
		return "Define Huffman Table"
	case MARKER_SOS:
		return "Start Of Scan"
	case MARKER_EOI:
		return "End Of Image"
	case MARKER_COM:
		return "Comment"
	default:
		return "unknown"
	}
}

func segmentLength(b1 byte, b2 byte) int {
	return (int(b1)<<8)&0xff00 + int(b2)
}

func identifySegment(data []byte, pos int) (*JfifSegment, int, error) {
	if data[pos] != 0xff {
		glog.Fatalf("segment at offset 0x%x does not start with 0xff", pos)
	}

	marker := data[pos+1]

	switch marker {

	case MARKER_SOI:
		fallthrough
	case MARKER_EOI:
		glog.Infof("identified %s segment\n", SegmentName(marker))
		return &JfifSegment{Marker: marker, Length: 2, Data: data[pos : pos+2]}, pos + 2, nil

	case MARKER_APP0:
		fallthrough
	case MARKER_DQT:
		fallthrough
	case MARKER_SOF0:
		fallthrough
	case MARKER_DHT:
		fallthrough
	case MARKER_SOS:
		fallthrough
	case MARKER_COM:
		dataLength := segmentLength(data[pos+2], data[pos+3]) + 2
		segment := JfifSegment{Marker: marker, Length: dataLength, Data: data[pos : pos+dataLength]}
		glog.Infof("identified %s segment of length 0x%x at offset 0x%x\n", SegmentName(marker), dataLength, pos)

		if marker != MARKER_SOS {
			return &segment, pos + dataLength, nil
		}

		pos += dataLength
		segment.ImageData = make([]byte, 0)
		stop := false
		for !stop {
			if data[pos] != 0xff {
				segment.ImageData = append(segment.ImageData, data[pos])
				pos += 1
			} else {
				if data[pos+1] == 0x00 { // compressed 0xff value
					segment.ImageData = append(segment.ImageData, data[pos])
					segment.ImageData = append(segment.ImageData, data[pos+1])
					pos += 2
				} else { // part of next marker
					stop = true
				}
			}
		}

		return &segment, pos, nil

	default:
		return nil, pos, fmt.Errorf("unknown segment marker %02x at offset 0x%x\n", marker, pos+1)
	}
}

func ParseJfif(data []byte) []*JfifSegment {
	if len(data) < 4 {
		glog.Fatal("file length is too small for a valid jpeg")
	}

	segments := make([]*JfifSegment, 0)
	segment, pos, err := identifySegment(data, 0)
	if err == nil {
		segments = append(segments, segment)
	} else {
		glog.Fatal(err.Error())
	}

	for pos < len(data) {
		segment, pos, err = identifySegment(data, pos)
		if err == nil {
			segments = append(segments, segment)
		} else {
			glog.Fatal(err.Error())
		}
	}

	return segments
}
