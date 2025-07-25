#  LFU Cache

A high-performance, thread-safe **LFU (Least Frequently Used) cache** written in Go.  
Supports **O(1)** time `Get` and `Set` operations, TTL-based expiration, automatic cleanup, and customizable eviction callbacks.


## 🚀 Features

-  **O(1) Get/Set** operations
-  **LFU eviction policy** (Least Frequently Used)
-  **Global TTL expiration** per entry
-  **Automatic cleanup loop** for stale keys
-  **Eviction callback hook** for logging or custom actions
-  **Thread-safe** 



## 📦 Installation

```bash
go get github.com/ozedd-ee/lfu
```


##  Usage:

```go
package main

import (
	"fmt"
	"time"
	"lfu"
)

func main() {
	cache := lfu.New[string, int](
		3,                    // capacity
		5*time.Minute,        // TTL
		1*time.Minute,        // cleanup interval
		func(k string, v int) { // customizable eviction callback
			fmt.Printf("Evicted: %s = %d\n", k, v)
		},
	)

	cache.Set("a", 100)
	cache.Set("b", 200)

	if val, ok := cache.Get("a"); ok {
		fmt.Println("a =", val)
	}
}
```



## 🧪 Running Tests

```bash
go test -v 
```
## Run benchmarks
```bash
go test -bench=. -benchmem
```

##  API

```go
func New[K comparable, V any](
	capacity int,
	ttl time.Duration,
	cleanupInterval time.Duration,
	onEvict EvictionCallback[K, V],
) *LFUCache[K, V]
```