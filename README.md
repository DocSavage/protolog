# protolog

Package protolog implements a simple file format for a sequence of blobs
with ability to store a message type with each blob as well as a checksum.
It is intended for logging protobuf messages.  Unlike other formats, it
tries to be simple, so writing a reader/writer in other languages is
trivial.  This design is a modified form of Eric Lesh's recordio Go
implementation (github.com/eclesh/recordio).  This package swaps out a single
varint header for fixed size headers with support for a uint16 ID of the message 
type and a CRC-32C checksum.

## Example: reading with a Scanner
	const fooTypeID uint16 = 13 // ids that tell us type of message

	f, _ := os.Open("data.log")
	defer f.Close()
	scanner := protolog.NewScanner(f)
	for scanner.Scan() {
		data := scanner.Bytes()
		switch scanner.TypeID() {
			case fooTypeID:
				var foo myFooProtobufType
				foo.Unmarshal(data)
			// Check other types
		}
	}
	if err := scanner.Error(); err != nil {
		// Do error handling
	}

## Example: reading with a Reader
	const fooTypeID uint16 = 13 // ids that tell us type of message

	f, _ := os.Open("data.log")
	f.Close()
	r := protolog.NewReader(f)
	for {
		data, typeID, err := r.Next()
		if err == io.EOF {
			break
		}
		switch typeID {
			case fooTypeID:
				var foo myFooProtobufType
				foo.Unmarshal(data)
			// Check other types
		}
	}

## Example: writing one type of message
	const fooTypeID uint16 = 13 // ids that tell us type of message

	f, _ := os.Create("data.log")
	w := protolog.NewTypedWriter(fooTypeID, f)

	data, _ := myFooProtobufObj.Marshal()
	w.Write(data)
	data, _ = my2ndFooProtobufObj.Marshal()
	w.Write(data)
	
	f.Close()

## Example: writing different messages
	const (
		fooTypeID uint16 = 13
		barTypeID uint16 = 14
	)

	f, _ := os.Create("data.log")
	w := protolog.NewMultiTypeWriter(f)

	typebuf := make([]byte, 2)

	serialization, _ := myFooProtobufObj.Marshal()
	w.Write(fooTypeID, serialization)

	serialization, _ = myBarProtobufObj.Marshal()
	w.Write(barTypeID, serialization)

	f.Close()

## File Format

Header 1 (10 bytes)
    Message # Bytes excluding fixed header (uint32, 4 bytes)
    Checksum (4 bytes)
    Message Type ID (uint16, 2 bytes)
Binary Data 1 (variable # bytes)
...
Header N (10 bytes)
Binary Data N (variable # bytes)

## License

Copyright (C) 2019 Bill Katz

Permission is hereby granted, free of charge, to any person obtaining
a copy of this software and associated documentation files (the
"Software"), to deal in the Software without restriction, including
without limitation the rights to use, copy, modify, merge, publish,
distribute, sublicense, and/or sell copies of the Software, and to
permit persons to whom the Software is furnished to do so, subject to
the following conditions:

The above copyright notice and this permission notice shall be
included in all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE
LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION
OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION
WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
