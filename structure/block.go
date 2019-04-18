package structure

import (
	"errors"
	"strconv"
)

type Block struct {
	Header    BlockHeader
	Literals  LiteralSection
	Sequences SequencesSection
}

type BlockType byte

const (
	BlockTypeRaw        = BlockType(0)
	BlockTypeRLE        = BlockType(1)
	BlockTypeCompressed = BlockType(2)
	BlockTypeReserved   = BlockType(3)
)

type BlockHeader struct {
	LastBlock bool
	Type      BlockType
	BlockSize uint64
}

var ErrNotEnoughBytesForBlockHeader = errors.New("Not enough / too much bytes to decode the blockheader. Must be 3.")
var ErrIllegalBlockType = errors.New("Illegal BlockType. Must be smaller than 3.")

//DecodeHeader takes the 3 raw bytes and fills the
func (bl *Block) DecodeHeader(raw []byte) error {
	if len(raw) != 3 {
		return ErrNotEnoughBytesForBlockHeader
	}

	bl.Header.LastBlock = (raw[0] & 0x1) == 1       //mask all but the least bit and compare
	bl.Header.Type = BlockType((raw[0] >> 1) & 0x3) //shift LastBlock flag out and mask all other bits but the least two

	bl.Header.BlockSize = uint64(raw[0] >> 3) // need to shift out the three lowest bits. (type and lastblock flags)
	bl.Header.BlockSize += uint64(raw[1]) << 5
	bl.Header.BlockSize += uint64(raw[2]) << 13

	if bl.Header.Type >= BlockTypeReserved {
		return ErrIllegalBlockType
	}

	//maximum 128KB
	if bl.Header.BlockSize > (128 * 1024) {
		return errors.New("BlockSize too big: " + strconv.Itoa(int(bl.Header.BlockSize)))
	}

	return nil
}
