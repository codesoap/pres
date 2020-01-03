package main

import (
	"errors"
	"fmt"
	"github.com/klauspost/reedsolomon"
	"github.com/mattn/go-isatty"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

const (
	intact  = true
	damaged = false
)

func restoreData(inFilename string) {
	fmt.Fprintln(os.Stderr, "Checking shards for damage.")
	conf, err := getConf(inFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading *.pres file:", err.Error())
		os.Exit(2)
	}
	shardStates, err := getShardStates(inFilename, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading *.pres file:", err.Error())
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "Restoring damaged shards.")
	restoredShards, err := restore(inFilename, shardStates, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error restoring damaged shards:", err.Error())
		os.Exit(3)
	}
	fmt.Fprintln(os.Stderr, "Verifying restored data.")
	err = verify(inFilename, restoredShards, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error verifying restored shards:", err.Error())
		os.Exit(4)
	}
	fmt.Fprintln(os.Stderr, "Writing output.")
	err = writeOutput(inFilename, restoredShards, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing output:", err.Error())
		os.Exit(5)
	}
	if err = removeFiles(restoredShards); err != nil {
		fmt.Fprintln(os.Stderr, "Error removing temporary files:", err.Error())
		os.Exit(6)
	}
}

func getConf(inFilename string) (conf, error) {
	confs, err := readConfs(inFilename)
	if err != nil {
		var dummy conf
		return dummy, err
	}
	correctConfs := getCorrectConfs(confs)
	if len(correctConfs) == 0 {
		var dummy conf
		return dummy, errors.New("could not find unharmed conf block")
	}
	return correctConfs[0], nil
}

func getShardStates(inFilename string, conf conf) ([]bool, error) {
	generatedHashes, err := generateHashes(inFilename, conf)
	if err != nil {
		return nil, err
	}
	shardStates := make([]bool, conf.dataShardCnt+conf.parityShardCnt)
	for i, hash := range conf.shardCRC32Cs {
		if hash == generatedHashes[i] {
			shardStates[i] = intact
		}
	}
	return shardStates, nil
}

func restore(inFilename string, shardStates []bool, conf conf) ([]string, error) {
	readers, files, err := getShardReaders(inFilename, conf)
	if err != nil {
		return nil, err
	}
	defer func() {
		for _, file := range files {
			file.Close()
		}
	}()
	i := conf.dataShardCnt - 1
	readers[i] = fillLastDataReader(readers[i], conf.dataShardCnt, conf.dataLen)
	outFilenames := make([]string, len(readers))
	writers := make([]io.Writer, len(readers))
	for i, shardState := range shardStates {
		if shardState == damaged {
			readers[i] = nil
			outFile, err := ioutil.TempFile("", "pres_restored_shard_*")
			if err != nil {
				return nil, err
			}
			defer outFile.Close()
			outFilenames[i] = outFile.Name()
			writers[i] = outFile
		}
	}
	dataShardCnt := int(conf.dataShardCnt)
	parityShardCnt := int(conf.parityShardCnt)
	enc, err := reedsolomon.NewStream(dataShardCnt, parityShardCnt)
	if err != nil {
		return nil, err
	}
	err = enc.Reconstruct(readers, writers)
	return outFilenames, err
}

func verify(inFilename string, restoredShards []string, conf conf) error {
	readers, files, err := getRestoredReaders(inFilename, restoredShards, conf)
	if err != nil {
		return err
	}
	defer func() {
		for _, file := range files {
			file.Close()
		}
	}()
	dataShardCnt := int(conf.dataShardCnt)
	parityShardCnt := int(conf.parityShardCnt)
	enc, err := reedsolomon.NewStream(dataShardCnt, parityShardCnt)
	if err != nil {
		return err
	}
	isOK, err := enc.Verify(readers)
	if !isOK {
		return errors.New("parity shards contain wrong data")
	}
	return err
}

func writeOutput(inFilename string, restoredShards []string, conf conf) error {
	readers, files, err := getRestoredReaders(inFilename, restoredShards, conf)
	if err != nil {
		return err
	}
	defer func() {
		for _, file := range files {
			file.Close()
		}
	}()
	dataShardCnt := int(conf.dataShardCnt)
	parityShardCnt := int(conf.parityShardCnt)
	enc, err := reedsolomon.NewStream(dataShardCnt, parityShardCnt)
	if err != nil {
		return err
	}
	output, err := getDataOutput(inFilename)
	if err != nil {
		return err
	}
	defer output.Close()
	var writer io.Writer = output
	return enc.Join(writer, readers, conf.dataLen)
}

func getRestoredReaders(inFilename string, restoredShards []string, conf conf) ([]io.Reader, []*os.File, error) {
	readers, files, err := getShardReaders(inFilename, conf)
	if err != nil {
		return nil, nil, err
	}
	i := conf.dataShardCnt - 1
	readers[i] = fillLastDataReader(readers[i], conf.dataShardCnt, conf.dataLen)
	for i, restoredShard := range restoredShards {
		if restoredShard != "" {
			file, err := os.Open(restoredShard)
			if err != nil {
				return nil, nil, err
			}
			files = append(files, file)
			readers[i] = file
		}
	}
	return readers, files, nil
}

func getDataOutput(inFilename string) (*os.File, error) {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		if !strings.HasSuffix(inFilename, ".pres") {
			return nil, errors.New("input file does not have .pres suffix")
		}
		outputFilename := strings.TrimSuffix(inFilename, ".pres")
		return os.Create(outputFilename)
	}
	return os.Stdout, nil
}
