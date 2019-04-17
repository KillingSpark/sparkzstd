# Sparkzstd
This is a decompressor for the Zstandard compression format (Original Documentation[https://github.com/facebook/zstd/blob/dev/doc/zstd_compression_format.md]

This is still work in progress but it can already decompress some files correctly.

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
1. Good benchmarks
2. Better doc
3. More bugs (I do have some unit tests and did some manual testing but you know...)

## Which bugs are known
I am testing this on a few files right know
1. A simple .jpg which I cant upload here for copyright reasons. 
This already decodes correctly but it is also just 36kb big
2. A ubuntu 18.04.2-live-server-amd64.iso with md5sum: fcbcc756a1aa5314d52e882067c4ca6a. 
This decodes almost correctly. The result has the correct length but differs in about 400 bytes in different locations
3. Another larger file (tar archive of some parts of my $HOME which I cant upload here) wont decompress. (Probably) At some point the decoder doesnt read the correct amount of bytes.

## Other Libaries
1. Some work has been done here towards a pure go implementation: https://github.com/klauspost/compress/tree/zstd-decoder/zstd Sadly I didnt find the project before I Started on this one.
2. A wuff implementation is WIP here (but wuff doesnt generate Go correctly yet) https://github.com/mvdan/zstd
3. A cgo binding to zsdt can be found here (which is needed if you want to compress stuff and not just decompress): https://github.com/DataDog/zstd 

