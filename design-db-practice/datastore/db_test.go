package datastore

import (
	"os"
	"strings"
	"testing"
)

func TestDb(t *testing.T) {
	tmp := t.TempDir()
	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	pairs := [][]string{
		{"k1", "v1"},
		{"k2", "v2"},
		{"k3", "v3"},
		{"k2", "v2.1"},
	}

	t.Run("put/get", func(t *testing.T) {
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
			value, err := db.Get(pair[0])
			if err != nil {
				t.Errorf("Cannot get %s: %s", pairs[0], err)
			}
			if value != pair[1] {
				t.Errorf("Bad value returned expected %s, got %s", pair[1], value)
			}
		}
	})

	t.Run("file growth", func(t *testing.T) {
		sizeBefore, err := db.Size()
		if err != nil {
			t.Fatal(err)
		}
		for _, pair := range pairs {
			err := db.Put(pair[0], pair[1])
			if err != nil {
				t.Errorf("Cannot put %s: %s", pairs[0], err)
			}
		}
		sizeAfter, err := db.Size()
		if err != nil {
			t.Fatal(err)
		}
		if sizeAfter <= sizeBefore {
			t.Errorf("Size does not grow after put (before %d, after %d)", sizeBefore, sizeAfter)
		}
	})

	t.Run("new db process", func(t *testing.T) {
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
		db, err = Open(tmp)
		if err != nil {
			t.Fatal(err)
		}

		uniquePairs := make(map[string]string)
		for _, pair := range pairs {
			uniquePairs[pair[0]] = pair[1]
		}

		for key, expectedValue := range uniquePairs {
			value, err := db.Get(key)
			if err != nil {
				t.Errorf("Cannot get %s: %s", key, err)
			}
			if value != expectedValue {
				t.Errorf("Get(%q) = %q, wanted %q", key, value, expectedValue)
			}
		}
	})
}

func TestSegmentSplitting(t *testing.T) {
	tmp := t.TempDir()

	origSize := maxSegmentSize
	maxSegmentSize = 7
	defer func() { maxSegmentSize = origSize }()

	db, err := Open(tmp)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	keys := []string{"a", "b", "c", "d", "e", "f", "g"}
	for _, k := range keys {
		if err := db.Put(k, strings.Repeat("x", 30)); err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	files, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatalf("cannot read dir: %v", err)
	}

	var segments int
	for _, f := range files {
		if strings.HasPrefix(f.Name(), "segment-") && strings.HasSuffix(f.Name(), ".db") {
			segments++
		}
	}

	if segments < 2 {
		t.Errorf("Expected multiple segments, got %d", segments)
	}
}

func TestMergeSegments(t *testing.T) {
	tmpDir := t.TempDir()

	maxSegmentSize = 20

	db, err := Open(tmpDir)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer db.Close()

	for i := 0; i < 10; i++ {
		key := "key" + string(rune('A'+i))
		value := "value" + string(rune('A'+i))
		err := db.Put(key, value)
		if err != nil {
			t.Fatalf("Put failed: %v", err)
		}
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	segCount := 0
	for _, f := range files {
		if f.Name()[0:7] == "segment" {
			segCount++
		}
	}
	if segCount < 2 {
		t.Fatalf("Expected more than 1 segment, we have : %d", segCount)
	}
	
	err = db.MergeSegments()
	if err != nil {
		t.Fatalf("MergeSegments failed: %v", err)
	}

	files, err = os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}
	segCount = 0
	for _, f := range files {
		if f.Name()[0:7] == "segment" {
			segCount++
		}
	}
	if segCount != 1 {
		t.Fatalf("After merging have more than 1 segment, we have: %d", segCount)
	}

	for i := 0; i < 10; i++ {
		key := "key" + string(rune('A'+i))
		want := "value" + string(rune('A'+i))
		got, err := db.Get(key)
		if err != nil {
			t.Fatalf("Get(%s) failed: %v", key, err)
		}
		if got != want {
			t.Errorf("Get(%s) = %s, want %s", key, got, want)
		}
	}
}
