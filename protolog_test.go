package protolog

import (
	"bytes"
	"io"
	"testing"
)

func TestEmpty(t *testing.T) {
	buf := new(bytes.Buffer)
	r := NewReader(buf)
	if _, _, err := r.Next(); err != io.EOF {
		t.Fatalf("got %v, expected %v", err, io.EOF)
	}
}

const (
	FooTypeID uint16 = iota
	BarTypeID
	BazTypeID
)

func TestOrder(t *testing.T) {
	testValues := []string{
		"first",
		"second",
		"third",
	}

	buf := new(bytes.Buffer)

	w := NewTypedWriter(BazTypeID, buf)
	for _, val := range testValues {
		_, err := w.Write([]byte(val))
		if err != nil {
			t.Fatalf("unexpected error %v for value %q\n", err, val)
		}
	}

	r := NewReader(buf)
	for _, expected := range testValues {
		typeID, received, err := r.Next()
		if err != nil {
			t.Fatalf("unexpected error '%v' for value %q\n", err, expected)
		}
		if len(received) != len(expected) {
			t.Fatalf("expected length %d, got %d for value %q\n", len(expected), len(received), expected)
		}
		if string(received) != expected {
			t.Fatalf("expected %q, got %q\n", expected, received)
		}
		if typeID != BazTypeID {
			t.Fatalf("expected type ID %d got %d\n", BazTypeID, typeID)
		}
	}
}

func TestScanner(t *testing.T) {
	testValues := [][]byte{
		[]byte("first"),
		[]byte("second"),
		[]byte("third"),
	}

	buf := new(bytes.Buffer)

	w := NewMultiTypedWriter(buf)
	typeID := FooTypeID
	for _, val := range testValues {
		if _, err := w.Write(typeID, val); err != nil {
			t.Fatalf("unexpected error %v for value %q\n", err, val)
		}
		typeID++
	}

	scanner := NewScanner(buf)
	typeID = FooTypeID
	i := 0
	for scanner.Scan() {
		if i >= len(testValues) {
			t.Fatalf("scanner scanned for %d elements but only %d exist\n", i, len(testValues))
		}
		expected := testValues[i]
		data := scanner.Bytes()
		if !bytes.Equal(data, expected) {
			t.Fatalf("expected value %q, got %q instead\n", string(expected), string(data))
		}
		id := scanner.TypeID()
		if id != typeID {
			t.Fatalf("expected type id %d, got %d instead\n", typeID, id)
		}
		i++
		typeID++
	}
	if err := scanner.Error(); err != nil {
		t.Fatalf("unexpected error %v during scan\n", err)
	}
}
