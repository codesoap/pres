package main

import (
	"bufio"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
)

func verifyPresFile(inFilename string) {
	confs, err := readConfs(inFilename)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading conf sections:", err.Error())
		os.Exit(2)
	}
	correctConfs := getCorrectConfs(confs)
	warned := false
	if len(correctConfs) == 0 {
		fmt.Println("Could not find unharmed conf block.")
		os.Exit(2)
	} else if len(correctConfs) < 3 {
		fmt.Fprintln(os.Stderr, "WARNING: One conf block is damaged!")
		warned = true
	} else {
		fmt.Fprintln(os.Stderr, "All conf blocks are intact.")
	}
	conf := correctConfs[0]
	generatedHashes, err := generateHashes(inFilename, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error calculating hashes:", err.Error())
		os.Exit(3)
	}
	matchingHashes := countMatchingHashes(generatedHashes, conf.shardCRC32Cs)
	var shardCnt uint8 = dataShardCnt + parityShardCnt
	fmt.Fprintln(os.Stderr, matchingHashes, "out of", shardCnt,
		"shards are intact.")
	if matchingHashes < dataShardCnt {
		fmt.Println("Restoration impossible: not enought shards are intact.")
		os.Exit(4)
	} else if matchingHashes < shardCnt {
		damagedShards := shardCnt - matchingHashes
		fmt.Fprintln(os.Stderr, "WARNING:", damagedShards,
			"shard(s) is/are damaged!")
		warned = true
	}
	if warned {
		fmt.Println("Restore data and newly create the *.pres file to remove",
			"warnings.")
	} else {
		fmt.Println("No problems found.")
	}
}

func readConfs(inFilename string) ([]conf, error) {
	confs := make([]conf, 3)
	inputFile, err := os.Open(inFilename)
	if err != nil {
		return nil, err
	}
	defer inputFile.Close()
	inputReader := bufio.NewReader(inputFile)
	var confIndex int = -1
	var line string
	reBinary := regexp.MustCompile(`ary\]|\[bin`)
	reShard := regexp.MustCompile(`shard_[0-9]*_crc32c=.*`)
	for err = nil; err == nil; line, err = inputReader.ReadString('\n') {
		line = strings.TrimSpace(line)
		switch line {
		case "[conf]":
			confIndex = 0
			continue
		case "[conf_copy_1]":
			confIndex = 1
			continue
		case "[conf_copy_2]":
			confIndex = 2
			continue
		}
		if confIndex < 0 {
			// There were probably corrupted lines at the beginning of the file.
			continue
		}
		switch {
		case strings.HasPrefix(line, "version="):
			confs[confIndex].version = strings.SplitAfterN(line, "=", 2)[1]
		case strings.HasPrefix(line, "data_len="):
			s := strings.SplitAfterN(line, "=", 2)[1]
			confs[confIndex].dataLen, _ = strconv.ParseInt(s, 10, 64)
		case strings.HasPrefix(line, "data_shard_cnt="):
			s := strings.SplitAfterN(line, "=", 2)[1]
			x, _ := strconv.ParseUint(s, 10, 8)
			confs[confIndex].dataShardCnt = uint8(x)
		case strings.HasPrefix(line, "parity_shard_cnt="):
			s := strings.SplitAfterN(line, "=", 2)[1]
			x, _ := strconv.ParseUint(s, 10, 8)
			confs[confIndex].parityShardCnt = uint8(x)
			shardCnt := confs[confIndex].dataShardCnt + uint8(x)
			confs[confIndex].shardCRC32Cs = make([]string, shardCnt)
		case reShard.Match([]byte(line)):
			s := strings.SplitAfterN(line, "_", 3)[1]
			s = strings.Trim(s, "_")
			x, _ := strconv.ParseUint(s, 10, 8)
			shardIndex := uint8(x - 1)
			if int(shardIndex) >= len(confs[confIndex].shardCRC32Cs) {
				// Something went wrong; just go on
				continue
			}
			s = strings.SplitAfterN(line, "=", 2)[1]
			confs[confIndex].shardCRC32Cs[shardIndex] = s
		case reBinary.Match([]byte(line)):
			return confs, nil
		}
	}
	if err != nil {
		return nil, err
	}
	return nil, errors.New("input file is incomplete")
}

