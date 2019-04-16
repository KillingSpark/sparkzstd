package structure

import (
	"bufio"
	"bytes"
	"github.com/killingspark/sparkzsdt/bitstream"
	"github.com/killingspark/sparkzsdt/fse"
	"io"
)

type HuffmanEncodingType byte

const (
	HuffmanEncodingTypeCompressed = 0
	HuffmanEncodingTypeDirect     = 1
)

type HuffmanTreeDesc struct {
	Type            HuffmanEncodingType
	LengthInByte    int //only relevant if type == Compressed
	NumberOfWeights int //only relevant if type == Direct

	NumBits map[int]int `json:"-"`
	Weights []byte      `json:"-"`

	DecodingTable map[int]int
}

type HuffmanDecodingTable struct {
	MaxBits int

	NumberOfBits []int
	Symbols      []int

	State int
}

//DecodeFramStream returns number of bytes used
func (htd *HuffmanTreeDesc) DecodeFromStream(source *bufio.Reader) (int, error) {
	header, err := source.ReadByte()
	if err != nil {
		return 0, err
	}

	bytesRead := 1

	if header < 128 {
		htd.Type = HuffmanEncodingTypeCompressed
		htd.LengthInByte = int(header)

		fset := fse.FSETable{}
		bs, err := fset.ReadTabledescriptionFromBitstream(source)
		bytesRead += bs
		if err != nil {
			return bytesRead, err
		}

		err = fset.BuildDecodingTable(nil, nil)
		if err != nil {
			return bytesRead, err
		}
		// Need to copy because we read two interleaved streams but with the same decoding table.
		// This shallow copy is enough because we only need to have separate states. The decoding table can be shared
		fset2 := fset

		//print("AccuracyLog: ")
		//println(fset.AccuracyLog)
		//for i := 0; i < len(fset.NumBits); i++ {
		//	print(i)
		//	print(",")
		//	println(fset.NumBits[i].Symbol)
		//}
		//println("")
		//
		//for i := 0; i < len(fset.Values); i++ {
		//	print(i)
		//	print(",")
		//	println(fset.Values[i])
		//}

		bitStreamLength := htd.LengthInByte - bs
		buffer := make([]byte, bitStreamLength)
		read, err := io.ReadFull(source, buffer)
		bytesRead += read
		if err != nil {
			return bytesRead, err
		}
		if read != bitStreamLength {
			panic("Corrupt lengths")
		}

		weightsOutput := bytes.Buffer{}
		fse.DecodeInterleavedFSEStreams([]*fse.FSETable{&fset, &fset2}, buffer, &weightsOutput)

		htd.Weights = weightsOutput.Bytes()

		//for _, b := range htd.Weights {
		//	print(b)
		//	print(" ")
		//}
		//println("")

	} else {
		htd.Type = HuffmanEncodingTypeDirect
		htd.NumberOfWeights = int(header - 127)

		htd.Weights = make([]byte, htd.NumberOfWeights)
		var buf byte
		for i := 0; i < htd.NumberOfWeights; i++ {
			if i%2 == 0 {
				buf, err = source.ReadByte()
				if err != nil {
					return i / 2, err
				}
				htd.Weights[i] = buf >> 4
			} else {
				htd.Weights[i] = buf & 0xF
			}
		}
	}

	return bytesRead, nil
}

