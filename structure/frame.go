package structure

import (
	"encoding/binary"
	"errors"
)

type Frame struct {
	MagicNumber [4]byte //always present as magic value
	Header      FrameHeader
	Checksum    []byte //optional
}

//Can be 2 to 14 bytes in encoded form
type FrameHeader struct {
	Descriptor       FrameDescriptor
	WindowSize       uint64
	DictionaryID     uint64
	FrameContentSize uint64
}

//DecodeFrameDescriptor just for completeness and not requiring the user to actually set any values in the frameheader directly
func (fh *FrameHeader) DecodeFrameDescriptor(raw byte) {
	fh.Descriptor = FrameDescriptor(raw)
}

//DecodeWindowSize calculates the windows size with the formula from the documentation from the byte named WindowDescriptor
func (fh *FrameHeader) DecodeWindowSize(raw byte) {
	//TODO check for overflows?
	exp := raw >> 3
	mantissa := uint64(raw & 0x7)
	windowLog := 10 + exp
	windowBase := uint64(1) << windowLog
	windowAdd := (windowBase / 8) * mantissa
	fh.WindowSize = windowBase + windowAdd
}

func (fh *FrameHeader) DecodeDictionaryID(raw []byte) {
	//TODO check values?
	if len(raw) == 8 {
		fh.DictionaryID = binary.LittleEndian.Uint64(raw)
	} else {
		var buf [8]byte
		copy(buf[:], raw)
		fh.DictionaryID = binary.LittleEndian.Uint64(buf[:])
	}
}

func (fh *FrameHeader) DecodeFrameContentSize(raw []byte) {
	//TODO check values?
	if len(raw) == 8 {
		fh.FrameContentSize = binary.LittleEndian.Uint64(raw)
	} else {
		var buf [8]byte
		copy(buf[:], raw)
		fh.FrameContentSize = binary.LittleEndian.Uint64(buf[:])
	}
	if size, _ := fh.Descriptor.GetContentSizeFlag(); size == 2 {
		fh.FrameContentSize += 256 //dont ask thats what the documentation says.
	}
}

//SetWindowSize calculates the windows size with the formula from the documentation from the byte named WindowDescriptor
func (fh *FrameHeader) SetWindowSize(raw byte) {
	//TODO check for overflows?
	exp := raw >> 3
	mantissa := uint64(raw & 0x7)
	windowLog := 10 + exp
	windowBase := uint64(1) << windowLog
	windowAdd := (windowBase / 8) * mantissa
	fh.WindowSize = windowBase + windowAdd
}

type FrameDescriptor byte

var ErrIllegalContentSizeFlag = errors.New("The SizeFlag for the Field ContentSize has an illegal value bigger than 3")

//GetContentSizeFlag returns the number of bytes the Field FrameContentSize uses
func (fd *FrameDescriptor) GetContentSizeFlag() (byte, error) {
	flagvalue := *fd >> 6 //two most sgnificant bits
	switch flagvalue {
	case 0:
		//See documentation why this is a thing
		if fd.GetSingleSegmentFlag() {
			return 1, nil
		} else {
			return 0, nil
		}
	case 1:
		return 2, nil
	case 2:
		return 4, nil
	case 3:
		return 8, nil
	default:
		return 0, ErrIllegalContentSizeFlag
	}
}

//GetSingleSegmentFlag extracts whether this is a single segment frame
func (fd *FrameDescriptor) GetSingleSegmentFlag() bool {
	return (*fd>>5)&0x1 == 1 //shift 5 to the right and mask all other bits besides the lowest one
}

//GetContentChecksumFlag extracts whether a checksum is in the header
func (fd *FrameDescriptor) GetContentChecksumFlag() bool {
	return (*fd>>2)&0x1 == 1 //shift 2 to the right and mask all other bits besides the lowest one
}

var ErrIllegalDictionaryIDFlag = errors.New("The SizeFlag for the Field DictionaryID has an illegal value bigger than 3")

//GetDictionaryFlag returns the number of bytes the Field DictionaryID uses
func (fd *FrameDescriptor) GetDictionaryFlag() (byte, error) {
	flagvalue := *fd & 0x3 //two least sgnificant bits
	switch flagvalue {
	case 0:
		return 0, nil
	case 1:
		return 1, nil
	case 2:
		return 2, nil
	case 3:
		return 4, nil
	default:
		return 0, ErrIllegalContentSizeFlag
	}
}
