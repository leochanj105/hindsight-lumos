package cache

import (
    "container/list"
    "fmt"
    "sync"

    "github.com/geraldleizhang/hindsight/agent/pkg/memory"
)

type TraceData struct {
    Request_id  uint64
    Buffers     []int
    Breadcrumbs []string
}

type TraceCache struct {
    lru         *list.List
    data        map[uint64]*list.Element // e.Value.(*TraceData)
    capacity    int // Above this threshold, we should evict
    eviction_batch_size int // Each eviction should aim for this many buffers
    size        int
    mutex       sync.RWMutex
}

// var cache LRU

func CacheInit(capacity int) *TraceCache {
    var cache TraceCache
    cache.lru = list.New()
    cache.data = make(map[uint64]*list.Element)
    cache.capacity = capacity
    cache.eviction_batch_size = 20 // Hard code to some value for now
    cache.size = 0
    return &cache
}

// Get the specified trace ID, but leave it in the cache
func (cache *TraceCache) Get(request_id uint64) *TraceData {
    var res []int
    
    cache.mutex.Lock()
    defer cache.mutex.Unlock()

    if entry, ok := cache.data[request_id]; ok {
        return entry.Value.(*TraceData)
    }
    return nil
}

func (cache *TraceCache) Put(buffers []memory.CompleteBuffer) {
    // First, organize by trace_id
    data := make(map[uint64][]int, len(buffers))
    for _, buf := range buffers {
        v := data[buf.Request_id]
        v = append(v, buf.Buffer_id)
        data[buf.Request_id] = v
    }

    // Acquire lock and update
    cache.mutex.Lock()
    defer cache.mutex.Unlock()

    for request_id, buffer_ids := range data {
        if entry, ok := cache.data[request_id]; ok {
            // This request already exists in the cache, update existing
            trace := entry.Value.(*TraceData)
            trace.Buffers = append(trace.Buffers, buffer_ids...)
        } else {
            // Request doesn't exist yet in the cache, create and insert
            var trace TraceData
            trace.Request_id = request_id
            trace.Buffers = buffer_ids

            entry := cache.lru.PushFront(&trace)
            cache.data[request_id] = entry
        }
        cache.size += len(buffer_ids)
    }
}

// Removes and returns the TraceData that should be evicted next
func (cache *TraceCache) evictNext() *TraceData {
    // Get next trace to be evicted
    entry := cache.lru.Back()
    trace := entry.Value.(*TraceData)

    // Remove from LRU and lookup
    delete(cache.data, trace.Request_id)
    cache.lru.Remove(entry)
    cache.size -= len(trace.Buffers)
    
    return trace
}


func CacheManager() {
    // fmt.Println("[cache mngr stat] size=", cache.size)
    for true {
        // TODO: cache manager strategy here
        if cache.cap-cache.size >= 5 {
            break
        }
        // fmt.Println("[cache mngr] cap", cache.cap, "- size", cache.size, "< 5")
        // request_id, evicted := Evict()
        // fmt.Println("[cache mngr] evict", request_id, len(evicted))
        CacheLock()
        _, evicted := Evict()
        for buffer_id, _ := range evicted {
            // fmt.Println("[cache mngr] put buffer", buffer_id, "to available")
            QueuePut(Available, buffer_id)
        }
        CacheUnlock()

    }
}

func CachePrint() []int64 {
    var res []int64
    for e := cache.lru.Front(); e != cache.lru.Back().Next(); e = e.Next() {
        res = append(res, e.Value.(Node).request_id)
    }
    return res
}

func PrintCacheStat() {
    fmt.Println("[cache_stat]", cache.cap, cache.size)
}
