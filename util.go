package main

import (
	"io"
	"os"
	"strings"
)

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func calculateShardSize(dataLen int64, dataShardCnt uint8) int64 {
	shardSize := dataLen / int64(dataShardCnt)
	if dataLen%int64(dataShardCnt) > 0 {
		shardSize += 1
	}
	return shardSize
}

func removeFiles(filenames []string) error {
	for _, filename := range filenames {
		if filename == "" {
			continue
		}
		if err := os.Remove(filename); err != nil {
			return err
		}
	}
	return nil
}

func fillLastDataReader(reader io.Reader, dataShardCnt uint8, dataLen int64) io.Reader {
	shardSize := calculateShardSize(dataLen, dataShardCnt)
	missingByteCnt := int((shardSize * int64(dataShardCnt)) - dataLen)
	for i := 0; i < missingByteCnt; i += 1 {
		r := strings.NewReader("0")
		reader = io.MultiReader(reader, r)
	}
	return reader
}
