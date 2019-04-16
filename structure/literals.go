package structure

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
)

type LiteralSection struct {
	Header          LiteralSectionHeader
	TreeDesc        HuffmanTreeDesc       `json:"-"`
	DecodingTable   *HuffmanDecodingTable `json:"-"`
	BytesUsedByTree int

	Data     []byte `json:"-"`
	dataRead int
}

type LiteralsBlockType byte

const (
	LiteralsBlockTypeRaw        = LiteralsBlockType(0)
	LiteralsBlockTypeRLE        = LiteralsBlockType(1)
	LiteralsBlockTypeCompressed = LiteralsBlockType(2)
	LiteralsBlockTypeTreeless   = LiteralsBlockType(3)
)

type LiteralSectionHeader struct {
	Type            LiteralsBlockType
	RegeneratedSize int
	CompressedSize  int
	NumberOfStreams int

	StreamSize1 uint16
	StreamSize2 uint16
	StreamSize3 uint16

	BytesUsedByHeader int
}

func (lsh *LiteralSectionHeader) DecodeJumpTable(raw []byte) error {
	if len(raw) < 6 {
		return errors.New("Not enough bytes for jumptable dacoding. Must be 6")
	}
	lsh.StreamSize1 = binary.LittleEndian.Uint16(raw[:2])
	lsh.StreamSize2 = binary.LittleEndian.Uint16(raw[2:4])
	lsh.StreamSize3 = binary.LittleEndian.Uint16(raw[4:6])

	if int(lsh.StreamSize1+lsh.StreamSize2+lsh.StreamSize3) > lsh.CompressedSize {
		return errors.New("Bad jump table")
	}
	return nil
}

func (lsh *LiteralSectionHeader) CalcStreamsize4() uint16 {
	return uint16(lsh.CompressedSize - int(lsh.StreamSize1+lsh.StreamSize2+lsh.StreamSize3))
}

var ErrIllegalLiteralSectionType = errors.New("Illegal LiteralSectionType. Must be between 0 to 3")
var ErrIllegalLiteralSectionSizeFormat = errors.New("Illegal LiteralSectionSizeformat. Must be between 0 to 3")

func (lsh *LiteralSectionHeader) DecodeType(raw byte) error {
	switch LiteralsBlockType(raw & 0x3) {
	case LiteralsBlockTypeRaw:
		lsh.Type = LiteralsBlockTypeRaw
	case LiteralsBlockTypeCompressed:
		lsh.Type = LiteralsBlockTypeCompressed
	case LiteralsBlockTypeRLE:
		lsh.Type = LiteralsBlockTypeRLE
	case LiteralsBlockTypeTreeless:
		lsh.Type = LiteralsBlockTypeTreeless
	default:
		return ErrIllegalLiteralSectionType
	}
	return nil
}

func (lsh *LiteralSectionHeader) DecodeSizes(raw []byte) error {
	//TODO check length of raw?
	sizeformat := (raw[0] >> 2) & 3
	if lsh.Type == LiteralsBlockTypeRaw || lsh.Type == LiteralsBlockTypeRLE {
		lsh.NumberOfStreams = 1

		switch sizeformat {
		case 0:
			fallthrough
		case 2:
			//sizeformat has a 0 as the last bit -> It actually only uses one bit
			//regenerated size uses 5 bits
			//need only one byte
			lsh.RegeneratedSize = int(raw[0] >> 3)
		case 1:
			//regenrated size uses 12 bits
			//need two bytes
			lsh.RegeneratedSize = int(raw[0]>>4) + int(uint(raw[1])<<4)
		case 3:
			//regenerated size uses 20 bits
			//need 3 bytes
			lsh.RegeneratedSize = int(raw[0]>>4) + int((uint32(raw[1])<<4)+(uint32(raw[2])<<12))
		default:
			return ErrIllegalLiteralSectionSizeFormat
		}

		//for convenience
		lsh.CompressedSize = lsh.RegeneratedSize
	} else {
		//normally there are 4 streams
		lsh.NumberOfStreams = 4
		var cpy [4]byte
		copy(cpy[:], raw)

		//the sizes encoded in the first 4 byte in the header which is enough for most cases. Where we need all 5 we use raw[4]
		// >> 4 just shifts out the flags at the beginning of the header
		sizes := (binary.LittleEndian.Uint32(cpy[:]) >> 4)

		switch sizeformat {
		case 0:
			lsh.NumberOfStreams = 1
			//difference is only in the number of streams
			fallthrough
		case 1:
			//regenerated and compressed size use 10 bits
			lsh.RegeneratedSize = int(sizes & 0x3FF)
			lsh.CompressedSize = int((sizes >> 10) & 0x3FF)

		case 2:
			//regenerated and compressed size use 14 bits
			lsh.RegeneratedSize = int(sizes & 0x3FFF)
			lsh.CompressedSize = int((sizes >> 14) & 0x3FFF)

		case 3:
			//regenerated and compressed size use 18 bits
			lsh.RegeneratedSize = int(sizes & 0x3FFFF)
			lsh.CompressedSize = int(((sizes >> 18) & 0x3FFFF) + (uint32(raw[4]) << 10))
		default:
			return ErrIllegalLiteralSectionSizeFormat
		}
	}
	return nil
}

