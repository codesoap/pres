package main

import (
	"fmt"
	"github.com/klauspost/reedsolomon"
	"hash"
	"hash/crc32"
	"io"
	"io/ioutil"
	"os"
)

func createPresFile(inFilename string) {
	var conf conf
	conf.dataShardCnt = 100
	conf.parityShardCnt = 3

	var err error
	conf.dataLen, err = getFilesize(inFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error checking input filesize:", err.Error())
		os.Exit(1)
	} else if conf.dataLen < int64(conf.dataShardCnt)*int64(conf.dataShardCnt) {
		fmt.Fprintln(os.Stderr, "The input file must contain at least",
			int(conf.dataShardCnt)*int(conf.dataShardCnt), "bytes.")
		os.Exit(1)
	}

	hashers := getShardsHashers(conf)
	fmt.Fprintln(os.Stderr, "Calculating parity information and checksums.")
	parityFilenames, err := makeParityFilesAndCalculateHashes(inFilename, conf, hashers)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error creating parity files:", err.Error())
		os.Exit(2)
	}
	fmt.Fprintf(os.Stderr, "Appending output to '%s'.\n", inFilename)
	if err = copyOverData(inFilename, parityFilenames...); err != nil {
		fmt.Fprintln(os.Stderr, "Error writing parity to output:", err.Error())
		os.Exit(3)
	}
	err = writeMetadata(inFilename, conf, hashers)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error writing metadata:", err.Error())
		os.Exit(3)
	}
	presFilename := fmt.Sprint(inFilename, ".pres")
	fmt.Fprintf(os.Stderr, "Renaming '%s' to '%s'.\n", inFilename, presFilename)
	if err = os.Rename(inFilename, presFilename); err != nil {
		fmt.Fprintln(os.Stderr, "Error renaming to *.pres:", err.Error())
		os.Exit(4)
	}
	if err = removeFiles(parityFilenames); err != nil {
		fmt.Fprintln(os.Stderr, "Error removing temporary files:", err.Error())
		os.Exit(5)
	}
}

func getShardsHashers(conf conf) []hash.Hash32 {
	hashers := make([]hash.Hash32, conf.dataShardCnt+conf.parityShardCnt)
	for i := 0; i < int(conf.dataShardCnt+conf.parityShardCnt); i += 1 {
		hashers[i] = crc32.New(crc32.MakeTable(crc32.Castagnoli))
	}
	return hashers
}

func makeParityFilesAndCalculateHashes(inFilename string, conf conf, hashers []hash.Hash32) ([]string, error) {
	dataInputs, err := getDataInputs(inFilename, conf.dataShardCnt)
	if err != nil {
		return nil, err
	}
	dataInputReaders, err := toDataInputReaders(dataInputs, conf.dataShardCnt, hashers)
	if err != nil {
		return nil, err
	}
	parityOutputs, err := getParityOutputs(conf.parityShardCnt)
	if err != nil {
		return nil, err
	}
	parityOutputWriters := getParityOutputWriters(parityOutputs, conf, hashers)
	err = writeParityFiles(dataInputReaders, conf, parityOutputWriters)
	if err != nil {
		return nil, err
	}
	for i := range dataInputs {
		err = dataInputs[i].Close()
		if err != nil {
			return nil, err
		}
	}
	parityFilenames := make([]string, conf.parityShardCnt)
	for i, parityOutput := range parityOutputs {
		parityFilenames[i] = parityOutput.Name()
		err = parityOutput.Close()
		if err != nil {
			return nil, err
		}
	}
	return parityFilenames, nil
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

func writeMetadata(inFilename string, conf conf, hashers []hash.Hash32) error {
	destFile, err := os.OpenFile(inFilename, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer destFile.Close()
	shardsHashes := make([]string, conf.dataShardCnt+conf.parityShardCnt)
	for i := range hashers {
		shardsHashes[i] = fmt.Sprint(hashers[i].Sum32())
	}
	conf.version = "1"
	conf.shardCRC32Cs = shardsHashes
	if _, err := fmt.Fprintln(destFile, "\n\n[conf]"); err != nil {
		return err
	}
	if err = writeConf(destFile, conf); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(destFile, "\n[conf_copy_1]"); err != nil {
		return err
	}
	if err = writeConf(destFile, conf); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(destFile, "\n[conf_copy_2]"); err != nil {
		return err
	}
	return writeConf(destFile, conf)
}

func getDataInputs(inFilename string, dataShardCnt uint8) ([]*os.File, error) {
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
			shardSize, err = getShardSize(inputs[0], dataShardCnt)
			if err != nil {
				return nil, err
			}
		}
	}
	return inputs, nil
}

func toDataInputReaders(dataInputs []*os.File, dataShardCnt uint8, shardHashers []hash.Hash32) ([]io.Reader, error) {
	shardSize, err := getShardSize(dataInputs[0], dataShardCnt)
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

func getParityOutputs(parityShardCnt uint8) ([]*os.File, error) {
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

func getParityOutputWriters(outputs []*os.File, conf conf, shardHashers []hash.Hash32) []io.Writer {
	writers := make([]io.Writer, conf.parityShardCnt)
	for i := range outputs {
		writers[i] = outputs[i]
		j := i + int(conf.dataShardCnt)
		writers[i] = io.MultiWriter(writers[i], shardHashers[j])
	}
	return writers
}

func writeParityFiles(dataInputReaders []io.Reader, conf conf, parityOutputWriters []io.Writer) error {
	enc, err := reedsolomon.NewStream(int(conf.dataShardCnt), int(conf.parityShardCnt))
	if err != nil {
		return err
	}
	err = enc.Encode(dataInputReaders, parityOutputWriters)
	return err
}

func getShardSize(input *os.File, dataShardCnt uint8) (int64, error) {
	fileSize, err := getDataLen(input)
	if err != nil {
		return -1, err
	}
	return calculateShardSize(fileSize, dataShardCnt), nil
}

func getFilesize(filename string) (int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return -1, err
	}
	defer file.Close()
	return getDataLen(file)
}
