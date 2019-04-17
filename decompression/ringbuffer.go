package decompression

import (
	"errors"
	"io"
)

//Ringbuffer is a buffer for decompression. It provides all methods needed for sequence execution (literal copy and offset/match copy)
type Ringbuffer struct {
	data      []byte
	repeatBuf []byte

	allDirty bool

	offset int
	Len    int

	Dump   io.Writer
	dumped int
}

//NewRingbuffer creates a new Ringbuffer with the appropriatly sized buffer
func NewRingbuffer(n int, dump io.Writer) *Ringbuffer {
	return &Ringbuffer{data: make([]byte, n), repeatBuf: make([]byte, n), offset: 0, Len: n, Dump: dump, allDirty: false}
}

//ErrIdxOutOfBounds is returned if Get(X) x is bigger than rb.Len
var ErrIdxOutOfBounds = errors.New("Index is out of bounds")

func (rb *Ringbuffer) calcRealIdx(idx int) int {
	return (idx + rb.offset) % len(rb.data)
}

//Get can be used to iterate over the buffer in the "correct" order
func (rb *Ringbuffer) Get(idx int) (byte, error) {
	if idx > len(rb.data) {
		return 0, ErrIdxOutOfBounds
	}

	return rb.data[rb.calcRealIdx(idx)], nil
}

//WriteFull is the equivalent to io.ReadFull
func WriteFull(w io.Writer, data []byte) (int, error) {
	written := 0
	for written < len(data) {
		x, err := w.Write(data)
		written += x
		if err != nil {
			return written, err
		}
	}
	return written, nil
}

func (rb *Ringbuffer) dumpAllDirty() error {
	//if all dirty is set there is old data that needs to be flushed
	if rb.allDirty {
		err := rb.dump(rb.offset, rb.Len)
		if err != nil {
			return err
		}
	}

	err := rb.dump(0, rb.offset)
	if err != nil {
		return err
	}

	//reset buffer state
	rb.offset = 0
	rb.allDirty = false

	return nil
}

//Push appends the new Data at the "end" of the buffer and dumps everything that would have been overwritten
func (rb *Ringbuffer) Push(newdata []byte) error {

	//deal with really large new data. Should not happen in zstd context
	if len(newdata) >= rb.Len {
		err := rb.dumpAllDirty()
		if err != nil {
			return err
		}

		// directly dump all data that would have been overwritten anyways
		//Keep exactly rb.Len many bytes and reset the offset to zero
		_, err = WriteFull(rb.Dump, newdata[:len(newdata)-rb.Len])
		if err != nil {
			return err
		}

		copy(rb.data, newdata[len(newdata)-rb.Len:])
		rb.offset = 0
		rb.allDirty = true
		return nil
	}

	offsetLength := len(newdata) + rb.offset
	if offsetLength <= rb.Len {
		// dont need to wrap. just dump until the upper index and write new data

		if rb.allDirty {
			err := rb.dump(rb.offset, offsetLength)
			if err != nil {
				return err
			}
		}
		copy(rb.data[rb.offset:offsetLength], newdata)

		rb.offset += len(newdata)
		if rb.offset >= rb.Len {
			rb.allDirty = true
		}
		rb.offset %= rb.Len
		return nil
	}

	//need to wrap. first dump all data after offset if dirty
	spaceAboveOffset := rb.Len - rb.offset
	if rb.allDirty {
		err := rb.dump(rb.offset, rb.Len)
		if err != nil {
			return err
		}
	}

	//write new data until the end of the buffer
	copy(rb.data[rb.offset:], newdata[:spaceAboveOffset])

	//then dump all data that would be overwritten by the rest of newdata
	rest := newdata[spaceAboveOffset:]
	err := rb.dump(0, len(rest))
	if err != nil {
		return err
	}

	//write the rest of the new data over the old data we just dumped
	copy(rb.data[:len(rest)], rest)

	rb.offset = len(rest)
	rb.allDirty = true
	if rb.offset >= rb.Len {
		panic("This should never happen. Large data gets handled ")
	}
	return nil
}

