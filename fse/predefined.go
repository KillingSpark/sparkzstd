package fse

var LiteralLengthDefaultAccuracyLog = 6

var LiteralLengthBaseValueTranslation = [36]int{
	0, 1, 2, 3, 4, 5, 6, 7,
	8, 9, 10, 11, 12, 13, 14, 15,
	16, 18, 20, 22, 24, 28, 32, 40,
	48, 64, 0x80, 0x100, 0x200, 0x400, 0x800,
	0x1000, 0x2000, 0x4000, 0x8000, 0x10000,
}
var LiteralLengthDefaultDistributions = [36]int{
	4, 3, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 2, 1, 1, 1,
	2, 2, 2, 2, 2, 2, 2, 2, 2, 3, 2, 1, 1, 1, 1, 1,
	-1, -1, -1, -1,
}

var LiteralLengthExtraBits = [36]byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1,
	1, 1, 2, 2, 3, 3, 4, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

func BuildLiteralLengthsTable() *FSETable {
	fset := FSETable{Values: make(map[int]int64), AccuracyLog: LiteralLengthDefaultAccuracyLog}
	for idx, prob := range LiteralLengthDefaultDistributions {
		fset.Values[idx] = int64(prob + 1) //value == probability+1
	}

	fset.BuildDecodingTable(LiteralLengthBaseValueTranslation[:], LiteralLengthExtraBits[:])
	return &fset
}

//#####

var MatchLengthDefaultAccuracyLog = 6

var MatchLengthBaseValueTranslation = [53]int{
	3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16,
	17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30,
	31, 32, 33, 34, 35, 37, 39, 41, 43, 47, 51, 59, 67, 83,
	99, 131, 259, 515, 1027, 2051, 4099, 8195, 16387, 32771, 65539}

var MatchLengthDefaultDistribution = [53]int{
	1, 4, 3, 2, 2, 2, 2, 2, 2, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, 1, -1, -1, -1, -1, -1, -1, -1}

var MatchLengthsExtraBits = [53]byte{
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1,
	2, 2, 3, 3, 4, 4, 5, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

func BuildMatchLengthsTable() *FSETable {
	fset := FSETable{Values: make(map[int]int64), AccuracyLog: MatchLengthDefaultAccuracyLog}
	for idx, prob := range MatchLengthDefaultDistribution {
		fset.Values[idx] = int64(prob + 1) //value == probability+1
	}

	fset.BuildDecodingTable(MatchLengthBaseValueTranslation[:], MatchLengthsExtraBits[:])
	return &fset
}

//######

var OffsetDefaultAccuracyLog = 5

var OffsetDefaultDistribution = [29]int{
	1, 1, 1, 1, 1, 1, 2, 2, 2, 1, 1, 1, 1, 1, 1,
	1, 1, 1, 1, 1, 1, 1, 1, 1, -1, -1, -1, -1, -1}

func BuildOffsetTable() *FSETable {
	fset := FSETable{Values: make(map[int]int64), AccuracyLog: OffsetDefaultAccuracyLog}
	for idx, prob := range OffsetDefaultDistribution {
		fset.Values[idx] = int64(prob + 1) //value == probability+1
	}

	fset.BuildDecodingTable(nil, nil)
	return &fset
}