//only first byte is needed to find out how many bytes the sizes need
func (lsh *LiteralSectionHeader) BytesNeededToDecodeSizes(raw byte) (int, error) {
	sizeformat := (raw >> 2) & 3
	if lsh.Type == LiteralsBlockTypeRaw || lsh.Type == LiteralsBlockTypeRLE {
		switch sizeformat {
		case 0:
			fallthrough
		case 2:
			//sizeformat has a 0 as the last bit -> It actually only uses one bit
			//regenerated size uses 5 bits
			//need only one byte
			return 1, nil
		case 1:
			//regenrated size uses 12 bits
			//need two bytes
			return 2, nil
		case 3:
			//regenerated size uses 20 bits
			//need 3 bytes
			return 3, nil
		default:
			return 0, ErrIllegalLiteralSectionSizeFormat
		}
	} else {
		switch sizeformat {
		case 0:
			//difference is only in the number of streams
			//TODO remember number of present streams somewhere
			fallthrough
		case 1:
			//regenerated and compressed size use 10 bits
			return 3, nil

		case 2:
			//regenerated and compressed size use 14 bits
			return 4, nil

		case 3:
			//regenerated and compressed size use 18 bits
			return 5, nil
		default:
			return 0, ErrIllegalLiteralSectionSizeFormat
		}
	}
}

