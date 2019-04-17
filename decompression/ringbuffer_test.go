package decompression

import (
	"bytes"
	"math/rand"
	"testing"
)

func TestRingbuffer(t *testing.T) {
	resultbuf := &bytes.Buffer{}

	rb := NewRingbuffer(10, resultbuf)
	err := rb.Push([]byte("Teststring"))
	if err != nil {
		t.Error(err.Error())
		return
	}

	if string(rb.data) != "Teststring" {
		t.Errorf("Wrong content: %s, should be: %s", string(rb.data), "Teststring")
	}
	result := resultbuf.String()
	if result != "" {
		t.Errorf("Pushed out but shouldnt at all: %s", result)
	}

	//#
	//#
	//#
	//#

	err = rb.Push([]byte("AABB"))
	if err != nil {
		t.Error(err.Error())
		return
	}
	if string(rb.data) != "AABBstring" {
		t.Errorf("Wrong content: %s, should be: %s", string(rb.data), "AABBstring")
	}
	result = resultbuf.String()
	if result != "Test" {
		t.Errorf("Pushed out: %s but should be: %s", result, "Test")
	}
	resultbuf.Reset()

	//#
	//#
	//#
	//#

	err = rb.Push([]byte("123456789012345678"))
	if err != nil {
		t.Error(err.Error())
		return
	}
	if string(rb.data) != "9012345678" {
		t.Errorf("Wrong content: %s, should be: %s", string(rb.data), "9012345678")
	}
	result = resultbuf.String()
	if result != "stringAABB12345678" {
		t.Errorf("Pushed out: %s but should be: %s", result, "stringAABB12345678")
	}
	resultbuf.Reset()

	//#
	//#
	//#
	//#

	err = rb.Push([]byte("ABCDEFGH"))
	if err != nil {
		t.Error(err.Error())
		return
	}
	if rb.String() != "78ABCDEFGH" {
		t.Errorf("Wrong content: %s, should be: %s", rb.String(), "78ABCDEFGH")
	}
	result = resultbuf.String()
	if result != "90123456" {
		t.Errorf("Pushed out: %s but should be: %s", result, "90123456")
	}
	resultbuf.Reset()
}

func TestRepeat(t *testing.T) {
	resultbuf := &bytes.Buffer{}

	rb := NewRingbuffer(10, resultbuf)
	err := rb.Push([]byte("Teststring"))
	if err != nil {
		t.Error(err.Error())
		return
	}
	if string(rb.data) != "Teststring" {
		t.Errorf("Wrong content: %s, should be: %s", string(rb.data), "Teststring")
	}
	result := resultbuf.String()
	if result != "" {
		t.Errorf("Pushed out but shouldnt at all: %s", result)
	}

	rb.RepeatLast(4)
	if string(rb.data) != "ringstring" {
		t.Errorf("Wrong content: %s, should be: %s", string(rb.data), "ringstring")
	}
	if rb.String() != "stringring" {
		t.Errorf("Wrong String output: %s, should be: %s", rb.String(), "stringring")
	}
	result = resultbuf.String()
	if result != "Test" {
		t.Errorf("Pushed out: %s but should be: %s", result, "Test")
	}
	resultbuf.Reset()

	rb.RepeatLast(8)
	if rb.String() != "ngringring" {
		t.Errorf("Wrong String output: %s, should be: %s", rb.String(), "ngringring")
	}
	result = resultbuf.String()
	if result != "stringri" {
		t.Errorf("Pushed out: %s but should be: %s", result, "stringri")
	}
	resultbuf.Reset()

	rb.Push([]byte("1234567890"))
	if rb.String() != "1234567890" {
		t.Errorf("Wrong String output: %s, should be: %s", rb.String(), "1234567890")
	}
	result = resultbuf.String()
	if result != "ngringring" {
		t.Errorf("Pushed out: %s but should be: %s", result, "ngringring")
	}
	resultbuf.Reset()

	rb.Repeat(5, 3)
	if rb.String() != "6789034567" {
		t.Errorf("Wrong String output: %s, should be: %s", rb.String(), "6789034567")
	}
	result = resultbuf.String()
	if result != "12345" {
		t.Errorf("Pushed out: %s but should be: %s", result, "12345")
	}
	resultbuf.Reset()

	rb.Repeat(3, 7)
	if rb.String() != "9034567678" {
		t.Errorf("Wrong String output: %s, should be: %s", rb.String(), "9034567678")
	}
	result = resultbuf.String()
	if result != "678" {
		t.Errorf("Pushed out: %s but should be: %s", result, "678")
	}
	resultbuf.Reset()
}

