package main

import (
	"bufio"
	"bytes"
	"compress/flate"
	"compress/zlib"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math"
	"os"
	"slices"

	"github.com/gliderlabs/ssh"
	"gitpatch.com/iefserge/gitssh-demo/githelpers"
)

func pkt(s string) string {
	return fmt.Sprintf("%04x%s\n", len(s)+5, s)
}

func main() {
	ssh.Handle(func(s ssh.Session) {
		ref, err := os.ReadFile(".git/refs/heads/main")
		if err != nil {
			s.Exit(1)
			return
		}
		ref = bytes.TrimRight(ref, "\n")

		io.WriteString(s, pkt(fmt.Sprintf("%s HEAD\x00object-format=sha1 symref=HEAD:refs/heads/main", ref)))
		io.WriteString(s, pkt(fmt.Sprintf("%s refs/heads/main", ref)))
		io.WriteString(s, "0000")

		scanner := bufio.NewScanner(s)
		for scanner.Scan() {
			if scanner.Text() == "00000009done" {
				break
			}
		}
		if err := scanner.Err(); err != nil {
			s.Exit(1)
			return
		}

		io.WriteString(s, pkt("NAK"))

		count := 0
		packObject := map[string][]githelpers.IdxObject{}
		for _, object := range githelpers.Index(&err) {
			packObject[object.Pack] = append(packObject[object.Pack], object)
			count++
		}
		if err != nil {
			s.Exit(1)
			return
		}

		h := sha1.New()
		sh := io.MultiWriter(s, h)
		sh.Write([]byte("PACK"))
		binary.Write(sh, binary.BigEndian, uint32(2))
		binary.Write(sh, binary.BigEndian, uint32(count))
		for pack, objects := range packObject {
			if err := writeObjects(sh, pack, objects); err != nil {
				slog.Error("writing objects", "pack", pack, "error", err)
				s.Exit(1)
			}
		}

		s.Write(h.Sum(nil))
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil,
		ssh.HostKeyFile("gitssh_host_key"), ssh.NoPty()))
}

// io.TeeReader does not implement ByteReader interface, which forces
// zlib to allocate an internal bufio.Reader. This custom tee reader
// implements both Reader() and ByteReader().
type teeByteReader struct {
	// Use flate.Reader here because this is the interface zlib
	// decompressor requires to avoid creating additional bufio.Reader.
	zlibInputReader flate.Reader
	zlibSize        int64
	teeWriter       io.Writer
}

func (zc *teeByteReader) Read(p []byte) (int, error) {
	n, err := zc.zlibInputReader.Read(p)
	zc.zlibSize += int64(n)
	if zc.teeWriter != nil && n > 0 {
		_, err = zc.teeWriter.Write(p[:n])
		if err != nil {
			return n, fmt.Errorf("zlib read tee error: %w", err)
		}
	}
	return n, err
}

func (zc *teeByteReader) ReadByte() (byte, error) {
	b, err := zc.zlibInputReader.ReadByte()
	if err == nil {
		zc.zlibSize++
		if zc.teeWriter != nil {
			_, err = zc.teeWriter.Write([]byte{b})
			if err != nil {
				return 0, fmt.Errorf("zlib readbyte tee error: %w", err)
			}
		}
	}
	return b, err
}

func writeObjects(w io.Writer, pack string, objects []githelpers.IdxObject) error {
	f, err := os.Open(pack)
	if err != nil {
		return fmt.Errorf("opening packfile: %w", err)
	}
	defer f.Close()

	slices.SortFunc(objects, func(a, b githelpers.IdxObject) int { return a.Offset - b.Offset })

	for _, object := range objects {
		sectionReader := io.NewSectionReader(f, int64(object.Offset), math.MaxInt64)
		reader := bufio.NewReader(sectionReader)

		// Read and copy object header.
		_, err := githelpers.CopyObjectHeader(reader, w)
		if err != nil {
			return fmt.Errorf("reading header: %w", err)
		}

		// Read and copy object body.
		zlibReader, err := zlib.NewReader(&teeByteReader{zlibInputReader: reader, teeWriter: w})
		if err != nil {
			return fmt.Errorf("creating zlib reader: %w", err)
		}

		// Read object and also write it into writer.
		if _, err := io.Copy(io.Discard, zlibReader); err != nil {
			return fmt.Errorf("copying object: %w", err)
		}
		zlibReader.Close()
	}

	return nil
}
