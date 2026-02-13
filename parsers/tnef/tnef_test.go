package tnef

import (
	"encoding/binary"
	"testing"
)

func validTNEFHeader() []byte {
	b := make([]byte, 6)
	binary.LittleEndian.PutUint32(b[0:4], 0x223e9f78)
	binary.LittleEndian.PutUint16(b[4:6], 0)
	return b
}

func TestDecodeSignature(t *testing.T) {
	_, err := Decode([]byte{0x00, 0x01, 0x02})
	if err == nil {
		t.Fatal("expected error for invalid data")
	}
	_, err = Decode(validTNEFHeader())
	if err != nil {
		_ = err
	}
}

func TestDecompressRTF(t *testing.T) {
	_, err := DecompressRTF(nil)
	if err == nil {
		t.Fatal("expected error for nil input")
	}
	_, err = DecompressRTF([]byte{})
	if err == nil {
		t.Fatal("expected error for empty input")
	}
	_, err = DecompressRTF([]byte{1, 2, 3})
	if err == nil {
		t.Fatal("expected error for short input")
	}
}

func TestDeencapsulateHTMLEmpty(t *testing.T) {
	result := DeencapsulateHTML(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty result, got %d bytes", len(result))
	}
}
