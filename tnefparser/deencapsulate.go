package tnefparser

import (
	"bytes"
	"strings"
)

// DeencapsulateHTML extracts the original HTML content from an RTF stream that
// was created with Outlook's \fromhtml1 encapsulation (MS-OXRTFEX).
//
// The RTF contains {\*\htmltag} groups with the original HTML markup, and plain
// text between those groups that represents visible content.  Regions wrapped in
// \htmlrtf ... \htmlrtf0 are RTF-only formatting and must be skipped.
//
// If the RTF is not HTML-encapsulated, it returns nil.
func DeencapsulateHTML(rtf []byte) []byte {
	// Quick check: must contain \fromhtml to be encapsulated HTML.
	if !bytes.Contains(rtf, []byte(`\fromhtml`)) {
		return nil
	}

	var out bytes.Buffer
	out.Grow(len(rtf)) // Pre-allocate — output is typically smaller than input.
	n := len(rtf)
	i := 0
	inHtmlRtf := false // true inside \htmlrtf ... \htmlrtf0 regions
	seenTag := false   // true after first {\*\htmltag} — suppresses RTF preamble text

	for i < n {
		// {\*\htmltag<N> <content>} — original HTML markup fragments.
		if rtf[i] == '{' && i+11 < n && rtf[i+1] == '\\' && rtf[i+2] == '*' &&
			rtf[i+3] == '\\' && hasPrefix(rtf[i+4:], "htmltag") {
			j := i + 11 // past "{\*\htmltag"
			for j < n && rtf[j] >= '0' && rtf[j] <= '9' {
				j++
			}
			if j < n && rtf[j] == ' ' {
				j++
			}
			content := extractGroupContent(rtf, j, n)
			decoded := decodeRTFFragment(content)
			out.WriteString(decoded)
			i = skipGroup(rtf, i, n)
			seenTag = true
			continue
		}

		// \htmlrtf0 — end of RTF-only region.  Check BEFORE \htmlrtf.
		if rtf[i] == '\\' && i+9 <= n && hasPrefix(rtf[i:], `\htmlrtf0`) {
			inHtmlRtf = false
			i += 9
			if i < n && rtf[i] == ' ' {
				i++
			}
			continue
		}

		// \htmlrtf — start of RTF-only region.
		if rtf[i] == '\\' && i+8 <= n && hasPrefix(rtf[i:], `\htmlrtf`) {
			// Verify it's the complete control word (not e.g. \htmlrtfoo).
			j := i + 8
			if j >= n || !isAlpha(rtf[j]) {
				inHtmlRtf = true
				// Skip optional numeric parameter.
				for j < n && rtf[j] >= '0' && rtf[j] <= '9' {
					j++
				}
				if j < n && rtf[j] == ' ' {
					j++
				}
				i = j
				continue
			}
		}

		// Inside \htmlrtf region — skip everything except state transitions.
		if inHtmlRtf {
			if rtf[i] == '\\' {
				i = skipControlWord(rtf, i, n)
			} else {
				i++
			}
			continue
		}

		// --- Outside htmlrtf: everything below is real content ---

		// Braces: step into/out of groups without emitting.
		if rtf[i] == '{' || rtf[i] == '}' {
			i++
			continue
		}

		// CR/LF: RTF line breaks are semantically ignored.
		if rtf[i] == '\r' || rtf[i] == '\n' {
			i++
			continue
		}

		// RTF control words and symbols.
		if rtf[i] == '\\' {
			if i+1 >= n {
				i++
				continue
			}
			if !seenTag {
				// Before first htmltag — skip all control words (RTF preamble).
				i = skipControlWord(rtf, i, n)
				continue
			}
			switch rtf[i+1] {
			case '\\':
				out.WriteByte('\\')
				i += 2
			case '{':
				out.WriteByte('{')
				i += 2
			case '}':
				out.WriteByte('}')
				i += 2
			case '~':
				// Non-breaking space.
				out.WriteString("&nbsp;")
				i += 2
			case '_':
				// Non-breaking hyphen.
				out.WriteString("&#8209;")
				i += 2
			case '-':
				// Optional hyphen — omit.
				i += 2
			case '\'':
				// \'XX hex-encoded byte.
				if i+3 < n {
					hi := unhex(rtf[i+2])
					lo := unhex(rtf[i+3])
					if hi >= 0 && lo >= 0 {
						out.WriteByte(byte(hi<<4 | lo))
					}
					i += 4
				} else {
					i += 2
				}
			case '\r', '\n':
				// \<CR>/<LF> — ignore.
				i += 2
			default:
				// Other control words: skip entirely.
				i = skipControlWord(rtf, i, n)
			}
			continue
		}

		// Literal text — this is actual content, include it (after preamble).
		if seenTag {
			out.WriteByte(rtf[i])
		}
		i++
	}

	result := out.String()
	result = strings.TrimSpace(result)
	if len(result) == 0 {
		return nil
	}
	return []byte(result)
}

