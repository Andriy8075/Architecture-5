package datastore

import (
	"bufio"
	"bytes"
	"testing"
)

func TestEntry_Encode(t *testing.T) {
	original := entry{
		key:   "key",
		value: "value",
	}

	encoded := original.Encode()

	var decoded entry
	decoded.Decode(encoded)

	if decoded.key != original.key {
		t.Error("incorrect key")
	}
	if decoded.value != original.value {
		t.Error("incorrect value")
	}
	if decoded.hash != original.hash {
		t.Error("hash mismatch")
	}
}

func TestReadValue(t *testing.T) {
	original := entry{"key", "test-value", [20]byte{}}

	encoded := original.Encode()

	var decoded entry
	decoded.Decode(encoded)

	if decoded.key != original.key {
		t.Error("key mismatch")
	}
	if decoded.value != original.value {
		t.Error("value mismatch")
	}
	if decoded.hash != original.hash {
		t.Error("hash mismatch")
	}

	decoded2 := entry{}
	n, err := decoded2.DecodeFromReader(bufio.NewReader(bytes.NewReader(encoded)))
	if err != nil {
		t.Fatal(err)
	}

	if decoded2.key != original.key {
		t.Error("key mismatch (DecodeFromReader)")
	}
	if decoded2.value != original.value {
		t.Error("value mismatch (DecodeFromReader)")
	}
	if decoded2.hash != original.hash {
		t.Error("hash mismatch (DecodeFromReader)")
	}

	if n != len(encoded) {
		t.Errorf("DecodeFromReader() read %d bytes, expected %d", n, len(encoded))
	}
}