func getCorrectConfs(confs []conf) []conf {
	correctConfs := make([]conf, 0, 3)
	if confs[0].seemsOK() &&
		(confs[0].equals(confs[1]) || confs[0].equals(confs[2])) {
		correctConfs = append(correctConfs, confs[0])
	}
	if confs[1].seemsOK() &&
		(confs[1].equals(confs[0]) || confs[1].equals(confs[2])) {
		correctConfs = append(correctConfs, confs[1])
	}
	if confs[2].seemsOK() &&
		(confs[2].equals(confs[1]) || confs[2].equals(confs[0])) {
		correctConfs = append(correctConfs, confs[2])
	}
	return correctConfs
}

func generateHashes(inFilename string, conf conf) ([]string, error) {
	readers, files, err := getShardReaders(inFilename, conf)
	if err != nil {
		return nil, err
	}
	defer func() {
		for _, file := range files {
			file.Close()
		}
	}()
	return generateHashesFromReaders(readers, conf)
}

func countMatchingHashes(generatedHashes, storedHashes []string) uint8 {
	var matchingHashes uint8
	if len(generatedHashes) != len(storedHashes) {
		return 0
	}
	for i := range generatedHashes {
		if generatedHashes[i] == storedHashes[i] {
			matchingHashes += 1
		}
	}
	return matchingHashes
}

func getShardReaders(inFilename string, conf conf) ([]io.Reader, []*os.File, error) {
	headerSize, err := getBinaryOffset(inFilename)
	if err != nil {
		return nil, nil, err
	}
	shardSize := calculateShardSize(conf.dataLen, conf.dataShardCnt)
	files := make([]*os.File, conf.dataShardCnt+conf.parityShardCnt)
	readers := make([]io.Reader, conf.dataShardCnt+conf.parityShardCnt)
	for i := 0; i < int(conf.dataShardCnt+conf.parityShardCnt); i += 1 {
		files[i], err = os.Open(inFilename)
		if err != nil {
			return nil, nil, err
		}
		var offset int64 = headerSize
		if i < int(conf.dataShardCnt) {
			offset += int64(i) * shardSize
		} else {
			offset += conf.dataLen + int64(i-int(conf.dataShardCnt))*shardSize
		}
		if _, err = files[i].Seek(offset, 0); err != nil {
			return nil, nil, err
		}
		readers[i] = files[i]
		if i != int(conf.dataShardCnt-1) {
			readers[i] = io.LimitReader(readers[i], shardSize)
		} else {
			size := conf.dataLen - (int64(i) * shardSize)
			readers[i] = io.LimitReader(readers[i], size)
		}
	}
	return readers, files, nil
}

// getBinaryOffset determines how many bytes are at the beginning of the
// input file, that are not part of the data and parity information.
func getBinaryOffset(inFilename string) (int64, error) {
	inputFile, err := os.Open(inFilename)
	if err != nil {
		return -1, err
	}
	defer inputFile.Close()
	reader := bufio.NewReader(inputFile)
	reBin := regexp.MustCompile(`\[bin`)
	reAry := regexp.MustCompile(`ary\]`)
	pos := reBin.FindReaderIndex(reader)
	if pos != nil {
		return int64(pos[1] + 5), nil
	}
	inputFile.Seek(0, 0)
	reader = bufio.NewReader(inputFile)
	pos = reAry.FindReaderIndex(reader)
	if pos == nil {
		return -1, errors.New("could not find [binary] token")
	}
	return int64(pos[1] + 1), nil
}

func generateHashesFromReaders(readers []io.Reader, conf conf) ([]string, error) {
	hashes := make([]string, len(readers))
	hasher := crc32.New(crc32.MakeTable(crc32.Castagnoli))
	for i := range readers {
		if _, err := bufio.NewReader(readers[i]).WriteTo(hasher); err != nil {
			return nil, err
		}
		hashes[i] = fmt.Sprint(hasher.Sum32())
		hasher.Reset()
	}
	return hashes, nil
}
