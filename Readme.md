# Sparkzstd
This is a decompressor for the Zstandard compression format [Original Documentation](https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md)

This is still work in progress but it can already decompress some files correctly.
While it is still WIP I would actually call this at least in an alpha stage. It can decompress many files correctly and only weird edge cases come up. Those have not yet failed silently and giving corrupted output but rather fail at detection so you will most likely not end up with wrongly decompressed files.

## If you want to help test this
I'd love some others to test this library. The workflow I use is:
1. Since this is only a decompressor you still need the original zstd to compress your test file
2. Compress any file you want (for example tar (without compression) some directories you have and compress the result with zstd)
3. Use the main.go in cmd/sparkzstd to compare the output of the reader with the original file

The main.go for now just takes a list of pathes to .zst files and checks if the decompressed output matches byte for byte the content of the file with the same path but without .zst extension. 
eg: `go run cmd/sparkzstd/main.go ../testdata/pi.txt.zst`
checks the decompression of ../testdata/pi.txt.zst against ../testdata/pi.txt

If you'd like I would be glad to add your results to the list below.

## What are the goals of this project
Well mainly I had some time on my hands and wanted to write something that might be useful to someone out there.
The goal is to provide a io.Reader compatible API for reading zstd encoded data from a provided io.Reader.

## Where do I find stuff
1. Frame/Block/Literals/Sequences and their decoding is in /structure (Some HeaderDecoding is happening in the /decompression/framedecompressor.go)
2. Actual decompression aka. SequenceExecution is in /decompression/sequence_execution.go and /decompression/ringbuffer.go
3. FSE related stuff like predefined tables etc. are in /fse/predefined
4. Helpers for operations that need to read bits out of a bitstream or a reversed bitstream are located in /bitstream

## What is still missing
Generally all concepts of the Format have been implemented and are working (to a degree, some subtle bugs are still there) except dictionary support.
1. Dictionary parsing
2. Checksum calculation
1. Good benchmarks
2. Better doc
3. More bugs (I do have some unit tests and did some manual testing but you know...)

## Tracking of tests I did as of yet
I am testing this on a few files right know
### Working
1. A simple .jpg which I cant upload here for copyright reasons. This already decodes correctly but it is also just 36kb big
1. (FIXED. Does now decode correctly) A ubuntu 18.04.2-live-server-amd64.iso with md5sum: fcbcc756a1aa5314d52e882067c4ca6a. This decodes almost correctly. The result has the correct length but differs in about 400 bytes in different locations
1. Tested all files from the Canterbury corpus from here http://corpus.canterbury.ac.nz/descriptions/#cantrbry . They decompress correctly
1. Tested all (the one pi.txt) files from the Miscellaneous corpus from here http://corpus.canterbury.ac.nz/descriptions/#misc . They decompress correctly
1. A bigger file that klauspost (see https://github.com/klauspost/compress/tree/zstd-decoder/zstd) uses to test his implementation decodes correctly. It had an edge case that I didnt account for. So thanks to Klaus for unveiling that bug!

### Not working
1. Another larger file (tar archive of some parts of my $HOME which I cant upload here) wont decompress. (Probably) At some point the decoder doesnt read the correct amount of bytes (which is unlikely because I check in many places for correctness of amounts read/decoded etc). It finds a block with the "reserved" block type 3. I tested just discarding the block but that just fails at the next block.
2. Most of the added decodecorpus files dont decompress. I will apparently need to hunt bugs in my huffman implementation.

## Other Libaries
1. Some work has been done here towards a pure go implementation: https://github.com/klauspost/compress/tree/zstd-decoder/zstd Sadly I didnt find the project before I Started on this one.
2. A wuff implementation is WIP here (but wuff doesnt generate Go correctly yet) https://github.com/mvdan/zstd
3. A cgo binding to zsdt can be found here (which is needed if you want to compress stuff and not just decompress): https://github.com/DataDog/zstd 

