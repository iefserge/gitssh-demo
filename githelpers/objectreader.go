package githelpers

import (
	"fmt"
	"io"
)

type ObjectType int

const (
	ObjectInvalid  ObjectType = 0
	ObjectCommit   ObjectType = 1
	ObjectTree     ObjectType = 2
	ObjectBlob     ObjectType = 3
	ObjectTag      ObjectType = 4
	ObjectOfsDelta ObjectType = 6
	ObjectRefDelta ObjectType = 7
)

type ObjectHeader struct {
	ObjType  ObjectType
	ObjSize  int64
	OfsDelta int64
	RefDelta [20]byte
}

func CopyObjectHeader(r io.ByteReader, teeWriter io.Writer) (*ObjectHeader, error) {
	b, err := r.ReadByte()
	if err != nil {
		return nil, fmt.Errorf("reading object type: %w", err)
	}
	if _, err := teeWriter.Write([]byte{b}); err != nil {
		return nil, fmt.Errorf("writing object type: %w", err)
	}
	objType := ObjectType((int(b) >> 4) & 7)
	objSize := int64(b & 15)
	shift := 4
	for i := 1; (b & 0x80) != 0; i++ {
		b, err = r.ReadByte()
		if err != nil {
			return nil, fmt.Errorf("reading length byte: %w", err)
		}
		if _, err := teeWriter.Write([]byte{b}); err != nil {
			return nil, fmt.Errorf("writing object type: %w", err)
		}
		objSize += int64(b&0x7f) << shift
	}
	res := ObjectHeader{
		ObjType: objType,
		ObjSize: objSize,
	}
	switch objType {
	case ObjectOfsDelta:
		res.OfsDelta, _, err = readDeltaBaseOffset(r, teeWriter)
		if err != nil {
			return nil, fmt.Errorf("reading ofs delta: %w", err)
		}
	case ObjectRefDelta:
		for i := range 20 {
			b, err = r.ReadByte()
			if err != nil {
				return nil, fmt.Errorf("reading ref delta: %w", err)
			}
			if _, err := teeWriter.Write([]byte{b}); err != nil {
				return nil, fmt.Errorf("writing object type: %w", err)
			}
			res.RefDelta[i] = b
		}
	}
	return &res, nil
}

func readDeltaBaseOffset(r io.ByteReader, teeWriter io.Writer) (int64, int, error) {
	b, err := r.ReadByte()
	if err != nil {
		return 0, 0, fmt.Errorf("reading first byte: %w", err)
	}
	if teeWriter != nil {
		_, err = teeWriter.Write([]byte{b})
		if err != nil {
			return 0, 0, fmt.Errorf("teeing first byte: %w", err)
		}
	}
	value := int64(b & 0x7f)
	bytesSize := 1
	for i := 1; (b & 0x80) != 0; i++ {
		b, err = r.ReadByte()
		if err != nil {
			return 0, 0, fmt.Errorf("reading byte: %w", err)
		}
		if teeWriter != nil {
			_, err = teeWriter.Write([]byte{b})
			if err != nil {
				return 0, 0, fmt.Errorf("teeing byte: %w", err)
			}
		}
		value = int64(b&0x7f) + ((value + 1) << 7)
		bytesSize++
	}
	return value, bytesSize, nil
}
