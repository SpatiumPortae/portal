package cache

import "time"

type Storage interface {
	Get(key string) []byte
	Set(key string, content []byte, duration time.Duration)
}