func (ls *LiteralSection) DecodeNextLiteralsSection(source *bufio.Reader, prevBlock *Block) error {
	//read literals section
	var err error

	var headerbuffer [6]byte

	//read first byte
	headerbuffer[0], err = source.ReadByte()
	if err != nil {
		return err
	}
	ls.Header.BytesUsedByHeader = 1

	err = ls.Header.DecodeType(headerbuffer[0])
	if err != nil {
		return err
	}

	needed, err := ls.Header.BytesNeededToDecodeSizes(headerbuffer[0])
	if err != nil {
		return err
	}

	//read all needed bytes
	//n = 1 because the first byte still carries some of the actual size value
	for n := 1; n < needed; _ = n {
		x, err := source.Read(headerbuffer[n:needed])
		ls.Header.BytesUsedByHeader += x
		n += x
		if err != nil {
			return err
		}
	}

	if ls.Header.BytesUsedByHeader != needed {
		panic("This shouldnt happen if above code reads correct amounts of bytes")
	}

	err = ls.Header.DecodeSizes(headerbuffer[:needed])
	if err != nil {
		return err
	}

	//carry over old huffman tree if no new one is included
	if ls.Header.Type == LiteralsBlockTypeTreeless {
		ls.DecodingTable = prevBlock.Literals.DecodingTable
	}

	if ls.Header.Type == LiteralsBlockTypeCompressed {
		bytes, err := ls.TreeDesc.DecodeFromStream(source)
		ls.DecodingTable = ls.TreeDesc.Build()

		if err != nil {
			return err
		}
		ls.Header.CompressedSize -= bytes
		ls.BytesUsedByTree = bytes
	}

	//either == 1 or == 4
	if ls.Header.NumberOfStreams == 4 {
		// need to read jumptable --> 6 bytes
		needed := 6
		for n := 0; n < needed; _ = n {
			x, err := source.Read(headerbuffer[n:needed])
			ls.Header.BytesUsedByHeader += x
			n += x

			if err != nil {
				return err
			}
		}
		ls.Header.DecodeJumpTable(headerbuffer[0:6])

		ls.Header.CompressedSize -= 6
	}

	//read the data for this literals section
	needed = ls.Header.CompressedSize // compressed size ==regenerated size if not actually compressed
	ls.Data = make([]byte, needed)

	n, err := io.ReadFull(source, ls.Data)

	if err != nil {
		println("Error while reading literals section data")
		return err
	}

	if n != needed {
		panic("Didnt read enough bytes for the literalsection")
	}

	//decompress if necessary
	if ls.Header.Type == LiteralsBlockTypeCompressed || ls.Header.Type == LiteralsBlockTypeTreeless {
		if ls.Header.NumberOfStreams == 1 {
			output := make([]byte, ls.Header.RegeneratedSize)
			_, err := ls.DecodingTable.DecodeStream(ls.Data, output)
			if err != nil {
				return err
			}
			ls.Data = output
		} else {
			normalSize := (ls.Header.RegeneratedSize + 3) / 4
			//lastSize := ls.Header.RegeneratedSize - 3*normalSize
			output1 := make([]byte, normalSize)
			output2 := make([]byte, normalSize)
			output3 := make([]byte, normalSize)
			output4 := make([]byte, normalSize)

			low := 0
			high := int(ls.Header.StreamSize1)

			bytes1, err := ls.DecodingTable.DecodeStream(ls.Data[low:high], output1)
			if err != nil {
				return err
			}
			if bytes1 != (ls.Header.RegeneratedSize+3)/4 {
				panic("First stream didnt have right decoded length")
			}

			low += int(ls.Header.StreamSize1)
			high += int(ls.Header.StreamSize2)

			bytes2, err := ls.DecodingTable.DecodeStream(ls.Data[low:high], output2)
			if err != nil {
				return err
			}

			if bytes2 != (ls.Header.RegeneratedSize+3)/4 {
				panic("First stream didnt have right decoded length")
			}

			low += int(ls.Header.StreamSize2)
			high += int(ls.Header.StreamSize3)

			if int(high) > len(ls.Data) {
				panic("Corrupt stream sizes")
			}

			bytes3, err := ls.DecodingTable.DecodeStream(ls.Data[low:high], output3)
			if err != nil {
				return err
			}

			if bytes3 != (ls.Header.RegeneratedSize+3)/4 {
				panic("First stream didnt have right decoded length")
			}

			low += int(ls.Header.StreamSize3)
			high += int(ls.Header.CalcStreamsize4())

			if int(high) != ls.Header.CompressedSize {
				panic("Corrupt stream sizes")
			}

			bytes4, err := ls.DecodingTable.DecodeStream(ls.Data[low:high], output4)
			if err != nil {
				return err
			}

			if bytes1+bytes2+bytes3+bytes4 != ls.Header.RegeneratedSize {
				println("----")
				println(ls.Header.RegeneratedSize)
				println(bytes1 + bytes2 + bytes3 + bytes4)
				panic("Streams decoded combined didnt have correct length")
			}

			ls.Data = make([]byte, ls.Header.RegeneratedSize)
			copy(ls.Data, output1[:bytes1])
			copy(ls.Data[bytes1:], output2[:bytes2])
			copy(ls.Data[bytes1+bytes2:], output3[:bytes4])
			copy(ls.Data[bytes1+bytes2+bytes3:], output4[:bytes4])
		}
	}

	//TODO decode huffman if necessary
	return nil
}

func bitsToByte(bits int) int {
	x := bits / 8
	if bits%8 != 0 {
		x++
	}
	return x
}

func (ls *LiteralSection) Read(target []byte) (int, error) {
	//TODO just return subslices.... absolutely no need for this much copying
	//TODO decode huffman on the fly
	if ls.dataRead == len(ls.Data) {
		return 0, errors.New("OutOfLiterals")
	}
	end := ls.dataRead + len(target)
	if end > len(ls.Data) {
		end = len(ls.Data)
	}
	copy(target, ls.Data[ls.dataRead:end])
	diff := end - ls.dataRead
	ls.dataRead += diff
	return diff, nil
}

func (ls *LiteralSection) GetRest() []byte {
	return ls.Data[ls.dataRead:]
}
