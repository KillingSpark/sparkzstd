package decompression

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"github.com/killingspark/sparkzstd/structure"
	"io"
	"strconv"
)

//FrameDecompressor is the Struct that holds all info and funcs for decompressing a zstd frame
type FrameDecompressor struct {
	frame  structure.Frame
	source *bufio.Reader
	target io.Writer

	//will be limited to the CurrentBlocks size and given to the decoding functions
	limitedSource *io.LimitedReader

	decodebuffer  *Ringbuffer //must be at least frame.Header.WindowSize long. Will be used in decoding the CurrentBlock
	offsetHistory [3]int64

	//used in sequence execution
	literalsCopyBuf [128 * 1024]byte //can only be max 128kb. Allocate here once instead of for every block

	//used while decoding/decompressing literals and sequences sections
	literalsDataBuf           [128 * 1024]byte //can only be max 128kb. Allocate here once instead of for every block
	literalsCompressedDataBuf [128 * 1024]byte //can only be max 128kb. Allocate here once instead of for every block
	sequencesDataBuf          [128 * 1024]byte //can only be max 128kb. Allocate here once instead of for every block

	CurrentBlock  structure.Block
	PreviousBlock structure.Block
	BlockCounter  int

	headerbuffer [14]byte //just used to temporarly hold frameheader or blockheader data while decoding these

	Verbose bool
}

func (fd *FrameDecompressor) Reset(newsource io.Reader, newtarget io.Writer) {
	fd.source = bufio.NewReader(newsource)
	fd.target = newtarget

	fd.frame = structure.Frame{}
	fd.limitedSource = nil
	fd.offsetHistory = [3]int64{1, 4, 8}
	fd.CurrentBlock = structure.Block{}
	fd.PreviousBlock = structure.Block{}
	fd.BlockCounter = 0
}

//NewFrameDecompressor makes a new FrameDecompressor that reads compressed data from s and writes decompressed data to t
func NewFrameDecompressor(s io.Reader, t io.Writer) *FrameDecompressor {
	return &FrameDecompressor{
		source:        bufio.NewReader(s),
		target:        t,
		offsetHistory: [3]int64{1, 4, 8},
	}
}

func (fd *FrameDecompressor) printStatus() error {
	println("##############################")
	println("Frame Decoder Status")
	println("##############################")

	fheader, err := json.MarshalIndent(fd.frame.Header, "\t", "   ")
	bheader, err := json.MarshalIndent(fd.CurrentBlock, "\t", "   ")

	status := "FrameHeader: \n\t" + string(fheader)
	status += "\n\nSingleSegment: " + strconv.FormatBool(fd.frame.Header.Descriptor.GetSingleSegmentFlag())
	status += "\n\nDecodebuffer Size: " + strconv.Itoa(fd.decodebuffer.Len)
	status += "\n\nCurrentBlock: \n\t" + string(bheader)
	status += "\n"

	if err != nil {
		return err
	}

	println(status)

	println("##############################")
	println("Frame Decoder Status Finished")
	println("##############################")

	return err
}

var ErrCorruptSizes = errors.New("The sizes of literal and sequence section did not add up to blocksize")

//DecodeNextBlockContent decodes the literal and sequence section of the current block
func (fd *FrameDecompressor) DecodeNextBlockContent() error {
	bufsrc := bufio.NewReader(fd.limitedSource)
	err := fd.CurrentBlock.Literals.DecodeNextLiteralsSection(bufsrc, &fd.PreviousBlock)
	if fd.Verbose {
		fd.printCurrentBlockLiterals()
	}
	if err != nil {
		return err
	}

	bytesUsedByLiterals := uint64(fd.CurrentBlock.Literals.Header.CompressedSize + fd.CurrentBlock.Literals.Header.BytesUsedByHeader + fd.CurrentBlock.Literals.BytesUsedByTree)
	bytesLeft := fd.CurrentBlock.Header.BlockSize - bytesUsedByLiterals

	err = fd.CurrentBlock.Sequences.DecodeNextSequenceSection(bufsrc, int(bytesLeft), &fd.PreviousBlock)
	if fd.Verbose {
		fd.printCurrentBlockSequences()
	}
	if err != nil {
		return err
	}

	bytesUsedWhileDecoding := int(bytesUsedByLiterals) + len(fd.CurrentBlock.Sequences.Data) + fd.CurrentBlock.Sequences.Header.BytesUsedByHeader
	if uint64(bytesUsedWhileDecoding) != fd.CurrentBlock.Header.BlockSize {
		return ErrCorruptSizes
	}
	if bytesUsedWhileDecoding != int(fd.CurrentBlock.Header.BlockSize) {
		return ErrCorruptSizes
	}
	if fd.limitedSource.N != 0 {
		return ErrCorruptSizes
	}

	return nil
}

