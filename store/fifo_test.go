package store

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// nolint:goconst
func TestFIFO(t *testing.T) {
	table := []struct {
		name        string
		key         string
		value       interface{}
		op          string
		expectedErr bool
		found       bool
	}{
		{
			name:        "it return err when key expired",
			op:          "load",
			key:         "expired",
			expectedErr: true,
			found:       true,
		},
		{
			name:  "it return false when key does not exist",
			op:    "load",
			key:   "key",
			found: false,
		},
		{
			name:  "it return true and value when exist",
			op:    "load",
			key:   "test",
			value: "test",
			found: true,
		},
		{
			name:  "it overwrite exist key and value when store",
			op:    "store",
			key:   "test",
			value: "test2",
			found: true,
		},
		{
			name:  "it create new record when store",
			op:    "store",
			key:   "key",
			value: "value",
			found: true,
		},
		{
			name:  "it's not crash when trying to delete a non exist record",
			key:   "key",
			found: false,
		},
		{
			name:  "it delete a exist record",
			op:    "delete",
			key:   "test",
			found: false,
		},
	}

	for _, tt := range table {
		t.Run(tt.name, func(t *testing.T) {

			queue := &queue{
				notify: make(chan struct{}, 10),
				mu:     &sync.Mutex{},
			}

			cache := &FIFO{
				queue: queue,
				TTL:   time.Second,
				records: map[string]*record{
					"test": {
						Value: "test",
						Exp:   time.Now().Add(time.Hour),
					},
					"expired": {
						Value: "expired",
						Exp:   time.Now().Add(-time.Hour),
					},
				},
				MU: &sync.Mutex{},
			}

			r, _ := http.NewRequest("GET", "/", nil)
			var err error

			switch tt.op {
			case "load":
				v, ok, err := cache.Load(tt.key, r)
				assert.Equal(t, tt.value, v)
				assert.Equal(t, tt.found, ok)
				assert.Equal(t, tt.expectedErr, err != nil)
				return
			case "store":
				err = cache.Store(tt.key, tt.value, r)
				assert.Equal(t, tt.key, queue.next().Key)
			case "delete":
				err = cache.Delete(tt.key, r)
			}

			assert.Equal(t, tt.expectedErr, err != nil)
			v, ok := cache.records[tt.key]
			assert.Equal(t, tt.found, ok)

			if tt.value != nil {
				assert.Equal(t, tt.value, v.Value)
			}
		})
	}
}

func TestFIFOEvict(t *testing.T) {
	evictedKeys := make([]string, 0)
	onEvictedFun := func(key string, value interface{}) {
		evictedKeys = append(evictedKeys, key)
	}

	fifo := NewFIFO(context.Background(), 1)
	fifo.OnEvicted = onEvictedFun

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("myKey%d", i)
		fifo.Store(key, 1234, nil)
	}

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("myKey%d", i)
		fifo.Delete(key, nil)
	}

	assert.Equal(t, 10, len(evictedKeys))

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("myKey%d", i)
		assert.Equal(t, key, evictedKeys[i])
	}
}

func TestQueue(t *testing.T) {
	queue := &queue{
		notify: make(chan struct{}, 10),
		mu:     &sync.Mutex{},
	}

	for i := 0; i < 5; i++ {
		queue.push(
			&record{
				Key:   "any",
				Value: i,
			})
	}

	for i := 0; i < 5; i++ {
		r := queue.next()
		assert.Equal(t, i, r.Value)
	}
}

func TestFifoKeys(t *testing.T) {
	ctx, cacnel := context.WithCancel(context.Background())
	defer cacnel()

	f := NewFIFO(ctx, time.Minute)

	f.Store("1", "", nil)
	f.Store("2", "", nil)
	f.Store("3", "", nil)

	assert.ElementsMatch(t, []string{"1", "2", "3"}, f.Keys())
}

func TestGC(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	cache := NewFIFO(ctx, time.Nanosecond*50)

	cache.Store("1", 1, nil)
	cache.Store("2", 2, nil)

	time.Sleep(time.Millisecond)
	_, ok, _ := cache.Load("1", nil)
	assert.False(t, ok)

	_, ok, _ = cache.Load("2", nil)
	assert.False(t, ok)
}

func BenchmarkFIFIO(b *testing.B) {
	cache := NewFIFO(context.Background(), time.Minute)
	benchmarkCache(b, cache)
}

func benchmarkCache(b *testing.B, cache Cache) {
	keys := []string{}

	for i := 0; i < 100; i++ {
		key := strconv.Itoa(i)
		keys = append(keys, key)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			key := keys[rand.Intn(100)]
			_, ok, _ := cache.Load(key, nil)
			if ok {
				cache.Delete(key, nil)
			} else {
				cache.Store(key, struct{}{}, nil)
			}
		}
	})
}
