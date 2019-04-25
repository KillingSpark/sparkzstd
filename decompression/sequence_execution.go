package decompression

import (
	"errors"
	"github.com/killingspark/sparkzsdt/structure"
)

var counter = 0
var totalOutput = 0

var ErrDidntCopyAllLiteralBytes = errors.New("Not enough bytes read to execute literals copy")

//ExecuteSequences is used after decoding to produce the actual decompressed content of the block
func (fd *FrameDecompressor) ExecuteSequences() error {

	for _, seq := range fd.CurrentBlock.Sequences.Sequences {

		//literals copy
		if seq.LiteralLength > 0 {
			lbuf := fd.literalsCopyBuf[:seq.LiteralLength]
			n, err := fd.CurrentBlock.Literals.Read(lbuf)

			if err != nil {
				return err
			}
			if n != seq.LiteralLength {
				return ErrDidntCopyAllLiteralBytes
			}

			err = fd.decodebuffer.Push(lbuf)
			if err != nil {
				return err
			}
		}

		//print(seq.LiteralLength)
		//print("/")
		//print(seq.MatchLength)
		//print("/")
		//println(seq.Offset)

		//offset & match
		offset := fd.nextOffset(seq) //updates offset history
		if seq.MatchLength > 0 {
			err := fd.decodebuffer.RepeatBeforeIndex(int(seq.MatchLength), int(offset))
			if err != nil {
				return err
			}
		}

		totalOutput += seq.LiteralLength
		totalOutput += seq.MatchLength
	}

	lastLiterals := fd.CurrentBlock.Literals.GetRest()
	err := fd.decodebuffer.Push(lastLiterals)
	if err != nil {
		return err
	}
	totalOutput += len(lastLiterals)

	return nil
}

func (fd *FrameDecompressor) nextOffset(seq structure.Sequence) int64 {
	var offset int64

	if seq.Offset <= 3 && seq.LiteralLength > 0 {
		switch seq.Offset {
		case 1:
			offset = fd.offsetHistory[0]
		case 2:
			offset = fd.offsetHistory[1]
			fd.offsetHistory[1] = fd.offsetHistory[0]
			fd.offsetHistory[0] = offset
		case 3:
			offset = fd.offsetHistory[2]
			fd.offsetHistory[2] = fd.offsetHistory[1]
			fd.offsetHistory[1] = fd.offsetHistory[0]
			fd.offsetHistory[0] = offset
		}
	} else {
		if seq.Offset <= 3 && seq.LiteralLength == 0 {
			switch seq.Offset {
			case 1:
				offset = fd.offsetHistory[1]
				fd.offsetHistory[1] = fd.offsetHistory[0]
				fd.offsetHistory[0] = offset
			case 2:
				offset = fd.offsetHistory[2]
				fd.offsetHistory[2] = fd.offsetHistory[1]
				fd.offsetHistory[1] = fd.offsetHistory[0]
				fd.offsetHistory[0] = offset
			case 3:
				offset = fd.offsetHistory[0] - 1
				fd.offsetHistory[2] = fd.offsetHistory[1]
				fd.offsetHistory[1] = fd.offsetHistory[0]
				fd.offsetHistory[0] = offset
			}
		} else {
			//STANDARD CASE
			if seq.Offset <= 3 {
				//keeping. Would be a programming error
				panic("Forgot some case?")
			}
			offset = int64(seq.Offset) - 3
			fd.offsetHistory[2] = fd.offsetHistory[1]
			fd.offsetHistory[1] = fd.offsetHistory[0]
			fd.offsetHistory[0] = offset
		}
	}

	return offset
}
