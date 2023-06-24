package util

import "sync/atomic"

var uploadBytesCnt int64 = 0
var downloadBytesCnt int64 = 0

func GetUploadBytes() int64 {
	return atomic.LoadInt64(&uploadBytesCnt)
}

func GetDownloadBytes() int64 {
	return atomic.LoadInt64(&downloadBytesCnt)
}

func AddUploadBytes(delta int64) int64 {
	return atomic.AddInt64(&uploadBytesCnt, delta)
}

func AddDownloadBytes(delta int64) int64 {
	return atomic.AddInt64(&downloadBytesCnt, delta)
}
