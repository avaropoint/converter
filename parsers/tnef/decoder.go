package tnef

import (
	"encoding/binary"
	"strings"
)

// Decode parses a raw TNEF byte stream and returns the decoded Message.
func Decode(data []byte) (*Message, error) {
	if len(data) < 6 {
		return nil, ErrBadSignature
	}
	if binary.LittleEndian.Uint32(data[0:4]) != tnefSignature {
		return nil, ErrBadSignature
	}

	msg := &Message{}
	offset := 6
	var cur *Attachment

	for offset+9 <= len(data) {
		lv := int(data[offset])
		id := int(binary.LittleEndian.Uint16(data[offset+1 : offset+3]))
		ln := int(binary.LittleEndian.Uint32(data[offset+5 : offset+9]))

		end := offset + 9 + ln + 2
		if end > len(data) {
			break
		}
		d := data[offset+9 : offset+9+ln]
		offset = end

		if lv == lvlAttachment && id == attrAttachRendData {
			cur = &Attachment{}
			msg.Attachments = append(msg.Attachments, cur)
			continue
		}

		if lv == lvlAttachment && cur != nil {
			switch id {
			case attrAttachTitle:
				cur.Title = cleanStr(string(d))
			case attrAttachData:
				cur.Data = d
			case attrAttachment:
				parseAttachProps(cur, d)
			}
			continue
		}

		if id == attrMAPIProps {
			attrs := decodeMAPI(d)
			msg.Attributes = append(msg.Attributes, attrs...)
			for _, a := range attrs {
				switch a.Name {
				case MAPIBody:
					msg.Body = a.Data
				case MAPIBodyHTML:
					msg.BodyHTML = a.Data
				case MAPIRtfCompressed:
					if rtf, err := DecompressRTF(a.Data); err == nil {
						msg.BodyRTF = rtf
						if html := DeencapsulateHTML(rtf); html != nil {
							msg.BodyRTFHTML = html
						}
					}
				}
			}
		}
	}

	return msg, nil
}

func parseAttachProps(att *Attachment, data []byte) {
	attrs := decodeMAPI(data)
	var obj []byte

	for _, a := range attrs {
		switch a.Name {
		case MAPIAttachFilename:
			if att.Title == "" {
				att.Title = cleanStr(string(a.Data))
			}
		case MAPIAttachLongFname:
			att.LongName = cleanStr(string(a.Data))
		case MAPIAttachMimeTag:
			att.MimeType = cleanStr(string(a.Data))
		case MAPIAttachContentID:
			att.ContentID = cleanStr(string(a.Data))
		case MAPIAttachMethod:
			if len(a.Data) >= 4 {
				att.Method = int(binary.LittleEndian.Uint32(a.Data))
			}
		case MAPIAttachDataObj:
			obj = a.Data
		}
	}

	if len(obj) > 0 && len(att.Data) == 0 {
		resolveNested(att, obj)
	}
}

func resolveNested(att *Attachment, obj []byte) {
	// Try with 16-byte IID prefix first.
	if len(obj) > 20 {
		after := obj[16:]
		if binary.LittleEndian.Uint32(after[0:4]) == tnefSignature {
			if n, e := Decode(after); e == nil {
				att.EmbeddedMsg = n
				att.Data = after
				return
			}
		}
	}
	// Try without prefix.
	if len(obj) >= 4 && binary.LittleEndian.Uint32(obj[0:4]) == tnefSignature {
		if n, e := Decode(obj); e == nil {
			att.EmbeddedMsg = n
			att.Data = obj
			return
		}
	}
	att.Data = obj
}

func cleanStr(s string) string {
	return strings.TrimSpace(strings.ReplaceAll(s, "\x00", ""))
}
