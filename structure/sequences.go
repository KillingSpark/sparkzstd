package structure

import (
	"bufio"
	"errors"
	"github.com/killingspark/sparkzsdt/bitstream"
	"github.com/killingspark/sparkzsdt/fse"
	"io"
)

type Sequence struct {
	MatchLength   int
	LiteralLength int
	Offset        int
}

type SequencesSection struct {
	Header                         SequencesSectionHeader
	LiteralLengthsFSEDecodingTable DecodingTable `json:"-"`
	MatchLengthsFSEDecodingTable   DecodingTable `json:"-"`
	OffsetsFSEDecodingTable        DecodingTable `json:"-"`
	Data                           []byte        `json:"-"`

	Sequences []Sequence `json:"-"`
}

type RepeatingDecodingTable byte

type DecodingTable interface {
	DecodeSymbol(src *bitstream.Reversebitstream) (symbol int, bitsRead int, err error)
	NextState(src *bitstream.Reversebitstream) (int, error)
	PeekSymbol() (int, error)
	GetAdditionalBits() (int, error)
	GetNumberOfBits() (int, error)
	InitState(src *bitstream.Reversebitstream) (int, error)
	GetState() int
}

func (rdc *RepeatingDecodingTable) DecodeSymbol(src *bitstream.Reversebitstream) (symbol int, bitsRead int, err error) {
	return int(*rdc), 0, nil
}
func (rdc *RepeatingDecodingTable) NextState(src *bitstream.Reversebitstream) (int, error) {
	return 0, nil
}
func (rdc *RepeatingDecodingTable) PeekSymbol() (int, error) {
	return int(*rdc), nil
}
func (rdc *RepeatingDecodingTable) GetAdditionalBits() (int, error) {
	return 0, nil
}
func (rdc *RepeatingDecodingTable) GetNumberOfBits() (int, error) {
	return 0, nil
}
func (rdc *RepeatingDecodingTable) InitState(src *bitstream.Reversebitstream) (int, error) {
	return 0, nil
}
func (rdc *RepeatingDecodingTable) GetState() int {
	return 0
}

func (ss *SequencesSection) DecodeSequence(source *bitstream.Reversebitstream) (Sequence, int, error) {
	var seq Sequence

	ofcode, err := ss.OffsetsFSEDecodingTable.PeekSymbol()
	if err != nil {
		return seq, 0, err
	}
	llcode, err := ss.LiteralLengthsFSEDecodingTable.PeekSymbol()
	if err != nil {
		return seq, 0, err
	}
	mlcode, err := ss.MatchLengthsFSEDecodingTable.PeekSymbol()
	if err != nil {
		return seq, 0, err
	}

	//print("of stat: ")
	//println(ss.OffsetsFSEDecodingTable.State)
	//print("of Code: ")
	//println(ofcode)
	//print("ll stat: ")
	//println(ss.LiteralLengthsFSEDecodingTable.State)
	//print("ll Code: ")
	//println(llcode)
	//print("ml stat: ")
	//println(ss.MatchLengthsFSEDecodingTable.State)
	//print("ml Code: ")
	//println(mlcode)
	//println("")

	bitsRead := 0

	offset, err := source.Read(ofcode)
	if err != nil {
		return seq, bitsRead, err
	}
	bitsRead += ofcode
	seq.Offset = (1 << uint(ofcode)) + int(offset)

	mlextrabits, _ := ss.MatchLengthsFSEDecodingTable.GetAdditionalBits()
	mlextra, err := source.Read(mlextrabits)
	if err != nil {
		return seq, bitsRead, err
	}
	bitsRead += mlextrabits
	seq.MatchLength = mlcode + int(mlextra)

	llextrabits, _ := ss.LiteralLengthsFSEDecodingTable.GetAdditionalBits()
	llextra, err := source.Read(llextrabits)
	if err != nil {
		return seq, bitsRead, err
	}
	bitsRead += llextrabits
	seq.LiteralLength = llcode + int(llextra)

	//print(seq.Offset)
	//print(" \t ")
	//print(seq.MatchLength)
	//print(" \t ")
	//println(seq.LiteralLength)

	return seq, bitsRead, nil
}

