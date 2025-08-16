package main

import (
	"fmt"
	"log"

	"github.com/gliderlabs/ssh"
)

func main() {
	ssh.Handle(func(s ssh.Session) {
		fmt.Printf("Running: %s\n", s.RawCommand())
		s.Write([]byte("ok"))
	})

	log.Fatal(ssh.ListenAndServe(":2222", nil,
		ssh.HostKeyFile("gitssh_host_key"), ssh.NoPty()))
}