// isAlpha returns true if c is an ASCII letter.
func isAlpha(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

// hasPrefix checks if data starts with prefix.
func hasPrefix(data []byte, prefix string) bool {
	if len(data) < len(prefix) {
		return false
	}
	for i := 0; i < len(prefix); i++ {
		if data[i] != prefix[i] {
			return false
		}
	}
	return true
}

// extractGroupContent extracts raw bytes from position start up to the matching
// closing brace, handling nested brace groups.
func extractGroupContent(data []byte, start, end int) string {
	var buf bytes.Buffer
	depth := 1
	i := start
	for i < end && depth > 0 {
		ch := data[i]
		if ch == '{' {
			depth++
			buf.WriteByte(ch)
			i++
		} else if ch == '}' {
			depth--
			if depth > 0 {
				buf.WriteByte(ch)
			}
			i++
		} else {
			buf.WriteByte(ch)
			i++
		}
	}
	return buf.String()
}

// decodeRTFFragment interprets RTF escape sequences within htmltag content.
func decodeRTFFragment(s string) string {
	var buf bytes.Buffer
	i := 0
	n := len(s)

	for i < n {
		if s[i] == '\\' {
			i++
			if i >= n {
				break
			}
			switch s[i] {
			case '\\':
				buf.WriteByte('\\')
				i++
			case '{':
				buf.WriteByte('{')
				i++
			case '}':
				buf.WriteByte('}')
				i++
			case '\'':
				// Hex escape: \'XX
				if i+2 < n {
					hi := unhex(s[i+1])
					lo := unhex(s[i+2])
					if hi >= 0 && lo >= 0 {
						buf.WriteByte(byte(hi<<4 | lo))
					}
					i += 3
				} else {
					i++
				}
			case '\r', '\n':
				// \<CR> and \<LF> are paragraph breaks in RTF — skip in HTML context.
				i++
			default:
				// Other control words like \par, \tab — skip the word + optional space.
				if s[i] == 'p' && i+3 < n && s[i:i+3] == "par" {
					i += 3
					if i < n && s[i] == ' ' {
						i++
					}
					buf.WriteString("\r\n")
				} else if s[i] == 't' && i+3 < n && s[i:i+3] == "tab" {
					i += 3
					if i < n && s[i] == ' ' {
						i++
					}
					buf.WriteByte('\t')
				} else if s[i] == 'l' && i+4 < n && s[i:i+4] == "line" {
					i += 4
					if i < n && s[i] == ' ' {
						i++
					}
					buf.WriteString("\r\n")
				} else {
					// Skip unknown control word.
					for i < n && ((s[i] >= 'a' && s[i] <= 'z') || (s[i] >= 'A' && s[i] <= 'Z')) {
						i++
					}
					// Skip optional numeric parameter.
					if i < n && (s[i] == '-' || (s[i] >= '0' && s[i] <= '9')) {
						if s[i] == '-' {
							i++
						}
						for i < n && s[i] >= '0' && s[i] <= '9' {
							i++
						}
					}
					// Skip optional trailing space (delimiter).
					if i < n && s[i] == ' ' {
						i++
					}
				}
			}
		} else if s[i] == '\r' || s[i] == '\n' {
			// Bare CR/LF in RTF is ignored.
			i++
		} else {
			buf.WriteByte(s[i])
			i++
		}
	}
	return buf.String()
}

// skipGroup advances past a brace-delimited group starting at pos.
func skipGroup(data []byte, pos, end int) int {
	if pos >= end || data[pos] != '{' {
		return pos + 1
	}
	depth := 0
	i := pos
	for i < end {
		if data[i] == '{' {
			depth++
		} else if data[i] == '}' {
			depth--
			if depth == 0 {
				return i + 1
			}
		}
		i++
	}
	return end
}

// skipControlWord advances past a backslash-prefixed control word.
func skipControlWord(data []byte, pos, end int) int {
	i := pos + 1 // skip the backslash
	if i >= end {
		return end
	}
	// Special characters: single-char control symbols.
	if !((data[i] >= 'a' && data[i] <= 'z') || (data[i] >= 'A' && data[i] <= 'Z')) {
		return i + 1
	}
	// Alphabetic control word.
	for i < end && ((data[i] >= 'a' && data[i] <= 'z') || (data[i] >= 'A' && data[i] <= 'Z')) {
		i++
	}
	// Optional numeric parameter (possibly negative).
	if i < end && (data[i] == '-' || (data[i] >= '0' && data[i] <= '9')) {
		if data[i] == '-' {
			i++
		}
		for i < end && data[i] >= '0' && data[i] <= '9' {
			i++
		}
	}
	// Optional delimiter space.
	if i < end && data[i] == ' ' {
		i++
	}
	return i
}

// unhex converts a hex digit character to its value, or -1 if invalid.
func unhex(c byte) int {
	switch {
	case c >= '0' && c <= '9':
		return int(c - '0')
	case c >= 'a' && c <= 'f':
		return int(c-'a') + 10
	case c >= 'A' && c <= 'F':
		return int(c-'A') + 10
	default:
		return -1
	}
}