//return bits read
func (ss *SequencesSection) DecodeSequences() (int, error) {
	bitsRead := 0
	bitsrc := bitstream.NewReversebitstream(ss.Data)
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

	//print("padding: ")
	//println(x)

	bits, err := ss.LiteralLengthsFSEDecodingTable.InitState(bitsrc)
	bitsRead += bits
	if err != nil {
		return bitsRead, err
	}
	bits, err = ss.OffsetsFSEDecodingTable.InitState(bitsrc)
	bitsRead += bits
	if err != nil {
		return bitsRead, err
	}
	bits, err = ss.MatchLengthsFSEDecodingTable.InitState(bitsrc)
	bitsRead += bits
	if err != nil {
		return bitsRead, err
	}

	//print("llstate: ")
	//println(ss.LiteralLengthsFSEDecodingTable.GetState())
	//print("ofstate: ")
	//println(ss.OffsetsFSEDecodingTable.GetState())
	//print("mlstate: ")
	//println(ss.MatchLengthsFSEDecodingTable.GetState())

	ss.Sequences = make([]Sequence, ss.Header.NumberOfSequences)

	for i := 0; i < ss.Header.NumberOfSequences; i++ {
		ss.Sequences[i], bits, err = ss.DecodeSequence(bitsrc)
		if err != nil {
			return bitsRead, err
		}
		bitsRead += bits

		//dont update on the last index.
		if i < ss.Header.NumberOfSequences-1 {
			bits, err := ss.LiteralLengthsFSEDecodingTable.NextState(bitsrc)
			bitsRead += bits
			if err != nil {
				return bitsRead, err
			}
			bits, err = ss.MatchLengthsFSEDecodingTable.NextState(bitsrc)
			bitsRead += bits
			if err != nil {
				return bitsRead, err
			}
			bits, err = ss.OffsetsFSEDecodingTable.NextState(bitsrc)
			bitsRead += bits
			if err != nil {
				return bitsRead, err
			}
		}
	}
	if bitsrc.BitsStillInStream() >= 0 {
		print("Bits originally in stream: ")
		println(len(bitsrc.Data) * 8)
		print("Bits still in stream: ")
		println(bitsrc.BitsStillInStream())
		//panic("Not all bits read")
		return bitsRead, errors.New("A")
	}
	return bitsRead, nil
}

type SymbolCompressionMode byte

const (
	SymbolCompressionModePredefined = SymbolCompressionMode(0)
	SymbolCompressionModeRLE        = SymbolCompressionMode(1)
	SymbolCompressionModeCompressed = SymbolCompressionMode(2)
	SymbolCompressionModeRepeat     = SymbolCompressionMode(3)
)

type SequencesSectionHeader struct {
	NumberOfSequences  int
	LiteralsLengthMode SymbolCompressionMode
	OffsetsMode        SymbolCompressionMode
	MatchLengthsMode   SymbolCompressionMode

	BytesUsedByHeader int
}

func (ssh *SequencesSectionHeader) DecodeSymbolCompressionModes(raw byte) {
	ssh.LiteralsLengthMode = SymbolCompressionMode(raw >> 6)
	ssh.OffsetsMode = SymbolCompressionMode(raw>>4) & 0x3
	ssh.MatchLengthsMode = SymbolCompressionMode(raw>>2) & 0x3
}

func (ssh *SequencesSectionHeader) EnoughBytesForNumberOfSequences(raw []byte) bool {
	if len(raw) == 0 {
		return false
	}
	return (raw[0] < 128 && len(raw) >= 1) || (raw[0] < 255 && len(raw) >= 2) || (raw[0] == 255 && len(raw) >= 3)
}

func (ssh *SequencesSectionHeader) BytesNeededForNumberOfSequences(raw byte) int {
	if raw < 128 {
		return 1
	}
	if raw < 255 {
		return 2
	}
	if raw == 255 {
		return 3
	}
	panic("Never should reach here")
}

//assumes the case raw[0] == 0 is already handled
func (ssh *SequencesSectionHeader) DecodeNumberOfSequences(raw []byte) (int, error) {
	if raw[0] < 128 {
		ssh.NumberOfSequences = int(raw[0])
		return 1, nil
	}
	if raw[0] < 255 {
		ssh.NumberOfSequences = ((int(raw[0] - 128)) << 8) + int(raw[1])
		return 2, nil
	}
	if raw[0] == 255 {
		ssh.NumberOfSequences = int(raw[1]) + (int(raw[2]) << 8) + 0x7F00
		return 3, nil
	}
	return 0, nil
}

