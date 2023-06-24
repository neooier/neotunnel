package util

import "sync"

var handleStatusMutex sync.Mutex // 同一时间只允许一个连接