func (htd *HuffmanTreeDesc) Build() *HuffmanDecodingTable {
	sum := uint64(0)
	for _, w := range htd.Weights {
		weight := uint64(0)
		if w > 0 {
			weight = uint64(1) << uint(w-1)
		}
		sum += weight
	}

	//print("Weightsum: ")
	//println(sum)

	log := fse.BIT_highbit32(uint32(sum)) + 1
	actualSum := uint64(1) << log
	leftOver := actualSum - sum
	if leftOver&(leftOver-1) != 0 {
		print("ActualSum: ")
		println(actualSum)
		print("LeftOver: ")
		println(leftOver)
		panic("Should be power of two")
	}
	lastWeight := fse.BIT_highbit32(uint32(leftOver)) + 1

	maxBits := log
	htd.NumBits = make(map[int]int, len(htd.Weights)+1)

	rankCount := make([]int, maxBits+1)

	for idx, w := range htd.Weights {
		nob := 0
		if w > 0 {
			nob = int(maxBits) + 1 - int(w)
		}
		htd.NumBits[idx] = nob
		//print("NumBits: ")
		//print(idx)
		//print(", ")
		//println(nob)

		rankCount[nob]++
	}

	//for i := 0; i < len(rankCount); i++ {
	//	print("Rank_Count: ")
	//	print(i)
	//	print(", ")
	//	println(rankCount[i])
	//}

	lastNob := 0
	if lastWeight > 0 {
		lastNob = int(maxBits) + 1 - int(lastWeight)
	}
	htd.NumBits[len(htd.Weights)] = lastNob
	rankCount[lastNob]++

	//########
	//##Actually fill the Table
	//########

	table := HuffmanDecodingTable{}
	table.MaxBits = int(maxBits)
	table.Symbols = make([]int, 1<<maxBits)
	table.NumberOfBits = make([]int, 1<<maxBits)

	rankIdx := make([]int, maxBits+1)
	for i := maxBits; i >= 1; i-- {
		rankIdx[i-1] = rankIdx[i] + rankCount[i]*(1<<(maxBits-i))

		base := rankIdx[i]
		for j := 0; j < rankIdx[i-1]-rankIdx[i]; j++ {
			table.NumberOfBits[base+j] = int(i)
		}
	}

	if rankIdx[0] != len(table.NumberOfBits) {
		panic("This should be the same")
	}

	for i := 0; i < len(htd.NumBits); i++ {
		if htd.NumBits[i] != 0 {
			code := rankIdx[htd.NumBits[i]]
			len := 1 << uint(int(maxBits)-htd.NumBits[i])

			for j := 0; j < len; j++ {
				table.Symbols[code+j] = i
			}
			rankIdx[htd.NumBits[i]] += len
		}
	}

	//for i := 0; i < len(table.Symbols); i++ {
	//	if table.NumberOfBits[i] != 0 {
	//		print("NoB: ")
	//		print(i)
	//		print(", ")
	//		println(table.NumberOfBits[i])
	//		print("Symbol: ")
	//		print(i)
	//		print(", ")
	//		println(table.Symbols[i])
	//	}
	//}

	return &table
}

func (ht *HuffmanDecodingTable) InitState(source *bitstream.Reversebitstream) error {
	state, err := source.Read(ht.MaxBits)
	ht.State = int(state)
	return err
}

//returns bits, symbol, error
func (ht *HuffmanDecodingTable) DecodeSymbol(source *bitstream.Reversebitstream) (int, int, error) {
	symbol := ht.Symbols[ht.State]
	bits := ht.NumberOfBits[ht.State]
	rest, err := source.Read(bits)
	//print("Bits")
	//println(bits)
	//print("Bitsleft")
	//println(source.BitsStillInStream())
	//print("Rest: ")
	//println(rest)

	if err != nil {
		return 0, 0, err
	}

	ht.State = int(uint16((ht.State<<uint(bits))+int(rest)) & ((uint16(1) << uint(ht.MaxBits)) - 1))
	return bits, symbol, nil
}

func (ht *HuffmanDecodingTable) DecodeStream(data, output []byte) (int, error) {
	bitsum := 0
	bitsrc := bitstream.NewReversebitstream(data)

	var err error
	//need to read bits from the stream (the back of the data...) until the first 1 arrives
	x := uint64(0)
	for x == 0 {
		x, err = bitsrc.Read(1)
		if err != nil {
			return 0, err
		}
		bitsum++
	}

	if x > 7 {
		panic("Bitstream is corrupt. More then the first(or rather last) byte was zero")
	}

	err = ht.InitState(bitsrc)
	if err != nil {
		return 0, err
	}
	bitsum += ht.MaxBits

	totalOutput := 0

	for i := 0; bitsrc.BitsStillInStream()+1 > -ht.MaxBits; i++ {
		bits, symbol, err := ht.DecodeSymbol(bitsrc)
		bitsum += bits
		if err != nil {
			return totalOutput, err
		}
		output[i] = byte(symbol)
		totalOutput++
		//if i > 14193 {
		//	print(i)
		//	print(", ")
		//	println(ht.State)
		//}
	}
	if bitsrc.BitsStillInStream()+1 != -ht.MaxBits {
		println(bitsrc.BitsStillInStream() + 1)
		println(-ht.MaxBits)
		panic("Should be the same")
	}

	//if bitsToByte(bitsum) != len(data) {
	//	println(bitsToByte(bitsum))
	//	println(len(data))
	//	panic("Didnt read all data")
	//}
	return totalOutput, nil
}
