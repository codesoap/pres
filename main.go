package main

import (
	"errors"
	"fmt"
	"os"
)

// Possible commands
const (
	createCommand = iota
	verifyCommand
	restoreCommand
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
	inFilename := os.Args[2]
	switch command {
	case createCommand:
		createPresFile(inFilename)
	case verifyCommand:
		verifyPresFile(inFilename)
	case restoreCommand:
		restoreData(inFilename)
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
		return createCommand, nil
	case "v":
		fallthrough
	case "verify":
		return verifyCommand, nil
	case "r":
		fallthrough
	case "restore":
		return restoreCommand, nil
	default:
		return -1, errors.New(fmt.Sprint("unknown command ", os.Args[1]))
	}
}
