// types.go defines the core data structures for decoded TNEF messages,
// attachments, and MAPI properties.

package tnef

import (
	"bytes"
	"strings"
)

// Message holds the decoded contents of a TNEF stream.
type Message struct {
	Body        []byte        // Plain text body (PR_BODY).
	BodyHTML    []byte        // HTML body (PR_BODY_HTML).
	BodyRTF     []byte        // Decompressed RTF (from PR_RTF_COMPRESSED).
	BodyRTFHTML []byte        // HTML extracted from fromhtml1 RTF, if applicable.
	Attachments []*Attachment // File and embedded message attachments.
	Attributes  []MAPIAttr    // All decoded MAPI properties.
}

// GetAttr returns the first MAPI attribute matching the given property ID,
// or nil if not found.
func (m *Message) GetAttr(propID int) *MAPIAttr {
	for i := range m.Attributes {
		if m.Attributes[i].Name == propID {
			return &m.Attributes[i]
		}
	}
	return nil
}

// GetAttrString returns the string value of the first MAPI attribute matching
// propID, with null bytes and surrounding whitespace removed.
func (m *Message) GetAttrString(propID int) string {
	if a := m.GetAttr(propID); a != nil {
		return strings.TrimSpace(strings.ReplaceAll(string(a.Data), "\x00", ""))
	}
	return ""
}

// Attachment holds a single attachment (file, embedded message, or OLE object).
type Attachment struct {
	Title       string   // Short filename (8.3 format).
	LongName    string   // Long filename.
	Data        []byte   // Raw attachment content.
	MimeType    string   // MIME type, if available.
	ContentID   string   // Content-ID for inline images (cid: references).
	Method      int      // AttachByValue, AttachEmbeddedMsg, or AttachOLE.
	EmbeddedMsg *Message // Decoded nested message, if Method is AttachEmbeddedMsg.
}

// Filename returns the best available display name for the attachment,
// preferring the long name over the short name.
func (a *Attachment) Filename() string {
	if a.LongName != "" {
		return a.LongName
	}
	if a.Title != "" {
		return a.Title
	}
	return "unnamed"
}

// MAPIAttr holds a single decoded MAPI property.
type MAPIAttr struct {
	Type int    // MAPI property type (e.g. PT_LONG, PT_STRING8, PT_BINARY).
	Name int    // MAPI property ID (e.g. 0x0037 for PR_SUBJECT).
	Data []byte // Raw property value bytes.
}

// ResolveContentIDs replaces cid: references in BodyHTML and BodyRTFHTML
// with the filenames returned by mapper for each attachment that has a
// Content-ID.
func (m *Message) ResolveContentIDs(mapper func(att *Attachment) string) {
	// Build CID → replacement filename map from attachments.
	cidMap := make(map[string]string)
	for _, att := range m.Attachments {
		if att.ContentID == "" {
			continue
		}
		name := mapper(att)
		if name == "" {
			continue
		}
		cidMap[att.ContentID] = name
	}
	if len(cidMap) == 0 {
		return
	}

	// Replace in both HTML bodies.
	m.BodyHTML = replaceCIDs(m.BodyHTML, cidMap)
	m.BodyRTFHTML = replaceCIDs(m.BodyRTFHTML, cidMap)
}

// replaceCIDs substitutes all cid: references in html with the mapped values.
func replaceCIDs(html []byte, cidMap map[string]string) []byte {
	if len(html) == 0 {
		return html
	}
	result := html
	for cid, filename := range cidMap {
		old := []byte("cid:" + cid)
		new := []byte(filename)
		result = bytes.ReplaceAll(result, old, new)
	}
	return result
}
