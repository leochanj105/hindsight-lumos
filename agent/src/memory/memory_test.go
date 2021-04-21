package memory

import (
	"testing"

	. "util"
)

func TestDataReading(t *testing.T) {
	pool := MemInit("/dev/shm/pool_data", 100*50*4)
	dict := MemInit("/dev/shm/dict_data", 3200)

	actual := len(pool)
	expected := 100 * 50 * 4
	if actual != expected {
		t.Errorf("Pool size = %d; expected %d", actual, expected)
	}

	actual = len(dict)
	expected = 3200
	if actual != expected {
		t.Errorf("Dict size = %d; expected %d", actual, expected)
	}

	actual_char := string(dict[0:15])
	expected_char := "breadcrumb_test"
	if actual_char != expected_char {
		t.Errorf("Breadcrumb data = %s; expected %s", actual_char, expected_char)
		t.Errorf("%d %d", len(actual_char), len(expected_char))
	}

	var payload int = 0

	for j := 0; j < 4; j++ {
		actual = int(BytesToInt64(pool[4*j*50 : 4*(j*50+2)]))
		expected = 1000
		if actual != expected {
			t.Errorf("Request ID = %d; expected %d", actual, expected)
		}

		if j == 0 {
			actual = int(BytesToInt64(pool[4*(j*50+4) : 4*(j*50+6)]))
			expected = 1001
			if actual != expected {
				t.Errorf("Span ID = %d; expected %d", actual, expected)
			}

			actual = int(BytesToInt64(pool[4*(j*50+6) : 4*(j*50+8)]))
			expected = 1002
			if actual != expected {
				t.Errorf("Parent Span ID = %d; expected %d", actual, expected)
			}
		}

		if j == 3 {
			actual = int(BytesToInt32(pool[4*(j*50+8) : 4*(j*50+9)]))
			expected = 1
			if actual != expected {
				t.Errorf("Breadcrumb count = %d; expected %d", actual, expected)
			}

			actual = int(BytesToInt32(pool[4*(j*50+9) : 4*(j*50+10)]))
			expected = 0
			if actual != expected {
				t.Errorf("Breadcrumb first index = %d; expected %d", actual, expected)
			}
		}

		for i := 18; i < 50; i++ {
			actual = int(BytesToInt32(pool[4*(j*50+i) : 4*(j*50+i+1)]))
			expected = payload
			if j*50+i == 18 {
				expected = 10000
			}
			if actual != expected {
				t.Errorf("Payload = %d; expected %d at offset %d", actual, expected, j*50+i)
			}

			payload += 1
			if payload == 100 {
				break
			}
		}

	}

}
