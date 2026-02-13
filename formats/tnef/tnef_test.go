package tnef

import (
	"encoding/binary"
	"testing"

	"github.com/avaropoint/converter/formats"
)

func TestConverterName(t *testing.T) {
	c := &converter{}
	if c.Name() != "TNEF (winmail.dat)" {
		t.Fatalf("unexpected name: %s", c.Name())
	}
}

func TestConverterExtensions(t *testing.T) {
	c := &converter{}
	exts := c.Extensions()
	if len(exts) != 2 {
		t.Fatalf("expected 2 extensions, got %d", len(exts))
	}
}

func TestMatchValid(t *testing.T) {
	c := &converter{}
	data := make([]byte, 10)
	binary.LittleEndian.PutUint32(data[0:4], 0x223e9f78)
	if !c.Match(data) {
		t.Fatal("expected Match to return true for valid TNEF signature")
	}
	if c.Match([]byte{0, 0, 0, 0}) {
		t.Fatal("expected Match to return false for invalid data")
	}
	if c.Match([]byte{0x78}) {
		t.Fatal("expected Match to return false for short data")
	}
}

func TestDetectRegistered(t *testing.T) {
	all := formats.All()
	found := false
	for _, c := range all {
		if c.Name() == "TNEF (winmail.dat)" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("TNEF converter not found in registry")
	}
}

func TestConvertInvalidData(t *testing.T) {
	c := &converter{}
	_, err := c.Convert([]byte{0, 1, 2, 3})
	if err == nil {
		t.Fatal("expected error converting invalid data")
	}
}
