package http

import (
	"time"
)

// TestHTTPGet 简化版的HTTP GET测试实现
func TestHTTPGet(url string, timeout time.Duration) *HTTPReturn {
	// 这里先返回一个示例结果，后续可以实现真正的HTTP请求逻辑
	result := &HTTPReturn{
		Success:          true,
		DestAddr:         url,
		Status:           200,
		ContentLength:    1024,
		DNSLookup:        time.Millisecond * 10,
		TCPConnection:    time.Millisecond * 20,
		TLSHandshake:     time.Millisecond * 30,
		ServerProcessing: time.Millisecond * 40,
		ContentTransfer:  time.Millisecond * 50,
		Total:            time.Millisecond * 150,
	}
	
	return result
} 