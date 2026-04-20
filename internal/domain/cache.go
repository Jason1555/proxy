package domain

import "time"

type CacheEntry struct {
	Key        string
	Value      []byte
	StatusCode int
	Headers    map[string]string
	ExpiresAt  time.Time
	Tags 	 []string
	Size 	int64
}


