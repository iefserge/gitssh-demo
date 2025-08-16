package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/gliderlabs/ssh"
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
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil,
		ssh.HostKeyFile("gitssh_host_key"), ssh.NoPty()))
}
