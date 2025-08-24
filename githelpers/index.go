package githelpers

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"iter"
	"os"
	"slices"
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
		fileIndex := slices.IndexFunc(entries, func(e os.DirEntry) bool { return strings.HasSuffix(e.Name(), ".idx") })
		if fileIndex < 0 {
			return
		}
		data, err := os.ReadFile(".git/objects/pack/" + entries[fileIndex].Name())
		if err != nil {
			*out = fmt.Errorf("reading idx file: %w", err)
			return
		}
		hashOffsets := 256*4 + 8
		objectCount := int(binary.BigEndian.Uint32(data[hashOffsets-4 : hashOffsets]))
		packOffsets := hashOffsets + objectCount*20 + objectCount*4
		for i := range objectCount {
			// Object hash.
			hash := hex.EncodeToString(data[hashOffsets+i*20 : hashOffsets+(i+1)*20])
			if !yield(hash, IdxObject{
				// Idx file name.
				Pack: ".git/objects/pack/" + strings.TrimSuffix(entries[fileIndex].Name(), ".idx") + ".pack",
				// Location of this object in *.pack file.
				Offset: int(binary.BigEndian.Uint32(data[packOffsets+i*4 : packOffsets+(i+1)*4])),
			}) {
				return
			}
		}
	}
}
