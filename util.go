package main

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
