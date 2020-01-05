package main

import (
	"fmt"
	"os"
)

type conf struct {
	version        string
	dataLen        int64
	dataShardCnt   uint8
	parityShardCnt uint8
	shardCRC32Cs   []string
}

func writeConf(outputFile *os.File, conf conf) error {
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

func (c1 conf) seemsOK() bool {
	if c1.version == "" ||
		c1.dataLen <= 0 ||
		c1.dataShardCnt <= 0 ||
		c1.parityShardCnt <= 0 ||
		len(c1.shardCRC32Cs) < 2 {
		return false
	}
	return true
}

func (c1 conf) equals(c2 conf) bool {
	if c1.version != c2.version ||
		c1.dataLen != c2.dataLen ||
		c1.dataShardCnt != c2.dataShardCnt ||
		c1.parityShardCnt != c2.parityShardCnt ||
		len(c1.shardCRC32Cs) != len(c2.shardCRC32Cs) {
		return false
	}
	for i := range c1.shardCRC32Cs {
		if c1.shardCRC32Cs[i] != c2.shardCRC32Cs[i] {
			return false
		}
	}
	return true
}
