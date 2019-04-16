package fse

import (
	"bufio"
	"errors"
	"github.com/killingspark/sparkzsdt/bitstream"
	"io"
)

type FSETableEntry struct {
	Baseline               uint16
	NumberOfAdditionalBits byte
	NumberOfBits           byte
	Symbol                 int
}

type FSETable struct {
	AccuracyLog   int
	Values        map[int]int64 //note that the probability is the value in this map -1
	DecodingTable []*FSETableEntry

	//State needed while decoding
	State int64
}

//DecodeBitstream reads the source byte by byte until the table has been built.
//It will report back any error and the number of bytes taken out of the reader
func (fset *FSETable) ReadTabledescriptionFromBitstream(source *bufio.Reader) (int, error) {
	fset.Values = make(map[int]int64)
	//TODO implement

	bitsrc := bitstream.NewBitstream(source)
	acclog, err := bitsrc.Read(4)
	if err != nil {
		return 0, err
	}

	fset.AccuracyLog = int(acclog) + 5

	sumOfProbabilities := int64(1) << uint(fset.AccuracyLog) //2^accuracylog
	remaining := sumOfProbabilities

	bitsRead := 4
	currentSymbol := 0

	for remaining > 0 {
		BitsNeeded := BIT_highbit32(uint32(remaining+1)) + 1

		v, err := bitsrc.Read(int(BitsNeeded))
		bitsRead += int(BitsNeeded)
		if err != nil {
			bytesRead := bitsRead / 8
			if bitsRead%8 != 0 {
				bytesRead++
			}
			return bytesRead, err
		}
		value := uint16(v)

		lowermask := (uint16(1) << (BitsNeeded - 1)) - 1
		thresh := (uint16(1) << (BitsNeeded)) - 1 - uint16((remaining + 1))

		if (value & lowermask) < thresh {
			// "small" number. Unwind last bit read
			bitsrc.UnwindBit()
			bitsRead--
			value = value & lowermask
		} else {
			if value > lowermask {
				value = value - thresh
			}
		}

		fset.Values[currentSymbol] = int64(value)
		currentSymbol++

		probabilitiy := int(value) - 1
		if probabilitiy == -1 {
			remaining-- //counts as a 1 because it will get one cell in the decoding table
		} else {
			remaining -= int64(probabilitiy)
		}

		//next two bit tell us how many symbols following have probability 0 too
		if probabilitiy == 0 {
			//if skip == 3 we need to read two more bits which tell us how many more symbols to skip
			skip := uint64(3)

			for skip == 3 {
				skip, err = bitsrc.Read(2)
				bitsRead += 2
				if err != nil {
					bytesRead := bitsRead / 8
					if bitsRead%8 != 0 {
						bytesRead++
					}
					return bytesRead, err
				}

				for i := uint64(0); i < skip; i++ {
					//does not count into remaining because probability == 0
					fset.Values[currentSymbol] = 1 //values = probability+1!
					currentSymbol++
				}
			}
		}
	}

	if remaining != 0 {
		print(remaining)
		print("/")
		println(sumOfProbabilities)
		panic("This should not happen")
	}

	bytesRead := bitsRead / 8
	if bitsRead%8 != 0 {
		bytesRead++
	}
	return bytesRead, nil
}

