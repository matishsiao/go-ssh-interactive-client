package main

import (
	"time"
	//"syscall"
	"fmt"
	"log"
	"os"

	//"code.google.com/p/go.crypto/ssh"

	//"code.google.com/p/go.crypto/ssh/terminal"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	//"github.com/fatih/color"
)

type password string

var sessionFlag int = 0

func main() {
	manClient()
}

//Start the ssh client
func manClient() {
	server := "127.0.0.1"
	port := "22"
	server = server + ":" + port
	user := "user"
	password = "iamPassword"

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	fmt.Println("Connect to server:", server)
	conn, err := ssh.Dial("tcp", server, config)
	if err != nil {
		panic("Failed to dial: " + err.Error())
	}
	defer conn.Close()

	// Each ClientConn can support multiple interactive sessions,
	// represented by a Session.
	session, err := conn.NewSession()
	if err != nil {
		panic("Failed to create session: " + err.Error())
	}
	fmt.Printf("%s Connected.", server)
	sessionFlag = 1
	defer session.Close()
	//Use local Stdin FD to interactive mode.
	fd := int(os.Stdin.Fd())
	oldState, err := terminal.MakeRaw(fd)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(fd, oldState)

	termWidth, termHeight, err := terminal.GetSize(fd)
	if err != nil {
		panic(err)
	}
	// Set IO
	out, _ := session.StdoutPipe()
	errout, _ := session.StderrPipe()
	session.Stdin = os.Stdin
	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", termHeight, termWidth, modes); err != nil {
		log.Fatal(err)
	}

	// Start remote shell
	if err := session.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}
	//Read the Stderr data
	go func() {
		var (
			buf [65 * 1024]byte
			t   int
		)
		for {
			n, err := errout.Read(buf[t:])
			if err != nil {
				if err.Error() != "EOF" {
					fmt.Println("STDERR:", err.Error())
				}
				sessionFlag = 2
				return
			}
			t += n
			result := string(buf[:t])
			fmt.Print(result)
			t = 0
		}
	}()

	//Read the Stdout data
	go func() {
		var (
			buf [65 * 1024]byte
			t   int
		)
		for {
			os.Stdout.Sync()
			n, err := out.Read(buf[t:])
			if err != nil {
				if err.Error() != "EOF" {
					fmt.Println("STDOUT:", err.Error())
				}
				sessionFlag = 3
				return
			}
			t += n
			result := string(buf[:t])
			fmt.Print(result)
			t = 0
		}
	}()
	//Keep the process.
	for {
		select {
		case <-time.After(100 * time.Millisecond):
			if sessionFlag > 1 {
				session.Close()
				return
			}
		}
	}
}
