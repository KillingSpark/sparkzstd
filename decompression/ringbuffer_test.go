package decompression

import (
	"bytes"
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