//BuildDecodingTable more or less is oriented on the implementation in https://github.com/facebook/zstd
// symbolTranslation may be nil. Then the symbols will just not be translated
func (fset *FSETable) BuildDecodingTable(symbolTranslation []int, extraBits []byte) error {

	symbolNext := make([]int, len(fset.Values))

	tablesize := 1 << uint(fset.AccuracyLog)
	highposition := tablesize - 1

	fset.DecodingTable = make([]*FSETableEntry, tablesize)

	//first find all symbols with a -1 probability
	for symbol := 0; symbol < len(fset.Values); symbol++ {
		probability := fset.Values[symbol] - 1
		if probability == -1 {
			fset.DecodingTable[highposition] = &FSETableEntry{Symbol: symbol}
			highposition--
			symbolNext[symbol] = 1 //full reset on these symbols
		} else {
			symbolNext[symbol] = int(probability)
		}
	}

	if highposition < 0 {
		panic("Too many small probabilities")
	}

	position := 0

	//then place all other symbols
	for symbol := 0; symbol < len(fset.Values); symbol++ {
		probability := fset.Values[symbol] - 1
		if probability != -1 {

			//allocate probability many cells to this symbol
			for i := int64(0); i < probability; i++ {
				if fset.DecodingTable[position] != nil {
					panic("Overwriting should never happen")
				}

				fset.DecodingTable[position] = &FSETableEntry{Symbol: symbol}

				//weird jumping around
				position += (tablesize >> 1) + (tablesize >> 3) + 3
				position &= tablesize - 1

				//skipping the "low probability" area
				for position > highposition {
					position += (tablesize >> 1) + (tablesize >> 3) + 3
					position &= tablesize - 1
				}
			}
		}
	}

	if position != 0 {
		panic("Position did not end up at 0")
	}

	//ported from https://github.com/facebook/zstd
	for i := 0; i < tablesize; i++ {
		entry := fset.DecodingTable[i]
		symbol := entry.Symbol
		nextState := uint32(symbolNext[symbol])
		symbolNext[symbol]++

		//this is some fancy magic that apparently does the same thing they describe WAY differently and pretty vague in their documentation

		// basically what happens is: every symbol has one or more states assigned in the decodingtable. Lets call these a group of states
		// each of these groups splits the set of all states into ranges. "Lower" ones have double the amout of higher ones.
		// this determines the values
		// "NumberOfBits" <-- how many bits are needed to encode the values in the range
		// "BaseLine" <-- start of the Range
		// the following code cleverly counts occurences of symbols in the decoding table (since we iterate monotonly increasing by index)
		// and thus is able to infere the values based on that.

		// This is probably just an efficiency optimization. Otherwise you'd have to actually collect the groups and
		// iterate over these.
		// TODO make an implementation that does the "umoptimized way". This could be way more readable and with clever
		// use of maps this shouldnt be too bad performance wise.
		fset.DecodingTable[i].NumberOfBits = byte(uint32(fset.AccuracyLog) - BIT_highbit32(uint32(nextState)))
		fset.DecodingTable[i].Baseline = uint16((nextState << fset.DecodingTable[i].NumberOfBits) - uint32(tablesize))

		//only do translation if necessary. Offsets dont need to
		if len(symbolTranslation) > symbol {
			fset.DecodingTable[i].Symbol = symbolTranslation[symbol] //translate the symbols in the end to the real ones
		}
		if len(extraBits) > symbol {
			fset.DecodingTable[i].NumberOfAdditionalBits = extraBits[symbol] //extra bits needed for decoding the sequences
		}

		//print("State: ")
		//print(i)
		//print(" --> ")
		//println(symbol)
	}

	return nil
}

//Bit_highbit32 returns the index of the highest set bit in the uint32
//implementation (of the software variant wihtout VisualC or gcc support) ported from https://github.com/facebook/zstd bitsream.h
//which is more or less the same as the code from wikipedia on this function anyways
func BIT_highbit32(value uint32) uint32 {
	DeBruijnClz := [32]uint32{
		0, 9, 1, 10, 13, 21, 2, 29,
		11, 14, 16, 18, 22, 25, 3, 30,
		8, 12, 20, 28, 15, 17, 24, 7,
		19, 27, 23, 6, 26, 5, 4, 31}

	v := value
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	return DeBruijnClz[uint32(uint64(v)*uint64(0x07C4ACDD))>>27]
}

//InitState reads the inital state from the bitstream
//returns the number of bits read
func (fset *FSETable) InitState(src *bitstream.Reversebitstream) (int, error) {
	state, err := src.Read(fset.AccuracyLog)
	fset.State = int64(state)
	return fset.AccuracyLog, err
}

