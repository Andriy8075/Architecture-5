package datastore

import (
	"bufio"
	"bytes"
	"crypto/sha1"
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

func TestEntry_HashChange(t *testing.T) {
	original := entry{"key", "value", [20]byte{}}
	original.hash = original.EncodeHash()

	modified := entry{"key", "value_modified", [20]byte{}}
	modified.hash = modified.EncodeHash()

	if bytes.Equal(original.hash[:], modified.hash[:]) {
		t.Error("hashes should differ when value changes")
	}
}

func TestEntry_EncodeDecodeHash(t *testing.T) {
	e := entry{key: "testkey", value: "testvalue"}
	encoded := e.Encode()

	// hash має оновитися після Encode
	expectedHash := sha1.Sum([]byte(e.key + e.value))
	if !bytes.Equal(e.hash[:], expectedHash[:]) {
		t.Fatalf("hash mismatch after Encode, got %x, want %x", e.hash, expectedHash)
	}

	var e2 entry
	e2.Decode(encoded)

	if e2.key != e.key {
		t.Errorf("key mismatch, got %s, want %s", e2.key, e.key)
	}
	if e2.value != e.value {
		t.Errorf("value mismatch, got %s, want %s", e2.value, e.value)
	}
	if !bytes.Equal(e2.hash[:], expectedHash[:]) {
		t.Errorf("hash mismatch after Decode, got %x, want %x", e2.hash, expectedHash)
	}
}

func TestEntry_HashChangesOnValueChange(t *testing.T) {
	e1 := entry{key: "key", value: "value"}
	e1.Encode()
	e2 := entry{key: "key", value: "value2"}
	e2.Encode()

	if bytes.Equal(e1.hash[:], e2.hash[:]) {
		t.Error("hash should differ if value changes")
	}
}
