package memory

import (
	"encoding/binary"
	"testing"
)

func TestMetadataReading(t *testing.T) {
	pool := MemInit("/dev/shm/pool_test", 1008)
	dict := MemInit("/dev/shm/dict", 3200)

	actual := len(pool)
	expected := 1008
	if actual != expected {
		t.Errorf("Pool size = %d; expected %d", actual, expected)
	}

	actual = len(dict)
	expected = 3200
	if actual != expected {
		t.Errorf("Dict size = %d; expected %d", actual, expected)
	}

	t.Log(pool[1000:1008])

	t.Log(int64(binary.LittleEndian.Uint64(pool[8:16])))

	for i := 0; i < 250; i++ {
		t.Log(i, int32(binary.LittleEndian.Uint32(pool[(i)*4:(i+1)*4])))
	}

	t.Log(dict[0:32])

	t.Log(string(dict[0:32]))
}
