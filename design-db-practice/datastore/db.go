package datastore

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const segmentFileFormat = "segment-%06d.db"

var maxSegmentSize int64 = 10

var ErrNotFound = fmt.Errorf("record does not exist")

type recordLocation struct {
	segmentID int
	offset    int64
}

type hashIndex map[string]recordLocation

type Db struct {
	dir           string
	currentFile   *os.File
	currentOffset int64
	currentID     int
	index         hashIndex
}

func Open(dir string) (*Db, error) {
	db := &Db{
		dir:   dir,
		index: make(hashIndex),
	}

	if err := db.loadSegments(); err != nil {
		return nil, err
	}

	return db, nil
}

func (db *Db) loadSegments() error {
	entries, err := os.ReadDir(db.dir)
	if err != nil {
		return err
	}

	var segments []int
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "segment-") && strings.HasSuffix(entry.Name(), ".db") {
			idStr := strings.TrimSuffix(strings.TrimPrefix(entry.Name(), "segment-"), ".db")
			id, err := strconv.Atoi(idStr)
			if err != nil {
				continue
			}
			segments = append(segments, id)
		}
	}
	sort.Ints(segments)

	for _, id := range segments {
		if err := db.loadSegment(id); err != nil {
			return err
		}
		db.currentID = id
	}

	return db.openCurrentSegment()
}

func (db *Db) loadSegment(id int) error {
	filePath := filepath.Join(db.dir, fmt.Sprintf(segmentFileFormat, id))
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	reader := bufio.NewReader(f)
	var offset int64
	for {
		var e entry
		n, err := e.DecodeFromReader(reader)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("loadSegment error: %w", err)
		}
		db.index[e.key] = recordLocation{segmentID: id, offset: offset}
		offset += int64(n)
	}
	return nil
}

func (db *Db) openCurrentSegment() error {
	path := filepath.Join(db.dir, fmt.Sprintf(segmentFileFormat, db.currentID))
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	db.currentFile = f

	info, err := f.Stat()
	if err != nil {
		return err
	}
	db.currentOffset = info.Size()
	return nil
}

func (db *Db) Close() error {
	if db.currentFile != nil {
		return db.currentFile.Close()
	}
	return nil
}

func (db *Db) Put(key, value string) error {
	e := entry{key: key, value: value}
	data := e.Encode()

	if db.currentOffset+int64(len(data)) > maxSegmentSize {
		// Закрити поточний файл
		if err := db.currentFile.Close(); err != nil {
			return err
		}
		db.currentID++
		// Відкрити новий сегмент
		if err := db.openCurrentSegment(); err != nil {
			return err
		}

		// Якщо сегментів стало більше ніж 3 — злиття
		files, _ := os.ReadDir(db.dir)
		count := 0
		for _, f := range files {
			if strings.HasPrefix(f.Name(), "segment-") && strings.HasSuffix(f.Name(), ".db") {
				count++
			}
		}
		if count > 3 {
			_ = db.MergeSegments()
		}
	}

	n, err := db.currentFile.Write(data)
	if err != nil {
		return err
	}

	db.index[key] = recordLocation{
		segmentID: db.currentID,
		offset:    db.currentOffset,
	}

	db.currentOffset += int64(n)
	return nil
}

func (db *Db) Get(key string) (string, error) {
	loc, ok := db.index[key]
	if !ok {
		return "", ErrNotFound
	}
	filePath := filepath.Join(db.dir, fmt.Sprintf(segmentFileFormat, loc.segmentID))
	f, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	_, err = f.Seek(loc.offset, io.SeekStart)
	if err != nil {
		return "", err
	}
	var e entry
	_, err = e.DecodeFromReader(bufio.NewReader(f))
	if err != nil {
		return "", err
	}
	return e.value, nil
}

func (db *Db) Size() (int64, error) {
	var total int64
	entries, err := os.ReadDir(db.dir)
	if err != nil {
		return 0, err
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "segment-") {
			info, err := os.Stat(filepath.Join(db.dir, entry.Name()))
			if err != nil {
				return 0, err
			}
			total += info.Size()
		}
	}
	return total, nil
}

func (db *Db) MergeSegments() error {
	tmpPath := filepath.Join(db.dir, "merged.tmp")
	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	writer := bufio.NewWriter(tmpFile)

	latest := make(map[string]string)

	entries, err := os.ReadDir(db.dir)
	if err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}

	var segmentFiles []string
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "segment-") && strings.HasSuffix(entry.Name(), ".db") {
			segmentFiles = append(segmentFiles, filepath.Join(db.dir, entry.Name()))
		}
	}
	sort.Strings(segmentFiles) // гарантований порядок

	for _, path := range segmentFiles {
		f, err := os.Open(path)
		if err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return err
		}
		r := bufio.NewReader(f)
		for {
			var e entry
			_, err := e.DecodeFromReader(r)
			if err == io.EOF {
				break
			}
			if err != nil {
				f.Close()
				tmpFile.Close()
				os.Remove(tmpPath)
				return err
			}
			latest[e.key] = e.value
		}
		f.Close()
	}

	var offset int64
	newIndex := make(hashIndex)

	for key, value := range latest {
		e := entry{key: key, value: value}
		data := e.Encode()
		if _, err := writer.Write(data); err != nil {
			tmpFile.Close()
			os.Remove(tmpPath)
			return err
		}
		newIndex[key] = recordLocation{
			segmentID: 0,
			offset:    offset,
		}
		offset += int64(len(data))
	}

	if err := writer.Flush(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		tmpFile.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := tmpFile.Close(); err != nil {
		os.Remove(tmpPath)
		return err
	}

	// Закрити старий файл
	if db.currentFile != nil {
		db.currentFile.Close()
	}

	// Видалити старі сегменти
	for _, path := range segmentFiles {
		os.Remove(path)
	}

	// Перейменувати тимчасовий файл у новий сегмент
	finalPath := filepath.Join(db.dir, fmt.Sprintf(segmentFileFormat, 0))
	if err := os.Rename(tmpPath, finalPath); err != nil {
		return err
	}

	// Відкрити його як поточний
	db.currentID = 0
	db.index = newIndex
	return db.openCurrentSegment()
}
