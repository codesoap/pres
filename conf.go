package main

import (
	"fmt"
	"os"
)

// Constants for tuning the behaviour of the program.
const (
	dataShardCnt   = 100
	parityShardCnt = 3
)

type Conf struct {
	version        string
	dataLen        int64
	dataShardCnt   uint8
	parityShardCnt uint8
	shardCRC32Cs   []string
}

func NewConf(version string, dataLen int64, shardsHashes []string) Conf {
	var conf Conf
	conf.version = version
	conf.dataLen = dataLen
	conf.dataShardCnt = dataShardCnt
	conf.parityShardCnt = parityShardCnt
	conf.shardCRC32Cs = shardsHashes
	return conf
}

func writeConf(outputFile *os.File, conf Conf) error {
	_, err := fmt.Fprintf(outputFile, "version=%s\n", conf.version)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(outputFile, "data_len=%d\n", conf.dataLen)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(outputFile, "data_shard_cnt=%d\n", conf.dataShardCnt)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(outputFile, "parity_shard_cnt=%d\n", conf.parityShardCnt)
	if err != nil {
		return err
	}
	for i, crc32c := range conf.shardCRC32Cs {
		_, err = fmt.Fprintf(outputFile, "shard_%d_crc32c=%s\n", i+1, crc32c)
		if err != nil {
			return err
		}
	}
	return nil
}
