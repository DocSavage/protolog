// Package protolog implements a simple file format for a sequence of blobs
// with ability to store a message type with each blob as well as a checksum.
// It is intended for logging protobuf messages.  Unlike other formats, it
// tries to be simple, so writing a reader/writer in other languages is
// trivial.  This design is a modified form of Eric Lesh's recordio Go
// implementation (github.com/eclesh/recordio).  It uses fixed size headers
// with support for a uint16 ID of the message type and a CRC-32C checksum.
// Each blob must be less than 4 GiB (2^32 bytes).
package protolog

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"math"
)

var (
	// ErrBadChecksum is an error returned when a bad checksum is detected.
	ErrBadChecksum = fmt.Errorf("bad checksum detected while reading data")
)

// Reader allows reading of checksummed binary blobs with optional uint16 type IDs.
type Reader struct {
	r      io.Reader // the reader
	buf    []byte    // the buffer
	bufcap uint32    // the capacity of the buffer
	hdr    header    // the header
}

type header struct {
	numBytes uint32
	checksum uint32
	typeID   uint16
}

var checksumTable = crc32.MakeTable(crc32.Castagnoli)

func readHeader(r io.Reader) (*header, error) {
	hdrbuf := make([]byte, 10)
	if _, err := io.ReadFull(r, hdrbuf); err != nil {
		return nil, err
	}
	hdr := header{
		numBytes: binary.LittleEndian.Uint32(hdrbuf[0:4]),
		checksum: binary.LittleEndian.Uint32(hdrbuf[4:8]),
		typeID:   binary.LittleEndian.Uint16(hdrbuf[8:10]),
	}
	return &hdr, nil
}

func writeRecord(w io.Writer, typeID uint16, data []byte) (int, error) {
	if uint64(len(data)) >= uint64(math.MaxUint32) {
		return 0, fmt.Errorf("cannot write data record exceeding 4 GiB in size")
	}
	hdrbuf := make([]byte, 10)
	numBytes := uint32(len(data))
	checksum := crc32.Checksum(data, checksumTable)
	binary.LittleEndian.PutUint32(hdrbuf[0:4], numBytes)
	binary.LittleEndian.PutUint32(hdrbuf[4:8], checksum)
	binary.LittleEndian.PutUint16(hdrbuf[8:10], typeID)
	n, err := w.Write(hdrbuf)
	if n != 10 {
		return n, fmt.Errorf("couldn't write record header, only %d of 10 bytes", n)
	}
	n, err = w.Write(data)
	if n != int(numBytes) {
		return n + int(numBytes), fmt.Errorf("only able to write %d of %d data bytes", n, numBytes)
	}
	return 10 + int(numBytes), err
}

// NewReader returns a new reader. If r doesn't implement
// io.ByteReader, it will be wrapped in a bufio.Reader.
func NewReader(r io.Reader) *Reader {
	if _, ok := r.(io.ByteReader); !ok {
		r = bufio.NewReader(r)
	}
	return &Reader{r: r}
}

// Next returns the next data record's type ID (if set) and data.
// It returns io.EOF if there are no more records.
func (r *Reader) Next() (uint16, []byte, error) {
	hdr, err := readHeader(r.r)
	if err != nil {
		return 0, nil, err
	}
	if hdr.numBytes > r.bufcap {
		r.buf = make([]byte, hdr.numBytes)
		r.bufcap = hdr.numBytes
	}
	_, err = io.ReadFull(r.r, r.buf[:hdr.numBytes])
	if err != nil {
		return 0, nil, err
	}
	checksum := crc32.Checksum(r.buf[:hdr.numBytes], checksumTable)
	if checksum != hdr.checksum {
		return 0, nil, ErrBadChecksum
	}
	return hdr.typeID, r.buf[:hdr.numBytes], nil
}

// A Scanner is a convenient method for reading records sequentially.
type Scanner struct {
	r       io.Reader // the reader
	err     error
	buf     []byte
	bufsize uint32
	bufcap  uint32
	hdr     *header
}

// NewScanner creates a new Scanner from reader r.
func NewScanner(r io.Reader) *Scanner {
	if _, ok := r.(io.ByteReader); !ok {
		r = bufio.NewReader(r)
	}
	return &Scanner{r: r}
}

// Scan chugs through the input record by record and stops at the first
// error or EOF.
func (s *Scanner) Scan() bool {
	var err error
	s.hdr, err = readHeader(s.r)
	if err != nil {
		s.err = err
		return false
	}
	s.bufsize = s.hdr.numBytes
	if s.hdr.numBytes > s.bufcap {
		s.buf = make([]byte, s.hdr.numBytes)
		s.bufcap = s.hdr.numBytes
	}
	_, err = io.ReadFull(s.r, s.buf[:s.hdr.numBytes])
	if err != nil {
		s.err = err
		return false
	}
	checksum := crc32.Checksum(s.buf[:s.hdr.numBytes], checksumTable)
	if checksum != s.hdr.checksum {
		log.Printf("expected %d, got %d\n", s.hdr.checksum, checksum)
		s.err = ErrBadChecksum
		return false
	}
	return true
}

// TypeID returns the optionally set type ID of the most recently scanned record.
func (s *Scanner) TypeID() uint16 {
	return s.hdr.typeID
}

// Bytes returns the data of the most recently scanned record. Subsequent calls may
// overwrite the returned data, so you must copy it if not using it immediately.
func (s *Scanner) Bytes() []byte {
	return s.buf[:s.bufsize]
}

// Error returns the most recent error or nil if the error was EOF.
func (s *Scanner) Error() error {
	if s.err == io.EOF {
		return nil
	}
	return s.err
}

// TypedWriter writes records that have a single optional type identifier
type TypedWriter struct {
	typeID uint16    // optional record type id
	w      io.Writer // the writer
}

// NewTypedWriter returns a new writer that uses the same record type.
func NewTypedWriter(recordType uint16, w io.Writer) *TypedWriter {
	return &TypedWriter{
		typeID: recordType,
		w:      w,
	}
}

// Write writes a data record associated with an optional type id.
func (w *TypedWriter) Write(data []byte) (int, error) {
	return writeRecord(w.w, w.typeID, data)
}

// MultiTypedWriter writes records that can have different optional type identifiers
type MultiTypedWriter struct {
	w io.Writer
}

// NewMultiTypedWriter returns a new writer that could have different record types.
func NewMultiTypedWriter(w io.Writer) *MultiTypedWriter {
	return &MultiTypedWriter{
		w: w,
	}
}

// Write writes a data record with a type identifier.
func (w *MultiTypedWriter) Write(typeID uint16, data []byte) (int, error) {
	return writeRecord(w.w, typeID, data)
}
