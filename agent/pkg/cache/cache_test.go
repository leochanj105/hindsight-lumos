package cache

import (
	"testing"
)

func TestLRU(t *testing.T) {
	CacheInit(2)
	CacheSet(1, 2, 3)
	CacheSet(4, 5, 6)
	CacheSet(7, 8, 9)
	cache := CachePrint()
	t.Log(cache)

	CacheGet(4)

	cache = CachePrint()
	t.Log(cache)

	actual := CacheGet(1)
	expected := 0
	if len(actual) != expected {
		t.Errorf("CacheGet = %d; expected %d", actual, expected)
	}

	actual = CacheGet(4)
	expected = 1
	if len(actual) != expected {
		t.Errorf("CacheGet = %d; expected %d", actual, expected)
	} else {
		t.Log(actual)
	}
}
