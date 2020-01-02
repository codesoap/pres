package main

import (
	"errors"
	"fmt"
	"os"
)

// Possible commands
const (
	create = iota
	verify
	restore
)

func main() {
	command, err := getCommand()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error when parsing command:", err.Error())
		os.Exit(1)
	}
	switch command {
	case create:
		createPresFile()
	}
}

func getCommand() (int, error) {
	if len(os.Args) < 2 {
		return -1, errors.New("no command given")
	}
	switch os.Args[1] {
	case "c":
		fallthrough
	case "create":
		return create, nil
	default:
		return -1, errors.New(fmt.Sprint("unknown command ", os.Args[1]))
	}
}