var ErrWrongMagicnumber = errors.New("Magicnum is not correct")

func (fd *FrameDecompressor) CheckMagicnum() error {
	//read the magicnumber at the beginning of the file
	var magicnum [4]byte
	n := 0
	for n < 4 {
		x, err := fd.source.Read(magicnum[:])
		n += x
		if err != nil {
			return err
		}
	}
	var magicnumshould [4]byte
	binary.LittleEndian.PutUint32(magicnumshould[:], 0xFD2FB528)

	for idx := range magicnum {
		if magicnum[idx] != magicnumshould[idx] {
			return ErrWrongMagicnumber
		}
	}
	return nil
}

//Decompress just decompresses the whole frame and writes the whole output to the target
func (fd *FrameDecompressor) Decompress() error {
	err := fd.CheckMagicnum()
	if err != nil {
		return err
	}

	err = fd.DecodeFrameHeader()
	if err != nil {
		return err
	}

	err = fd.decodeAllBlocks()

	if err != nil {
		return err
	}
	return nil
}

func (fd *FrameDecompressor) printCurrentBlockHeader() {
	println("##############################")
	print("Next Block: ")
	println(fd.BlockCounter)
	println("##############################")

	msh, _ := json.MarshalIndent(fd.CurrentBlock.Header, "\t", "   ")
	println("\t" + string(msh))
	if fd.CurrentBlock.Header.Type == structure.BlockTypeRaw {
		println("\tRaw Block, just copy blocksize bytes.\n")
	}
}

func (fd *FrameDecompressor) printCurrentBlockLiterals() {
	println("Literals")
	msh, _ := json.MarshalIndent(fd.CurrentBlock.Literals, "\t", "   ")
	println("\t" + string(msh))
}
func (fd *FrameDecompressor) printCurrentBlockSequences() {
	println("Sequences")
	msh, _ := json.MarshalIndent(fd.CurrentBlock.Sequences, "\t", "   ")
	println("\t" + string(msh))
}

var ErrOutOfBlocks = errors.New("No blocks left in frame")

