package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

const tnefSig = 0x223e9f78

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "usage: inspect <file>")
		os.Exit(1)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if len(data) < 6 || binary.LittleEndian.Uint32(data[0:4]) != tnefSig {
		fmt.Fprintln(os.Stderr, "not a TNEF file")
		os.Exit(1)
	}
	fmt.Printf("TNEF file: %s (%d bytes)\n\n", os.Args[1], len(data))
	offset := 6
	attNum := 0
	for offset+9 <= len(data) {
		lv := data[offset]
		id := binary.LittleEndian.Uint16(data[offset+1 : offset+3])
		tp := binary.LittleEndian.Uint16(data[offset+3 : offset+5])
		ln := int(binary.LittleEndian.Uint32(data[offset+5 : offset+9]))
		end := offset + 9 + ln + 2
		if end > len(data) {
			fmt.Printf("  [TRUNCATED at offset %d]\n", offset)
			break
		}
		payload := data[offset+9 : offset+9+ln]
		offset = end
		lvStr := "MSG"
		if lv == 0x02 {
			lvStr = "ATT"
		}
		fmt.Printf("[%s] attr=0x%04X %-22s type=0x%04X  size=%d\n", lvStr, id, aName(id), tp, ln)
		if id == 0x9002 {
			attNum++
			fmt.Printf("       >>> Attachment #%d\n", attNum)
		}
		if id == 0x8010 {
			fmt.Printf("       Title: %q\n", noNull(string(payload)))
		}
		if id == 0x800F {
			fmt.Printf("       AttachData: %d bytes\n", ln)
		}
		if id == 0x9003 || id == 0x9005 {
			dumpMAPI(payload, "       ")
		}
	}
}

func dumpMAPI(data []byte, ind string) {
	if len(data) < 4 {
		return
	}
	count := int(binary.LittleEndian.Uint32(data[0:4]))
	fmt.Printf("%sMAPI props: %d\n", ind, count)
	off := 4
	for i := 0; i < count && off+4 <= len(data); i++ {
		pt := int(binary.LittleEndian.Uint16(data[off : off+2]))
		pid := int(binary.LittleEndian.Uint16(data[off+2 : off+4]))
		off += 4
		mv := (pt & 0x1000) != 0
		bt := pt & 0xEFFF
		fs := fSize(bt)
		if fs < 0 {
			mv = true
		}
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
				off += 4 + nl + p4(nl)
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
		total := 0
		var samp []byte
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
			total += l
			if v == 0 && l <= 200 {
				samp = data[off : off+l]
			}
			off += l + p4(l)
		}
		if !ok {
			fmt.Printf("%s  [PARSE ERROR at prop %d]\n", ind, i)
			break
		}
		fmt.Printf("%s  0x%04X %-30s %-10s %d", ind, pid, pName(pid), tName(bt), total)
		if bt == 0x001E || bt == 0x001F {
			fmt.Printf("  %q", noNull(string(samp)))
		} else if bt == 0x0003 && len(samp) >= 4 {
			fmt.Printf("  val=%d", binary.LittleEndian.Uint32(samp))
		} else if bt == 0x000B && len(samp) >= 4 {
			v := binary.LittleEndian.Uint32(samp)
			fmt.Printf("  val=%v", v != 0)
		}
		fmt.Println()
	}
}

func fSize(t int) int {
	switch t {
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

func p4(n int) int { return (4 - n%4) % 4 }

func noNull(s string) string {
	o := make([]byte, 0, len(s))
	for i := range s {
		if s[i] != 0 {
			o = append(o, s[i])
		}
	}
	return string(o)
}

func aName(id uint16) string {
	m := map[uint16]string{
		0x8005: "attDateSent", 0x8006: "attDateRecd", 0x8008: "attMessageClass",
		0x8009: "attMessageID", 0x800C: "attBody", 0x800D: "attPriority",
		0x800F: "attAttachData", 0x8010: "attAttachTitle", 0x8011: "attAttachMetaFile",
		0x8012: "attAttachCreateDate", 0x8013: "attAttachModifyDate",
		0x8020: "attDateModified", 0x9001: "attFrom", 0x9002: "attAttachRendData",
		0x9003: "attMAPIProps", 0x9004: "attRecipTable", 0x9005: "attAttachment",
		0x9006: "attTnefVersion", 0x9007: "attOemCodepage",
	}
	if n, ok := m[id]; ok {
		return n
	}
	return ""
}

func pName(pid int) string {
	m := map[int]string{
		0x001A: "PR_MESSAGE_CLASS", 0x0037: "PR_SUBJECT", 0x003D: "PR_SUBJECT_PREFIX",
		0x0042: "PR_SENT_REPRESENTING_NAME", 0x0065: "PR_SENT_REPRESENTING_EMAIL",
		0x0070: "PR_CONVERSATION_TOPIC", 0x0071: "PR_CONVERSATION_INDEX",
		0x0C1A: "PR_SENDER_NAME", 0x0C1E: "PR_SENDER_ADDRTYPE",
		0x0C1F: "PR_SENDER_EMAIL_ADDRESS", 0x0E03: "PR_DISPLAY_CC",
		0x0E04: "PR_DISPLAY_TO", 0x0E06: "PR_MESSAGE_DELIVERY_TIME",
		0x0E07: "PR_MESSAGE_FLAGS", 0x0E08: "PR_MESSAGE_SIZE",
		0x0E1D: "PR_SUBJECT_NORMALIZED", 0x0FF9: "PR_RECORD_KEY",
		0x1000: "PR_BODY", 0x1009: "PR_RTF_COMPRESSED", 0x1013: "PR_BODY_HTML",
		0x1035: "PR_INTERNET_MESSAGE_ID", 0x1039: "PR_INTERNET_CPID",
		0x3001: "PR_DISPLAY_NAME", 0x3007: "PR_CREATION_TIME",
		0x3008: "PR_LAST_MODIFICATION_TIME", 0x300B: "PR_SEARCH_KEY",
		0x3701: "PR_ATTACH_DATA_OBJ", 0x3702: "PR_ATTACH_ENCODING",
		0x3703: "PR_ATTACH_EXTENSION", 0x3704: "PR_ATTACH_FILENAME",
		0x3705: "PR_ATTACH_METHOD", 0x3707: "PR_ATTACH_LONG_FILENAME",
		0x3709: "PR_ATTACH_RENDERING", 0x370B: "PR_RENDERING_POSITION",
		0x370E: "PR_ATTACH_MIME_TAG", 0x3712: "PR_ATTACH_CONTENT_ID",
		0x3714: "PR_ATTACH_FLAGS", 0x0002: "PR_ALTERNATE_RECIPIENT",
	}
	if n, ok := m[pid]; ok {
		return n
	}
	return ""
}

func tName(t int) string {
	m := map[int]string{
		0x0002: "PT_SHORT", 0x0003: "PT_LONG", 0x000B: "PT_BOOLEAN",
		0x001E: "PT_STRING8", 0x001F: "PT_UNICODE", 0x0040: "PT_SYSTIME",
		0x0048: "PT_CLSID", 0x0102: "PT_BINARY", 0x000D: "PT_OBJECT",
	}
	if n, ok := m[t]; ok {
		return n
	}
	return fmt.Sprintf("0x%04X", t)
}
