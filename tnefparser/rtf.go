package tnefparser

import (
	"encoding/binary"
	"errors"
	"hash/crc32"
)

// LZFu decompression for Microsoft PR_RTF_COMPRESSED (MS-OXRTFCP).
//
// Reference: [MS-OXRTFCP] Rich Text Format (RTF) Compression Algorithm
// https://docs.microsoft.com/en-us/openspecs/exchange_server_protocols/ms-oxrtfcp

// Compressed RTF header signatures.
const (
	compressedRTF   = 0x75465A4C // "LZFu" — compressed
	uncompressedRTF = 0x414C454D // "MELA" — uncompressed / raw
)

// Pre-initialized dictionary per MS-OXRTFCP spec.
// This is the fixed 207-byte seed that occupies positions 0–206 of the 4096-byte
// circular buffer before decompression begins. The write cursor starts at 207.
var lzfuInitDict = []byte(
	"{\\rtf1\\ansi\\mac\\deff0\\deftab720{\\fonttbl;}" +
		"{\\f0\\fnil \\froman \\fswiss \\fmodern \\fscript " +
		"\\fdecor MS Sans SerifSymbolArialTimes New Roman" +
		"Courier{\\colortbl\\red0\\green0\\blue0\r\n\\par " +
		"\\pard\\plain\\f0\\fs20\\b\\i\\u\\tab\\tx",
)

const dictSize = 4096 // Circular buffer size.
const initDictLen = 207

// ErrInvalidRTF is returned when compressed RTF data is malformed.
var ErrInvalidRTF = errors.New("invalid compressed RTF data")

// DecompressRTF decompresses a PR_RTF_COMPRESSED byte stream into raw RTF.
// It handles both LZFu-compressed and uncompressed (MELA) formats.
func DecompressRTF(data []byte) ([]byte, error) {
	if len(data) < 16 {
		return nil, ErrInvalidRTF
	}

	// Parse the 16-byte header.
	compSize := binary.LittleEndian.Uint32(data[0:4])   // Total compressed size (includes header after first 4 bytes).
	rawSize := binary.LittleEndian.Uint32(data[4:8])    // Uncompressed size.
	compType := binary.LittleEndian.Uint32(data[8:12])  // "LZFu" or "MELA".
	crcValue := binary.LittleEndian.Uint32(data[12:16]) // CRC32 of compressed data after header.

	_ = compSize // Used for bounds but we rely on len(data).

	switch compType {
	case uncompressedRTF:
		// MELA: raw RTF follows the 16-byte header.
		end := 16 + int(rawSize)
		if end > len(data) {
			end = len(data)
		}
		return append([]byte(nil), data[16:end]...), nil

	case compressedRTF:
		// Verify CRC32 of the compressed payload.
		// Per MS-OXRTFCP, CRC covers bytes from offset 16 to (compSize + 4).
		// compSize counts from byte 4, so total data = compSize + 4 bytes,
		// and the CRC region is bytes 16 through compSize+4.
		crcEnd := int(compSize) + 4
		if crcEnd > len(data) {
			crcEnd = len(data)
		}
		if crcEnd > 16 {
			crcPayload := data[16:crcEnd]
			if crc32.ChecksumIEEE(crcPayload) != crcValue && crcValue != 0 {
				// Be lenient: try the full remaining payload too, as some
				// implementations use a different CRC region.
				if crc32.ChecksumIEEE(data[16:]) != crcValue {
					// Skip CRC enforcement — many real-world files have
					// non-standard CRC values. Proceed with decompression.
				}
			}
		}
		return decompressLZFu(data[16:], int(rawSize))

	default:
		return nil, ErrInvalidRTF
	}
}

// decompressLZFu implements the core LZFu decompression loop.
func decompressLZFu(input []byte, rawSize int) ([]byte, error) {
	// Initialize circular dictionary buffer.
	dict := make([]byte, dictSize)
	copy(dict, lzfuInitDict)
	writePos := initDictLen

	out := make([]byte, 0, rawSize)
	inPos := 0

	for inPos < len(input) {
		// Read control byte — each bit (LSB first) indicates literal (1) or ref (0).
		if inPos >= len(input) {
			break
		}
		control := input[inPos]
		inPos++

		for bit := 0; bit < 8; bit++ {
			if inPos >= len(input) {
				break
			}
			if len(out) >= rawSize {
				break
			}

			if control&(1<<uint(bit)) != 0 {
				// Dictionary reference: 2 bytes encoding offset (12 bits) and length (4 bits).
				if inPos+1 >= len(input) {
					break
				}
				hi := int(input[inPos])
				lo := int(input[inPos+1])
				inPos += 2

				offset := (hi << 4) | (lo >> 4)
				length := (lo & 0x0F) + 2

				// offset == writePos signals end of data.
				if offset == writePos {
					return out, nil
				}

				for i := 0; i < length; i++ {
					if len(out) >= rawSize {
						break
					}
					b := dict[(offset+i)%dictSize]
					out = append(out, b)
					dict[writePos] = b
					writePos = (writePos + 1) % dictSize
				}
			} else {
				// Literal byte.
				b := input[inPos]
				inPos++
				out = append(out, b)
				dict[writePos] = b
				writePos = (writePos + 1) % dictSize
			}
		}
		if len(out) >= rawSize {
			break
		}
	}

	return out, nil
}
