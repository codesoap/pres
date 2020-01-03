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
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Provide one input file as the second argument")
		os.Exit(1)
	}
	inputFilename := os.Args[2]
	switch command {
	case create:
		createPresFile(inputFilename)
	case verify:
		verifyPresFile(inputFilename)
	case restore:
		restoreData(inputFilename)
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
	case "v":
		fallthrough
	case "verify":
		return verify, nil
	case "r":
		fallthrough
	case "restore":
		return restore, nil
	default:
		return -1, errors.New(fmt.Sprint("unknown command ", os.Args[1]))
	}
}
