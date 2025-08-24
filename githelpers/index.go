package githelpers

import (
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"iter"
	"os"
	"strings"
)

type IdxObject struct {
	Pack   string
	Offset int
}

func Index(out *error) iter.Seq2[string, IdxObject] {
	return func(yield func(string, IdxObject) bool) {
		entries, err := os.ReadDir(".git/objects/pack")
		if err != nil {
			*out = fmt.Errorf("reading pack dir: %w", err)
			return
		}
		for _, e := range entries {
			if !strings.HasSuffix(e.Name(), ".idx") {
				continue
			}
			// For each *.idx file, read all objects from it.
			data, err := os.ReadFile(".git/objects/pack/" + e.Name())
			if err != nil {
				*out = errors.Join(*out, fmt.Errorf("reading idx file: %w", err))
				continue
			}
			hashOffsets := 256*4 + 8
			objectCount := int(binary.BigEndian.Uint32(data[hashOffsets-4 : hashOffsets]))
			packOffsets := hashOffsets + objectCount*20 + objectCount*4
			for i := range objectCount {
				// Object hash.
				hash := hex.EncodeToString(data[hashOffsets+i*20 : hashOffsets+(i+1)*20])
				if !yield(hash, IdxObject{
					// Idx file name.
					Pack: ".git/objects/pack/" + strings.TrimSuffix(e.Name(), ".idx") + ".pack",
					// Location of this object in *.pack file.
					Offset: int(binary.BigEndian.Uint32(data[packOffsets+i*4 : packOffsets+(i+1)*4])),
				}) {
					return
				}
			}
		}
	}
}
