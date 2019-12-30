package main

import (
	"bufio"
	"fmt"
	"github.com/klauspost/reedsolomon"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

func createConsFile() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Provide one input file as the second argument")
		os.Exit(1)
	}
	inputFilename := os.Args[2]
	outputFilename := getOutputFilename(inputFilename)
	if _, err := os.Stat(outputFilename); !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "The file", outputFilename, "already exists")
		os.Exit(1)
	}

	fmt.Fprintln(os.Stderr, "Calculating parity information")
	parityFilenames, err := makeParityFiles(inputFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating parity files:", err.Error())
		os.Exit(2)
	}
	fmt.Fprintln(os.Stderr, "Writing headers to", outputFilename)
	if err := writeHeader(inputFilename, parityFilenames); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing Header:", err.Error())
		os.Exit(3)
	}
	fmt.Fprintln(os.Stderr, "Writing binary to", outputFilename)
	if err = copyOverData(outputFilename, inputFilename); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing data to outfile:", err.Error())
		os.Exit(4)
	}
	if err = copyOverData(outputFilename, parityFilenames...); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing parity to outfile:", err.Error())
		os.Exit(4)
	}
	fmt.Fprintln(os.Stderr, "Removing temporary files")
	if err = removeFiles(parityFilenames); err != nil {
		fmt.Fprintln(os.Stderr, "Error removing temporary files:", err.Error())
		os.Exit(5)
	}
}

func getOutputFilename(inputFilename string) string {
	return fmt.Sprint(inputFilename, ".cons")
}

func makeParityFiles(inputFilename string) ([]string, error) {
	dataInputs, err := getDataInputs(inputFilename)
	if err != nil {
		return nil, err
	}
	parityOutputs, err := getParityOutputs()
	if err != nil {
		return nil, err
	}
	dataInputReaders, err := toDataInputReaders(dataInputs)
	if err != nil {
		return nil, err
	}
	if err = writeParityFiles(dataInputReaders, parityOutputs); err != nil {
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

func writeHeader(inputFilename string, parityFilenames []string) error {
	outputFilename := getOutputFilename(inputFilename)
	outputFile, err := os.Create(outputFilename)
	if err != nil {
		return err
	}
	defer outputFile.Close()
	// FIXME: Do I really have to open the file again?
	inputFile, err := os.Open(inputFilename)
	dataLen, err := getDataLen(inputFile)
	if err != nil {
		return err
	}
	if err = inputFile.Close(); err != nil {
		return err
	}
	shardsHashes, err := getShardsHashes(inputFilename, parityFilenames)
	if err != nil {
		return err
	}
	conf := NewConf("1", dataLen, shardsHashes)
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

func copyOverData(destFilename string, srcFilenames ...string) error {
	destFile, err := os.OpenFile(destFilename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer destFile.Close()
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

func removeFiles(filenames []string) error {
	for _, filename := range filenames {
		if err := os.Remove(filename); err != nil {
			return err
		}
	}
	return nil
}

func getDataInputs(inputFilename string) ([]*os.File, error) {
	var err error
	var shardSize int64 = 0
	inputs := make([]*os.File, dataShardCnt)
	for i := range inputs {
		if inputs[i], err = os.Open(inputFilename); err != nil {
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

func getParityOutputs() ([]*os.File, error) {
	var err error
	outputs := make([]*os.File, parityShardCnt)
	for i := range outputs {
		outputs[i], err = ioutil.TempFile("", "cons_parity_file_*")
		if err != nil {
			return nil, err
		}
	}
	return outputs, nil
}

func toDataInputReaders(dataInputs []*os.File) ([]io.Reader, error) {
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
	}
	for i := 0; i < int((shardSize*dataShardCnt)-dataLen); i += 1 {
		r := strings.NewReader("0")
		inputReaders[dataShardCnt-1] = io.MultiReader(inputReaders[dataShardCnt-1], r)
	}
	return inputReaders, nil
}

func writeParityFiles(dataInputReaders []io.Reader, parityOutputs []*os.File) error {
	parityOutputWriters := make([]io.Writer, parityShardCnt)
	for i := range parityOutputs {
		parityOutputWriters[i] = parityOutputs[i]
	}
	enc, err := reedsolomon.NewStream(dataShardCnt, parityShardCnt)
	if err != nil {
		return err
	}
	err = enc.Encode(dataInputReaders, parityOutputWriters)
	return err
}

func getShardsHashes(inputFilename string, parityFilenames []string) ([]string, error) {
	hashes := make([]string, dataShardCnt+parityShardCnt)
	if err := fillInDataShardsHashes(hashes, inputFilename); err != nil {
		return nil, err
	}
	if err := fillInParityShardsHashes(hashes, parityFilenames); err != nil {
		return nil, err
	}
	return hashes, nil
}

func fillInDataShardsHashes(hashes []string, inputFilename string) error {
	inputFile, err := os.Open(inputFilename)
	if err != nil {
		return err
	}
	shardSize, err := getShardSize(inputFile)
	if err != nil {
		return err
	}
	buffer := make([]byte, bufferSize)
	var currentShard int = 0
	var readBytes int64 = 0
	hash := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	for {
		bytesLeftForShard := int((int64(1+currentShard) * shardSize) - readBytes)
		toRead := min(bytesLeftForShard, bufferSize)
		n, err := inputFile.Read(buffer[:toRead])
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}
		readBytes += int64(n)
		hash.Write(buffer[:n])
		if readBytes == (int64(1+currentShard)*shardSize) || n < toRead {
			hashes[currentShard] = fmt.Sprint(hash.Sum32())
			currentShard += 1
			hash.Reset()
		}
	}
	return nil
}

func fillInParityShardsHashes(hashes, parityFilenames []string) error {
	hash := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	for i, parityFilename := range parityFilenames {
		parityFile, err := os.Open(parityFilename)
		if err != nil {
			return err
		}
		reader := bufio.NewReader(parityFile)
		if _, err = reader.WriteTo(hash); err != nil {
			return err
		}
		hashes[dataShardCnt+i] = fmt.Sprint(hash.Sum32())
		hash.Reset()
	}
	return nil
}

func getShardSize(input *os.File) (int64, error) {
	fileSize, err := getDataLen(input)
	if err != nil {
		return -1, err
	}
	shardSize := fileSize / dataShardCnt
	if fileSize%dataShardCnt > 0 {
		shardSize += 1
	}
	return shardSize, nil
}

func getDataLen(input *os.File) (int64, error) {
	stat, err := input.Stat()
	if err != nil {
		return -1, err
	}
	return stat.Size(), nil
}
