package cache

import (
	"container/list"
	"fmt"

	. "queue"
)

type Node struct {
	request_id int64
	buffers    map[int]int64
}

type LRU struct {
	lru        *list.List
	hash_table map[int64]*list.Element // e.Value.(Node)
	cap        int
	size       int
}

var cache LRU

func CacheInit(cap int) {
	cache.lru = list.New()
	cache.hash_table = make(map[int64]*list.Element)
	cache.cap = cap
	cache.size = 0

	return
}

func CacheGet(request_id int64) map[int]int64 {
	if e, ok := cache.hash_table[request_id]; ok {
		cache.lru.MoveToFront(e)
		return e.Value.(Node).buffers
	}

	return nil
}

func CacheSet(request_id int64, buffer_id int, timestamp int64) map[int]int64 {
	var res map[int]int64
	fmt.Println("[cache_set]", request_id, buffer_id)

	if e, ok := cache.hash_table[request_id]; ok {
		if cache.size == cache.cap {
			_, res = Evict()
		}
		e.Value.(Node).buffers[buffer_id] = timestamp
		cache.lru.MoveToFront(e)
		cache.size += 1
	} else {
		if cache.size == cache.cap {
			_, res = Evict()
		}
		buffers := map[int]int64{buffer_id: timestamp}
		node := Node{request_id, buffers}
		e := cache.lru.PushFront(node)
		cache.hash_table[request_id] = e
		cache.size += 1
	}

	return res
}

func Evict() (int64, map[int]int64) {
	var res map[int]int64
	e := cache.lru.Back()
	res = e.Value.(Node).buffers
	request_id := e.Value.(Node).request_id
	delete(cache.hash_table, request_id)
	cache.lru.Remove(e)
	cache.size -= 1

	return request_id, res
}

func CacheManager() {
	fmt.Println("[cache mngr stat] size=", cache.size)
	for true {
		// TODO: cache manager strategy here
		if cache.cap-cache.size >= 5 {
			break
		}
		fmt.Println("[cache mngr] cap", cache.cap, "- size", cache.size, "< 5")
		request_id, evicted := Evict()
		fmt.Println("[cache mngr] evict", request_id, len(evicted))
		for buffer_id, _ := range evicted {
			fmt.Println("[cache mngr] put buffer", buffer_id, "to available")
			QueuePut(Available, buffer_id)
		}
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
