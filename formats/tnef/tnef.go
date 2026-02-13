// Package tnef implements the TNEF (winmail.dat) format converter.
// It is automatically registered with the formats registry on import.
package tnef

import (
	"encoding/base64"
	"encoding/binary"
	"strings"

	"github.com/avaropoint/converter/formats"
	parser "github.com/avaropoint/converter/parsers/tnef"
)

const tnefSignature = 0x223e9f78

func init() {
	formats.Register(&converter{})
}

type converter struct{}

func (c *converter) Name() string {
	return "TNEF (winmail.dat)"
}

func (c *converter) Extensions() []string {
	return []string{".dat", ".tnef"}
}

func (c *converter) Match(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	return binary.LittleEndian.Uint32(data[0:4]) == tnefSignature
}

func (c *converter) Convert(data []byte) ([]formats.ConvertedFile, error) {
	msg, err := parser.Decode(data)
	if err != nil {
		return nil, err
	}
	return collectAll(msg, ""), nil
}

func collectAll(msg *parser.Message, prefix string) []formats.ConvertedFile {
	var files []formats.ConvertedFile

	if len(msg.BodyHTML) > 0 || len(msg.BodyRTFHTML) > 0 {
		msg.ResolveContentIDs(func(att *parser.Attachment) string {
			if len(att.Data) == 0 {
				return ""
			}
			mime := mimeFromName(att.Filename())
			b64 := base64.StdEncoding.EncodeToString(att.Data)
			return "data:" + mime + ";base64," + b64
		})
	}

	// Fetch and embed any remaining external images so the HTML is
	// fully self-contained and viewable offline. Share the cache so
	// duplicate URLs across bodies are only fetched once.
	imgCache := make(map[string]string)
	msg.BodyHTML = formats.InlineExternalImages(msg.BodyHTML, imgCache)
	msg.BodyRTFHTML = formats.InlineExternalImages(msg.BodyRTFHTML, imgCache)

	if len(msg.Body) > 0 {
		files = append(files, formats.ConvertedFile{
			Name:     prefixed(prefix, "body.txt"),
			Data:     msg.Body,
			Category: "body",
		})
	}
	if len(msg.BodyHTML) > 0 {
		files = append(files, formats.ConvertedFile{
			Name:     prefixed(prefix, "body.html"),
			Data:     msg.BodyHTML,
			Category: "body",
		})
	}
	if len(msg.BodyRTF) > 0 {
		files = append(files, formats.ConvertedFile{
			Name:     prefixed(prefix, "body.rtf"),
			Data:     msg.BodyRTF,
			Category: "body",
		})
	}
	if len(msg.BodyRTFHTML) > 0 {
		files = append(files, formats.ConvertedFile{
			Name:     prefixed(prefix, "body_from_rtf.html"),
			Data:     msg.BodyRTFHTML,
			Category: "body",
		})
	}

	for _, att := range msg.Attachments {
		if att.EmbeddedMsg != nil {
			sub := formats.SanitizeFilename(att.Filename())
			if prefix != "" {
				sub = prefix + "_" + sub
			}
			files = append(files, collectAll(att.EmbeddedMsg, sub)...)
		} else if len(att.Data) > 0 {
			name := formats.SanitizeFilename(att.Filename())
			if prefix != "" {
				name = prefix + "_" + name
			}
			files = append(files, formats.ConvertedFile{
				Name:     name,
				Data:     att.Data,
				Category: "attachment",
			})
		}
	}

	return files
}

func prefixed(prefix, name string) string {
	if prefix != "" {
		return prefix + "_" + name
	}
	return name
}

// mimeFromName returns a MIME type based on file extension.
func mimeFromName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.HasSuffix(lower, ".png"):
		return "image/png"
	case strings.HasSuffix(lower, ".jpg"), strings.HasSuffix(lower, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lower, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lower, ".bmp"):
		return "image/bmp"
	case strings.HasSuffix(lower, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lower, ".webp"):
		return "image/webp"
	default:
		return "application/octet-stream"
	}
}
