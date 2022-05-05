package jfif

import (
	"fmt"
	"github.com/golang/glog"
)

type JfifSegment struct {
	Marker    byte
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

// Length returns the 16 bit integer at positions 2 and 3 (0 based indices). This integer denotes 2 plus the amount of
// data following the first 4 bytes.
func (s *JfifSegment) Length() int {
	if len(s.Data) < 4 {
		glog.Fatal("not enough data in this segment to parse the two length bytes")
	}

	b1 := s.Data[2]
	b2 := s.Data[3]

	return (int(b1)<<8)&0xff00 + int(b2)
}

func parseSegment(bytes []byte, pos int) (*JfifSegment, error) {
	if bytes[pos] != 0xff {
		glog.Fatalf("segment at offset 0x%x does not start with 0xff", pos)
	}

	marker := bytes[pos+1]

	switch marker {

	// segments with length 2 (i.e. only the 0xff byte and the marker)
	case MARKER_SOI:
		fallthrough
	case MARKER_EOI:
		glog.Infof("identified %s segment\n", SegmentName(marker))
		data := make([]byte, 2)
		copy(data, bytes[pos:])
		return &JfifSegment{
			Marker:    marker,
			Data:      data,
			ImageData: make([]byte, 0),
		}, nil

	// segments with length greater than 2
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
		data := make([]byte, 4)
		copy(data, bytes[pos:])

		segment := JfifSegment{
			Marker:    marker,
			Data:      data,
			ImageData: make([]byte, 0),
		}

		segment.Data = append(segment.Data, bytes[pos+len(segment.Data):pos+segment.Length()+2]...)
		glog.Infof("identified %s segment of length 0x%x at offset 0x%x\n", SegmentName(marker), len(segment.Data), pos)

		// if this is not a Start of Scan segment, then the job is finished
		if segment.Marker != MARKER_SOS {
			return &segment, nil
		}

		// if this is a Start of Scan segment, then additional bytes are appended to the segment data
		// keep appending bytes to ImageData while
		//  * byte value is different than 0xff
		//  * byte value is 0xff and the following byte is 0x00
		pos += len(segment.Data)
		stop := false
		for !stop {
			if bytes[pos] != 0xff {
				segment.ImageData = append(segment.ImageData, bytes[pos])
				pos += 1
			} else {
				if bytes[pos+1] == 0x00 { // compressed 0xff value
					segment.ImageData = append(segment.ImageData, bytes[pos])
					segment.ImageData = append(segment.ImageData, bytes[pos+1])
					pos += 2
				} else { // part of next marker
					stop = true
				}
			}
		}

		return &segment, nil

	default:
		return nil, fmt.Errorf("unknown segment marker %02x at offset 0x%x\n", marker, pos+1)
	}
}

func ParseJfif(data []byte) []*JfifSegment {
	if len(data) < 4 {
		glog.Fatal("file length is too small for a valid jpeg")
	}

	segments := make([]*JfifSegment, 0)
	pos := 0

	for pos < len(data) {
		if segment, err := parseSegment(data, pos); err == nil {
			pos += len(segment.Data) + len(segment.ImageData)
			segments = append(segments, segment)
		} else {
			glog.Fatal(err.Error())
		}
	}

	return segments
}