func TestRandomPushes(t *testing.T) {
	resultbuf := &bytes.Buffer{}
	rb := NewRingbuffer(100, resultbuf)
	pushed := 0
	for i := 0; i < 10000; i++ {

		toPush := make([]byte, rand.Intn(rb.Len))
		for idx := range toPush {
			//for convenience take only ascii characters, so error messages dont completly destroy your terminal
			toPush[idx] = 33 + byte(rand.Intn(127-33))
		}

		offsetbefore := rb.offset
		before := []byte(rb.String())
		resultbuf.Reset()

		err := rb.Push(toPush)
		if err != nil {
			t.Error(err.Error())
			return
		}
		out := resultbuf.Bytes()

		if rb.offset != (offsetbefore+len(toPush))%rb.Len {
			t.Errorf("Offset is: %d should: %d", rb.offset, (offsetbefore+len(toPush))%rb.Len)
		}

		if pushed > rb.Len {
			//length of pushed out must match length of pushed in
			should := len(toPush)
			if len(out) != should {
				t.Errorf("Not right amount pushed out: %d should be: %d", len(out), should)
				return
			}
			for j := 0; j < should; j++ {
				if out[j] != before[j] {
					t.Errorf("At Push/Index: %d/%d, Pushed out: %b when it should have pushed out: %b", i, j, out[j], before[j])
					return
				}
			}
		}
		if pushed <= rb.Len && pushed+len(toPush) > rb.Len {
			//length of pushed out must match difference
			should := (pushed + len(toPush)) - rb.Len
			if len(out) != should {
				t.Errorf("Not right amount pushed out: %d should be: %d", len(out), should)
				return
			}
			for j := 0; j < should; j++ {
				if out[j] != before[j] {
					t.Errorf("At Push/Index: %d/%d, Pushed out: %b when it should have pushed out: %b", i, j, out[j], before[j])
					return
				}
			}
		}
		if pushed <= rb.Len && pushed+len(toPush) <= rb.Len {
			//lenght of pushed out must be zero
			should := 0
			if len(out) != should {
				t.Errorf("Not right amount pushed out: %d should be: %d", len(out), should)
				return
			}
		}
		pushed += len(toPush)
	}
}

func TestRandomRepeates(t *testing.T) {
	resultbuf := &bytes.Buffer{}
	rb := NewRingbuffer(100, resultbuf)

	initial := make([]byte, 50)
	for idx := range initial {
		//for convenience take only ascii characters, so error messages dont completly destroy your terminal
		initial[idx] = 33 + byte(rand.Intn(127-33))
	}
	err := rb.Push(initial)
	if err != nil {
		t.Error(err.Error())
		return
	}
	pushed := len(initial)

	for i := 0; i < 10000; i++ {
		var toRepeat int
		toRepeat = rand.Intn(rb.Len)

		var oldest int
		if rb.allDirty {
			oldest = rand.Intn(rb.Len)
		} else {
			oldest = rand.Intn(rb.offset)
		}

		offsetbefore := rb.offset
		before := []byte(rb.String())
		resultbuf.Reset()

		rb.RepeatBeforeIndex(toRepeat, oldest)

		out := resultbuf.Bytes()
		after := []byte(rb.String())

		if rb.offset != (offsetbefore+toRepeat)%rb.Len {
			t.Errorf("Offset is: %d should: %d", rb.offset, (offsetbefore+toRepeat)%rb.Len)
		}

		startOfRepeated := len(after) - toRepeat
		startOfToBeRepated := len(after) - toRepeat - oldest

		startOfCompare := 0
		if startOfToBeRepated < 0 {
			startOfCompare = -startOfToBeRepated
		}

		for j := startOfCompare; j < toRepeat; j++ {
			x := after[startOfRepeated+j]
			y := after[startOfToBeRepated+j]
			if x != y {
				t.Errorf("Wrong byte repeated: %d: %d, %d", j, x, y)
			}
		}

		if pushed > rb.Len {
			//length of pushed out must match length of pushed in
			should := toRepeat
			if len(out) != should {
				t.Errorf("Not right amount pushed out: %d should be: %d", len(out), should)
				return
			}
			for j := 0; j < should; j++ {
				if out[j] != before[j] {
					t.Errorf("At Push/Index: %d/%d, Pushed out: %b when it should have pushed out: %b", i, j, out[j], before[j])
					return
				}
			}
		}
		if pushed <= rb.Len && pushed+toRepeat > rb.Len {
			//length of pushed out must match difference
			should := (pushed + toRepeat) - rb.Len
			if len(out) != should {
				t.Errorf("Not right amount pushed out: %d should be: %d", len(out), should)
				return
			}
			for j := 0; j < should; j++ {
				if out[j] != before[j] {
					t.Errorf("At Push/Index: %d/%d, Pushed out: %b when it should have pushed out: %b", i, j, out[j], before[j])
					return
				}
			}
		}
		if pushed <= rb.Len && pushed+toRepeat <= rb.Len {
			//lenght of pushed out must be zero
			should := 0
			if len(out) != should {
				t.Errorf("Not right amount pushed out: %d should be: %d", len(out), should)
				return
			}
		}
		pushed += toRepeat
	}
}
