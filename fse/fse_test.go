package fse

import (
	"testing"
)

//copied from github.com/facebook/zstd
var expectedLLDecodingTable = [64]FSETableEntry{
	{0, 0, 4, 0}, {16, 0, 4, 0},
	{32, 0, 5, 1}, {0, 0, 5, 3},
	{0, 0, 5, 4}, {0, 0, 5, 6},
	{0, 0, 5, 7}, {0, 0, 5, 9},
	{0, 0, 5, 10}, {0, 0, 5, 12},
	{0, 0, 6, 14}, {0, 1, 5, 16},
	{0, 1, 5, 20}, {0, 1, 5, 22},
	{0, 2, 5, 28}, {0, 3, 5, 32},
	{0, 4, 5, 48}, {32, 6, 5, 64},
	{0, 7, 5, 128}, {0, 8, 6, 256},
	{0, 10, 6, 1024}, {0, 12, 6, 4096},
	{32, 0, 4, 0}, {0, 0, 4, 1},
	{0, 0, 5, 2}, {32, 0, 5, 4},
	{0, 0, 5, 5}, {32, 0, 5, 7},
	{0, 0, 5, 8}, {32, 0, 5, 10},
	{0, 0, 5, 11}, {0, 0, 6, 13},
	{32, 1, 5, 16}, {0, 1, 5, 18},
	{32, 1, 5, 22}, {0, 2, 5, 24},
	{32, 3, 5, 32}, {0, 3, 5, 40},
	{0, 6, 4, 64}, {16, 6, 4, 64},
	{32, 7, 5, 128}, {0, 9, 6, 512},
	{0, 11, 6, 2048}, {48, 0, 4, 0},
	{16, 0, 4, 1}, {32, 0, 5, 2},
	{32, 0, 5, 3}, {32, 0, 5, 5},
	{32, 0, 5, 6}, {32, 0, 5, 8},
	{32, 0, 5, 9}, {32, 0, 5, 11},
	{32, 0, 5, 12}, {0, 0, 6, 15},
	{32, 1, 5, 18}, {32, 1, 5, 20},
	{32, 2, 5, 24}, {32, 2, 5, 28},
	{32, 3, 5, 40}, {32, 4, 5, 48},
	{0, 16, 6, 65536}, {0, 15, 6, 32768},
	{0, 14, 6, 16384}, {0, 13, 6, 8192},
}

func TestBuilding(t *testing.T) {
	fset := FSETable{Values: make(map[int]int64), AccuracyLog: LiteralLengthDefaultAccuracyLog}
	for idx, prob := range LiteralLengthDefaultDistributions {
		fset.Values[idx] = int64(prob + 1) //value == probability+1
	}

	fset.BuildDecodingTable(llBaseValueTranslation)

	println("IDX\tSymbol\tnbBits\tBase\tnbAdd")

	for idx, entry := range expectedLLDecodingTable {
		newentry := fset.DecodingTable[idx]
		if newentry.Baseline != entry.Baseline {
			t.Errorf("Baselines didnt match at index: %d", idx)
		}
		if newentry.NumberOfBits != entry.NumberOfBits {
			t.Errorf("NumberOfBits didnt match at index: %d", idx)
		}
		if newentry.Symbol != entry.Symbol {
			t.Errorf("Symbols didnt match at index: %d", idx)
		}
	}

	//pretty print the decoding table similarly to the table in the doc for human readable checking of values
	//for idx := 0; idx < len(fset.DecodingTable); idx++ {
	//	entry := fset.DecodingTable[idx]
	//
	//	print(idx)
	//	print(" :\t")
	//	print(entry.Symbol)
	//	print(" ,\t")
	//	print(entry.NumberOfBits)
	//	print(" ,\t")
	//	print(entry.Baseline)
	//	print(" ,\t")
	//	print(entry.NumberOfAdditionalBits)
	//	print("\n")
	//}
	//panic("a")
}