var ErrNoSymbolForState = errors.New("Probably a bad state?")

func (fset *FSETable) GetAdditionalBits() (int, error) {
	return int(fset.DecodingTable[fset.State].NumberOfAdditionalBits), nil
}
func (fset *FSETable) GetNumberOfBits() (int, error) {
	return int(fset.DecodingTable[fset.State].NumberOfBits), nil
}
func (fset *FSETable) GetState() int {
	return int(fset.State)
}

//PeekSymbol returns the symbol the current state decodes to, without advancing the state
func (fset *FSETable) PeekSymbol() (int, error) {
	if fset.State > int64(len(fset.DecodingTable)) {
		return 0, ErrNoSymbolForState
	}

	return fset.DecodingTable[fset.State].Symbol, nil
}

//NextState reads bits from the stream to determin the next state
//returns the number of bits read
func (fset *FSETable) NextState(src *bitstream.Reversebitstream) (int, error) {
	bitsNeeded := fset.DecodingTable[fset.State].NumberOfBits
	baseLine := fset.DecodingTable[fset.State].Baseline
	add, err := src.Read(int(bitsNeeded))
	if err == nil {
		fset.State = int64(baseLine) + int64(add)
	}
	return int(bitsNeeded), err
}

//DecodeSymbol Combines PeekSymbol and NextState
func (fset *FSETable) DecodeSymbol(src *bitstream.Reversebitstream) (symbol int, bitsRead int, err error) {
	symbol, err = fset.PeekSymbol()
	if err != nil {
		return
	}

	bitsRead, err = fset.NextState(src)
	return
}

// DecodeInterleavedFSEStreams intializes the states for each table in the order of the slice and
// then decodes values in a round robin fashion
func DecodeInterleavedFSEStreams(decodingTables []*FSETable, src []byte, target io.Writer) (int, error) {
	bitsRead := 0
	bitsrc := bitstream.NewReversebitstream(src)
	var err error

	//need to read bits from the stream (the back of the data...) until the first 1 arrives
	x := uint64(0)
	for x == 0 {
		x, err = bitsrc.Read(1)
		if err != nil {
			return bitsRead, err
		}
		bitsRead++
	}

	if x > 7 {
		panic("Bitstream is corrupt. More then the first(or rather last) byte was zero")
	}

	//print("Padding: ")
	//println(bitsRead)

	for _, dt := range decodingTables {
		read, err := dt.InitState(bitsrc)
		bitsRead += read
		if err != nil {
			return bitsRead, err
		}
	}

	var buf [1]byte

	shouldFinish := false
	//loop until the end of the stream is reached
	for !shouldFinish {
		for idx, dt := range decodingTables {
			symbol, read, err := dt.DecodeSymbol(bitsrc)
			if err != nil {
				return bitsRead, err
			}
			bitsRead += read
			buf[0] = byte(symbol)

			w := 0
			for w == 0 {
				w, err = target.Write(buf[:])
				if err != nil {
					return bitsRead, err
				}
			}

			//print(symbol)
			//print(", ")
			//println(bitsrc.BitsStillInStream() + 1)

			if bitsrc.BitsStillInStream() < -1 {
				//collect all streams last symbol and exit outer loop
				for i := 1; i < len(decodingTables); i++ {
					peekIdx := (i + idx) % len(decodingTables)
					symbol, err := decodingTables[peekIdx].PeekSymbol()
					if err != nil {
						return bitsRead, err
					}
					buf[0] = byte(symbol)

					//print(symbol)
					//print(", ")
					//println(bitsrc.BitsStillInStream() + 1)

					w := 0
					for w == 0 {
						w, err = target.Write(buf[:])
						if err != nil {
							return bitsRead, err
						}
					}
				}
				shouldFinish = true
				break
			}
		}
	}
	return bitsRead, nil
}
