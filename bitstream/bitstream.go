package bitstream

import (
	"bufio"
	"errors"
)

type Bitstream struct {
	buffer byte
	offset uint
	source *bufio.Reader
}

func NewBitstream(src *bufio.Reader) *Bitstream {
	return &Bitstream{buffer: 0, offset: 8, source: src}
}

var ErrCantUnwind = errors.New("Cant unwind more bits")

func (bs *Bitstream) UnwindBit() error {
	if bs.offset == 0 {
		// This should actually never happen on the first call to this after a read
		// We only read a new byte because we needed bits from it so there is always at least one bit to rewind after a read
		return ErrCantUnwind
	}
	bs.offset--
	if bs.offset == 0 {
		//if we unread the bit because of which we read the byte in the first place, push the byte back into the buffer
		err := bs.source.UnreadByte()
		if err != nil {
			return err
		}

		bs.offset = 8
	}
	return nil
}

func (bs *Bitstream) Read(n int) (uint64, error) {
	if n == 0 {
		return 0, nil
	}

	var err error
	var val uint64

	if bs.offset+uint(n) <= 8 { //buffer has enough bits ready
		mask := (byte(1) << uint(n)) - 1
		val = uint64((bs.buffer >> bs.offset) & mask)
		bs.offset += uint(n)
		return val, nil
	}

	//cant satisfy with buffer. flush buffer into value and read new bytes
	bitsFromBuffer := (8 - bs.offset)

	if bs.offset < 8 { //flush the bits from the buffer into the value
		val = uint64(bs.buffer >> bs.offset)
		bs.offset = 8
	}

	remainingBitsNeeded := uint(n) - bitsFromBuffer
	bitsNeededFromLastByte := remainingBitsNeeded % 8
	bytesNeeded := (remainingBitsNeeded / 8) //only full bytes counted. The last one will be read separatly if necessary

	//read bytes until the last one and add them to the value
	i := uint(0)
	for _ = i; i < bytesNeeded; i++ {
		bs.buffer, err = bs.source.ReadByte()
		if err != nil {
			return 0, err
		}
		val += uint64(bs.buffer) << (bitsFromBuffer + 8*i)
	}

	if bitsNeededFromLastByte > 0 {
		//read final byte into buffer, add to value, and set offset
		bs.buffer, err = bs.source.ReadByte()
		if err != nil {
			return 0, err
		}
		mask := (byte(1) << bitsNeededFromLastByte) - 1
		val += uint64(bs.buffer&mask) << (uint(n) - bitsNeededFromLastByte)
		bs.offset = bitsNeededFromLastByte
	} else {
		bs.offset = 8
	}

	return val, nil
}
