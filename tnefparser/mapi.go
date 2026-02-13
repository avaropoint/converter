package tnefparser

import "encoding/binary"

func decodeMAPI(data []byte) []MAPIAttr {
	if len(data) < 4 {
		return nil
	}
	count := int(binary.LittleEndian.Uint32(data[0:4]))
	off := 4

	// Cap pre-allocation to prevent OOM from crafted files.
	// Each MAPI attr needs at least 8 bytes, so limit capacity accordingly.
	maxAttrs := len(data) / 8
	if count > maxAttrs {
		count = maxAttrs
	}
	attrs := make([]MAPIAttr, 0, count)

	for i := 0; i < count && off+4 <= len(data); i++ {
		pt := int(binary.LittleEndian.Uint16(data[off : off+2]))
		pid := int(binary.LittleEndian.Uint16(data[off+2 : off+4]))
		off += 4

		mv := (pt & 0x1000) != 0
		bt := pt & 0xEFFF
		fs := fixedPropSize(bt)
		if fs < 0 {
			mv = true
		}

		// Named properties carry extra GUID + kind header.
		if pid >= 0x8000 && pid <= 0xFFFE {
			if off+16 > len(data) {
				break
			}
			off += 16
			if off+4 > len(data) {
				break
			}
			kind := int(binary.LittleEndian.Uint32(data[off : off+4]))
			off += 4
			if kind == 0 {
				if off+4 > len(data) {
					break
				}
				off += 4
			} else {
				if off+4 > len(data) {
					break
				}
				nl := int(binary.LittleEndian.Uint32(data[off : off+4]))
				off += 4 + nl + padTo4(nl)
			}
		}

		vc := 1
		if mv {
			if off+4 > len(data) {
				break
			}
			vc = int(binary.LittleEndian.Uint32(data[off : off+4]))
			off += 4
		}
		if vc < 0 || vc > 4096 {
			break
		}

		var ad []byte
		ok := true
		for v := 0; v < vc; v++ {
			l := fs
			if fs < 0 {
				if off+4 > len(data) {
					ok = false
					break
				}
				l = int(binary.LittleEndian.Uint32(data[off : off+4]))
				off += 4
			}
			if l < 0 || off+l > len(data) {
				ok = false
				break
			}
			ad = append(ad, data[off:off+l]...)
			off += l + padTo4(l)
		}
		if !ok {
			break
		}
		attrs = append(attrs, MAPIAttr{Type: bt, Name: pid, Data: ad})
	}
	return attrs
}

func fixedPropSize(pt int) int {
	switch pt {
	case 0x0002, 0x000B:
		return 4
	case 0x0003, 0x0004, 0x000A:
		return 4
	case 0x0005, 0x0006, 0x0007, 0x0014, 0x0040:
		return 8
	case 0x0048:
		return 16
	case 0x001E, 0x001F, 0x000D, 0x0102:
		return -1
	default:
		return 4
	}
}

func padTo4(n int) int {
	return (4 - n%4) % 4
}