func (fd *FrameDecompressor) DecodeNextBlock() error {
	if fd.CurrentBlock.Header.LastBlock {
		return ErrOutOfBlocks
	}
	err := fd.DecodeNextBlockHeader()
	if fd.Verbose {
		fd.printCurrentBlockHeader()
	}
	if err != nil {
		return err
	}

	switch fd.CurrentBlock.Header.Type {
	case structure.BlockTypeRaw:
		_, err := io.CopyN(fd.decodebuffer, fd.source, int64(fd.CurrentBlock.Header.BlockSize))
		if err != nil {
			return err
		}

	case structure.BlockTypeCompressed:
		fd.limitedSource = &io.LimitedReader{R: fd.source, N: int64(fd.CurrentBlock.Header.BlockSize)}
		err = fd.DecodeNextBlockContent()

		if err != nil {
			return err
		}

		err = fd.ExecuteSequences()
		if err != nil {
			return err
		}
	default:
		//TODO implement RLE Blocks
		var b [1]byte
		b[0], err = fd.source.ReadByte()
		if err != nil {
			return err
		}
		for i := uint64(0); i < fd.CurrentBlock.Header.BlockSize; i++ {
			err = fd.decodebuffer.Push(b[:])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (fd *FrameDecompressor) decodeAllBlocks() error {
	for !fd.CurrentBlock.Header.LastBlock {
		err := fd.DecodeNextBlock()
		if err != nil {
			return err
		}

		fd.BlockCounter++
	}
	if fd.Verbose {
		println("##############################")
		println("Blocks Done")
		println("##############################")
	}

	fd.decodebuffer.Flush()
	if fd.Verbose {
		print("Total written: ")
		println(fd.decodebuffer.dumped)
	}
	return nil
}

//DecodeNextBlockHeader reads the next blockheader and swaps out previous block with currentblock
func (fd *FrameDecompressor) DecodeNextBlockHeader() error {
	buf := fd.headerbuffer[:3] //just need three bytes

	_, err := io.ReadFull(fd.source, buf)
	if err != nil {
		return err
	}

	//discard old block
	newBlock := structure.Block{}
	err = newBlock.DecodeHeader(buf)

	//carry over any decoding tables from the old current block
	if fd.CurrentBlock.Sequences.LiteralLengthsFSEDecodingTable != nil {
		fd.PreviousBlock.Sequences.LiteralLengthsFSEDecodingTable = fd.CurrentBlock.Sequences.LiteralLengthsFSEDecodingTable
	}
	if fd.CurrentBlock.Sequences.MatchLengthsFSEDecodingTable != nil {
		fd.PreviousBlock.Sequences.MatchLengthsFSEDecodingTable = fd.CurrentBlock.Sequences.MatchLengthsFSEDecodingTable
	}
	if fd.CurrentBlock.Sequences.OffsetsFSEDecodingTable != nil {
		fd.PreviousBlock.Sequences.OffsetsFSEDecodingTable = fd.CurrentBlock.Sequences.OffsetsFSEDecodingTable
	}
	if fd.CurrentBlock.Literals.DecodingTable != nil {
		fd.PreviousBlock.Literals.DecodingTable = fd.CurrentBlock.Literals.DecodingTable
	}

	//carry over buffers for reuse
	newBlock.Literals.CompressedData = fd.literalsCompressedDataBuf[:]
	newBlock.Literals.Data = fd.literalsDataBuf[:]
	newBlock.Sequences.Data = fd.sequencesDataBuf[:]

	fd.CurrentBlock = newBlock
	return err
}

//DecodeFrameHeader before starting to read the blocks
func (fd *FrameDecompressor) DecodeFrameHeader() error {

	n := 0
	var err error

	//read until at least one byte headerdescriptor has been read
	for n <= 0 {
		n, err = fd.source.Read(fd.headerbuffer[:1])
		if err != nil {
			return err
		}
	}

	fd.frame.Header.DecodeFrameDescriptor(fd.headerbuffer[0])

	windowdescriptorsize := byte(0)
	if !fd.frame.Header.Descriptor.GetSingleSegmentFlag() {
		windowdescriptorsize = 1
	}

	dictIDsize, err := fd.frame.Header.Descriptor.GetDictionaryFlag()
	if err != nil {
		return err
	}

	framecontentsize, err := fd.frame.Header.Descriptor.GetContentSizeFlag()
	if err != nil {
		return err
	}

	headersize := int(windowdescriptorsize + dictIDsize + framecontentsize)
	_, err = io.ReadFull(fd.source, fd.headerbuffer[:headersize])
	if err != nil {
		return err
	}

	buf := fd.headerbuffer[:headersize]
	if !fd.frame.Header.Descriptor.GetSingleSegmentFlag() {
		fd.frame.Header.DecodeWindowSize(buf[0])
		buf = buf[1:]
	}

	if dictIDsize > 0 {
		fd.frame.Header.DecodeDictionaryID(buf[:dictIDsize])
		buf = buf[dictIDsize:]
	}

	if framecontentsize > 0 {
		fd.frame.Header.DecodeFrameContentSize(buf[:framecontentsize])
		buf = buf[:framecontentsize]

		//if single segment, all data must fit into the window
		if fd.frame.Header.Descriptor.GetSingleSegmentFlag() {
			fd.frame.Header.WindowSize = fd.frame.Header.FrameContentSize
		}
	}

	if fd.decodebuffer == nil {
		fd.decodebuffer = NewRingbuffer(int(fd.frame.Header.WindowSize), fd.target)
	} else {
		fd.decodebuffer.Reset(int(fd.frame.Header.WindowSize), fd.target)
	}

	if fd.Verbose {
		fd.printStatus()
	}

	return nil
}
