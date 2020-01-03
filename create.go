package main

import (
	"fmt"
	"github.com/klauspost/reedsolomon"
	"github.com/mattn/go-isatty"
	"hash"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
)

func createPresFile(inFilename string) {
	outputFile, err := getPresOutput(inFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error opening output:", err.Error())
		os.Exit(1)
	}
	defer outputFile.Close()

	hashers := getShardsHashers()
	fmt.Fprintln(os.Stderr, "Calculating parity information and checksums.")
	parityFilenames, err := makeParityFilesAndCalculateHashes(inFilename, hashers)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating parity files:", err.Error())
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "Writing output.")
	if err := writeHeader(inFilename, outputFile, hashers); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing Header:", err.Error())
		os.Exit(3)
	}
	if err = copyOverData(outputFile, inFilename); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing data to output:", err.Error())
		os.Exit(4)
	}
	if err = copyOverData(outputFile, parityFilenames...); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing parity to output:", err.Error())
		os.Exit(4)
	}
	if err = removeFiles(parityFilenames); err != nil {
		fmt.Fprintln(os.Stderr, "Error removing temporary files:", err.Error())
		os.Exit(5)
	}
}

func getPresOutput(inFilename string) (*os.File, error) {
	if isatty.IsTerminal(os.Stdout.Fd()) {
		outputFilename := fmt.Sprint(inFilename, ".pres")
		return os.Create(outputFilename)
	}
	return os.Stdout, nil
}

func getShardsHashers() []hash.Hash32 {
	hashers := make([]hash.Hash32, dataShardCnt+parityShardCnt)
	for i := 0; i < dataShardCnt+parityShardCnt; i += 1 {
		hashers[i] = crc32.New(crc32.MakeTable(crc32.Castagnoli))
	}
	return hashers
}

func makeParityFilesAndCalculateHashes(inFilename string, shardHashers []hash.Hash32) ([]string, error) {
	dataInputs, err := getDataInputs(inFilename)
	if err != nil {
		return nil, err
	}
	dataInputReaders, err := toDataInputReaders(dataInputs, shardHashers)
	if err != nil {
		return nil, err
	}
	parityOutputs, err := getParityOutputs()
	if err != nil {
		return nil, err
	}
	parityOutputWriters := getParityOutputWriters(parityOutputs, shardHashers)
	if err = writeParityFiles(dataInputReaders, parityOutputWriters); err != nil {
		return nil, err
	}
	for i := range dataInputs {
		err = dataInputs[i].Close()
		if err != nil {
			return nil, err
		}
	}
	parityFilenames := make([]string, parityShardCnt)
	for i, parityOutput := range parityOutputs {
		parityFilenames[i] = parityOutput.Name()
		err = parityOutput.Close()
		if err != nil {
			return nil, err
		}
	}
	return parityFilenames, nil
}

func writeHeader(inFilename string, outputFile *os.File, hashers []hash.Hash32) error {
	// FIXME: Do I really have to open the file again?
	inputFile, err := os.Open(inFilename)
	if err != nil {
		return err
	}
	dataLen, err := getDataLen(inputFile)
	if err != nil {
		return err
	}
	if err = inputFile.Close(); err != nil {
		return err
	}
	shardsHashes := make([]string, dataShardCnt+parityShardCnt)
	for i := range hashers {
		shardsHashes[i] = fmt.Sprint(hashers[i].Sum32())
	}
	conf := newConf("1", dataLen, shardsHashes)
	if _, err := fmt.Fprintln(outputFile, "[conf]"); err != nil {
		return err
	}
	writeConf(outputFile, conf)
	if _, err := fmt.Fprintln(outputFile, "\n[conf_copy_1]"); err != nil {
		return err
	}
	writeConf(outputFile, conf)
	if _, err := fmt.Fprintln(outputFile, "\n[conf_copy_2]"); err != nil {
		return err
	}
	writeConf(outputFile, conf)
	_, err = fmt.Fprintln(outputFile, "\n[binary]")
	return err
}

func copyOverData(destFile *os.File, srcFilenames ...string) error {
	for _, srcFilename := range srcFilenames {
		srcFile, err := os.Open(srcFilename)
		if err != nil {
			return err
		}
		defer srcFile.Close()
		if _, err = io.CopyBuffer(destFile, srcFile, nil); err != nil {
			return err
		}
	}
	return nil
}

func getDataInputs(inFilename string) ([]*os.File, error) {
	var err error
	var shardSize int64
	inputs := make([]*os.File, dataShardCnt)
	for i := range inputs {
		if inputs[i], err = os.Open(inFilename); err != nil {
			return nil, err
		}
		if _, err = inputs[i].Seek(int64(i)*shardSize, 0); err != nil {
			return nil, err
		}
		if i == 0 {
			shardSize, err = getShardSize(inputs[0])
			if err != nil {
				return nil, err
			}
		}
	}
	return inputs, nil
}

func toDataInputReaders(dataInputs []*os.File, shardHashers []hash.Hash32) ([]io.Reader, error) {
	shardSize, err := getShardSize(dataInputs[0])
	if err != nil {
		return nil, err
	}
	dataLen, err := getDataLen(dataInputs[0])
	if err != nil {
		return nil, err
	}
	inputReaders := make([]io.Reader, dataShardCnt)
	for i := range dataInputs {
		inputReaders[i] = dataInputs[i]
		inputReaders[i] = io.LimitReader(inputReaders[i], shardSize)
		inputReaders[i] = io.TeeReader(inputReaders[i], shardHashers[i])
	}
	i := dataShardCnt - 1
	inputReaders[i] = fillLastDataReader(inputReaders[i], dataShardCnt, dataLen)
	return inputReaders, nil
}

func getParityOutputs() ([]*os.File, error) {
	var err error
	outputs := make([]*os.File, parityShardCnt)
	for i := range outputs {
		outputs[i], err = ioutil.TempFile("", "pres_parity_file_*")
		if err != nil {
			return nil, err
		}
	}
	return outputs, nil
}

func getParityOutputWriters(outputs []*os.File, shardHashers []hash.Hash32) []io.Writer {
	writers := make([]io.Writer, parityShardCnt)
	for i := range outputs {
		writers[i] = outputs[i]
		writers[i] = io.MultiWriter(writers[i], shardHashers[i+dataShardCnt])
	}
	return writers
}

func writeParityFiles(dataInputReaders []io.Reader, parityOutputWriters []io.Writer) error {
	enc, err := reedsolomon.NewStream(dataShardCnt, parityShardCnt)
	if err != nil {
		return err
	}
	err = enc.Encode(dataInputReaders, parityOutputWriters)
	return err
}

func getShardSize(input *os.File) (int64, error) {
	fileSize, err := getDataLen(input)
	if err != nil {
		return -1, err
	}
	return calculateShardSize(fileSize, dataShardCnt), nil
}

func getDataLen(input *os.File) (int64, error) {
	stat, err := input.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}