func (ss *SequencesSection) DecodeTables(source *bufio.Reader, previousBlock *Block) (int, error) {
	bytesUsed := 0

	switch ss.Header.LiteralsLengthMode {
	case SymbolCompressionModePredefined:
		bytesUsed += 0
		ss.LiteralLengthsFSEDecodingTable = fse.BuildLiteralLengthsTable()
	case SymbolCompressionModeRLE:
		//read the byte that should be repeated
		b, err := source.ReadByte()
		if err != nil {
			return bytesUsed, err
		}
		bytesUsed++
		rdc := RepeatingDecodingTable(b)
		ss.LiteralLengthsFSEDecodingTable = &rdc
	case SymbolCompressionModeRepeat:
		ss.LiteralLengthsFSEDecodingTable = previousBlock.Sequences.LiteralLengthsFSEDecodingTable
		if previousBlock.Sequences.LiteralLengthsFSEDecodingTable == nil {
			panic("AA")
		}
	case SymbolCompressionModeCompressed:
		fset := fse.FSETable{}
		bytesread, err := fset.ReadTabledescriptionFromBitstream(source)
		if err != nil {
			return bytesUsed, err
		}
		bytesUsed += bytesread
		fset.BuildDecodingTable(fse.LiteralLengthBaseValueTranslation[:], fse.LiteralLengthExtraBits[:])
		ss.LiteralLengthsFSEDecodingTable = &fset
	}

	switch ss.Header.OffsetsMode {
	case SymbolCompressionModePredefined:
		bytesUsed += 0
		ss.OffsetsFSEDecodingTable = fse.BuildOffsetTable()
	case SymbolCompressionModeRLE:
		//read the byte that should be repeated
		b, err := source.ReadByte()
		if err != nil {
			return bytesUsed, err
		}
		bytesUsed++
		rdc := RepeatingDecodingTable(b)
		ss.OffsetsFSEDecodingTable = &rdc
	case SymbolCompressionModeRepeat:
		//TODO copy right table from previous block
		panic("Not implemented")
	case SymbolCompressionModeCompressed:
		fset := fse.FSETable{}
		bytesread, err := fset.ReadTabledescriptionFromBitstream(source)
		if err != nil {
			return bytesUsed, err
		}
		bytesUsed += bytesread
		fset.BuildDecodingTable(nil, nil)
		ss.OffsetsFSEDecodingTable = &fset
	}

	switch ss.Header.MatchLengthsMode {
	case SymbolCompressionModePredefined:
		bytesUsed += 0
		ss.MatchLengthsFSEDecodingTable = fse.BuildMatchLengthsTable()
	case SymbolCompressionModeRLE:
		bytesUsed++
		//read the byte that should be repeated
		_, err := source.ReadByte()
		if err != nil {
			return bytesUsed, err
		}
		//TODO represent rle table somehow
	case SymbolCompressionModeRepeat:
		ss.MatchLengthsFSEDecodingTable = previousBlock.Sequences.MatchLengthsFSEDecodingTable
	case SymbolCompressionModeCompressed:
		fset := fse.FSETable{}
		bytesread, err := fset.ReadTabledescriptionFromBitstream(source)
		if err != nil {
			return bytesUsed, err
		}
		bytesUsed += bytesread
		fset.BuildDecodingTable(fse.MatchLengthBaseValueTranslation[:], fse.MatchLengthsExtraBits[:])
		ss.MatchLengthsFSEDecodingTable = &fset
	}

	return bytesUsed, nil
}

func (ss *SequencesSection) DecodeNextSequenceSection(source *bufio.Reader, bytesLeftInBlock int, previousBlock *Block) error {
	//read sequence section
	var err error

	bytesUsedInHeader := 0

	var buf [3]byte //maximum 3 byte
	buf[0], err = source.ReadByte()
	if err != nil {
		return err
	}

	bytesNeededForNOS := ss.Header.BytesNeededForNumberOfSequences(buf[0])
	bytesReadForNOS, err := io.ReadFull(source, buf[1:bytesNeededForNOS])
	if err != nil {
		return err
	}
	if bytesReadForNOS != bytesNeededForNOS-1 {
		panic("Not enough bytes read to decode NumberOfSequences")
	}

	bytesUsedByNOS, err := ss.Header.DecodeNumberOfSequences(buf[:bytesNeededForNOS])
	bytesUsedInHeader += bytesUsedByNOS
	if err != nil {
		panic(err.Error())
	}
	if buf[0] == 0 {
		//the data in the literals section is the actual data, no sequences have been written
		ss.Header.BytesUsedByHeader = 1
		return nil
	}

	//print("Number of Sequences: ")
	//println(ss.Header.NumberOfSequences)

	//read the byte needed for symbol compression modes
	buf[0], err = source.ReadByte()
	if err != nil {
		return err
	}
	bytesUsedInHeader++

	ss.Header.DecodeSymbolCompressionModes(buf[0])

	bytesUsedByTables, err := ss.DecodeTables(source, previousBlock)
	if err != nil {
		return err
	}
	bytesUsedInHeader += bytesUsedByTables

	needed := bytesLeftInBlock - bytesUsedInHeader
	ss.Data = make([]byte, needed)

	if bytesLeftInBlock != bytesUsedInHeader+needed {
		panic("Corrupted lenghtes somewhere")
	}

	ss.Header.BytesUsedByHeader = bytesUsedInHeader

	//read the rest of the data. It contains a bitsream that needs to be read "backwards" it needs to be read in full before
	//it can be processed
	n, err := io.ReadFull(source, ss.Data)

	if err != nil {
		print("Error while reading sequence section data. Still need: ")
		println(needed - n)
		return err
	}

	if n != needed {
		panic("Wrong number of bytes have been read")
	}

	if bytesUsedInHeader+needed != bytesLeftInBlock {
		panic("Not enough bytes used by sequence block")
	}

	//ss.Data should now only include the bitsream containing the sequences
	bits, err := ss.DecodeSequences()
	if err != nil {
		return err
	}

	bytesUsed := bits / 8
	if bits%8 != 0 {
		bytesUsed++
	}

	if bytesUsed != len(ss.Data) {
		print(bytesUsed)
		print("/")
		println(len(ss.Data))
		panic("Not all bytes used?")
	}

	return nil
}
