package bitstream

type Reversebitstream struct {
	Data   []byte
	offset int
}

//cant implemented this on a reader because we need to read backwards
func NewReversebitstream(data []byte) *Reversebitstream {
	return &Reversebitstream{Data: data, offset: (len(data) * 8) - 1}
}

func (rbs *Reversebitstream) BitsStillInStream() int {
	return rbs.offset
}

func (rbs *Reversebitstream) Read(n int) (uint64, error) {
	var value uint64

	if rbs.offset <= -1 {
		//we allow reading over the "end" of the stream, and we update the offset accordingly
		rbs.offset -= n
		return 0, nil
	}

	bitsLeftInLastByte := (rbs.offset + 1) % 8
	if bitsLeftInLastByte == 0 {
		bitsLeftInLastByte = 8 // if mod 8 == 0 --> we are at the last bit of the next byte
	}
	idxOfLastByte := rbs.offset / 8

	//can satisfy with bits from the last byte
	if n < bitsLeftInLastByte {
		keepBitsInLastByte := uint(bitsLeftInLastByte - n) //how many bits we need ro preserve in the last byte
		mask := byte(1<<uint(n)) - 1                       //mask for the "n" bits we actually want to add to the value
		var tmp byte
		tmp = rbs.Data[idxOfLastByte]   //local copy
		tmp = tmp >> keepBitsInLastByte //shift out the lower bits we need to keep in the last byte

		value = uint64(tmp & mask)

		rbs.offset -= n
		return value, nil
	}

	//take bits from last byte
	lastByteMask := byte(1<<uint(bitsLeftInLastByte)) - 1 //mask for the lower bits we actually want to add to the value
	lastByteValue := uint64(rbs.Data[idxOfLastByte] & lastByteMask)

	if n == bitsLeftInLastByte {
		//finished. No need for any looping
		rbs.offset -= n
		return lastByteValue, nil
	}

	spanningFullBytes := (n - bitsLeftInLastByte) / 8
	idxOfLowestByte := idxOfLastByte - spanningFullBytes //index of lowest full byte

	//take bits from the one below the lowest full byte
	//shift because we want the highest bits. We read backwards.
	bitsFromLowestByte := (n - bitsLeftInLastByte) % 8
	shiftInLowestByte := 8 - bitsFromLowestByte

	//we may go beyond the offset "-1". Act as if those were just zeros
	if idxOfLowestByte-1 >= 0 {
		value = uint64(rbs.Data[idxOfLowestByte-1] >> uint(shiftInLowestByte))
	}

	start := 0
	if idxOfLowestByte < 0 {
		start = -idxOfLowestByte
	}

	for i := start; i < spanningFullBytes; i++ {
		value += uint64(rbs.Data[idxOfLowestByte+i]) << uint(i*8+bitsFromLowestByte)
	}

	shift := uint(spanningFullBytes*8 + bitsFromLowestByte)
	add := lastByteValue << shift

	value += add

	rbs.offset -= n
	return value, nil
}