//For the sake of it implement io.Writer interface
func (rb *Ringbuffer) Write(data []byte) (int, error) {
	err := rb.Push(data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

//Repeat is used to provide the offset/matchlength sequence execution
//skip "after" newest bytes and repeat n bytes
// say buffer contains: 1234123456
//	n == 3, after == 4
//  this amounts to pushing to the back: 412
// result would be: 1234123456412
func (rb *Ringbuffer) Repeat(n int, after int) {
	buf := rb.repeatBuf[:n]

	start := rb.offset - after
	lowerBound := start - n

	if lowerBound >= 0 && start >= 0 {
		copy(buf, rb.data[lowerBound:start])
	} else {
		if !rb.allDirty {
			print("Offset: ")
			println(rb.offset)
			print("LowerBound: ")
			println(lowerBound)
			print("Start: ")
			println(start)
			panic("Cant fullfill. The area above the offset has no data")
		}
		bytesFromTop := -lowerBound
		if lowerBound < 0 && start >= 0 {
			copy(buf, rb.data[rb.Len-bytesFromTop:])
			copy(buf[bytesFromTop:], rb.data[:start])
		} else {
			if lowerBound < 0 && start < 0 {
				skipBytesFromTop := -start
				if (bytesFromTop - skipBytesFromTop) != n {
					panic("WRONG!")
				}
				copy(buf, rb.data[rb.Len-bytesFromTop:rb.Len-skipBytesFromTop])
			} else {
				panic("I forgot a case?")
			}
		}

	}

	rb.Push(buf)
}

//RepeatBeforeIndex is used to translate the semantics Zstd uses in the sequences
//to the one Ringbuffer.Repeat() offers.
//repeat n newest after (and including) the specified oldest byte
// abcdefgh <- current string(may be represented differently in the cyclic buffer)
// n = 3, oldest = 5
// repeat: "def"
// result string: abcdefghdef
func (rb *Ringbuffer) RepeatBeforeIndex(n int, oldest int) {
	skip := oldest - n

	//special case allowed by the zstd specification.
	//we need to "repeat" data that is not yet in the buffer
	if skip < 0 {
		//need to sequientally generate data. Want to match copy more than possible with simple copying
		//first clean out necessary space
		if rb.allDirty {
			rb.dumpWrapping(rb.offset, rb.offset+n)
		} else {
			spaceAboveOffset := rb.Len - rb.offset
			bytesGettingWrapped := n - spaceAboveOffset
			if bytesGettingWrapped > 0 {
				rb.dump(0, bytesGettingWrapped)
			}
		}
		//then generate the data
		for i := 0; i < n; i++ {
			idx := rb.offset - oldest
			if idx < 0 {
				idx = rb.Len + idx
			}
			rb.data[rb.offset] = rb.data[idx]
			rb.offset++
			rb.offset %= rb.Len
			if rb.offset == 0 {
				rb.allDirty = true
			}
		}
		return
	}

	rb.Repeat(n, skip)
}

//convenience function. Use if "high" might be higher than rb.Len
func (rb *Ringbuffer) dumpWrapping(low, high int) error {
	if !rb.allDirty {
		panic("Only use this if actually dirty")
	}
	modhigh := high % rb.Len
	if modhigh < low {
		err := rb.dump(low, rb.Len)
		if err != nil {
			return err
		}
		err = rb.dump(0, modhigh)
		if err != nil {
			return err
		}
	} else {
		err := rb.dump(low, modhigh)
		if err != nil {
			return err
		}
	}
	return nil
}

// write bytes in the specified range to the rb.Dump writer
func (rb *Ringbuffer) dump(low, high int) error {
	w, err := WriteFull(rb.Dump, rb.data[low:high])
	if err != nil {
		return err
	}
	if w != (high - low) {
		panic("AAAAA")
	}
	rb.dumped += w
	return nil
}

//RepeatLast the last n bytes that were written to the buffer
func (rb *Ringbuffer) RepeatLast(n int) {
	rb.Repeat(n, 0)
}

//Flush cleans out the buffer. Needed when we are finished decoding and want to write the last bytes into the stream/file/...
func (rb *Ringbuffer) Flush() {
	rb.dumpAllDirty()
}

//String is just for testing purposes but might be useful in other scenarios
//returns the content stitched together correctly
func (rb *Ringbuffer) String() string {
	if rb.allDirty {
		return string(rb.data[rb.offset:]) + string(rb.data[:rb.offset])
	}
	return string(rb.data[:rb.offset])
}
