// Package tnef decodes Microsoft TNEF (Transport Neutral Encapsulation
// Format) streams, commonly found as winmail.dat email attachments.
//
// It handles the full format including MAPI property streams, embedded
// messages, LZFu-compressed RTF (MS-OXRTFCP), and HTML de-encapsulation
// from Outlook's fromhtml1 format (MS-OXRTFEX).
//
// Zero external dependencies.
package tnef

import "errors"

// TNEF stream constants.
const (
	tnefSignature = 0x223e9f78
	lvlMessage    = 0x01
	lvlAttachment = 0x02
)

// TNEF attribute IDs.
const (
	attrAttachData     = 0x800F
	attrAttachTitle    = 0x8010
	attrAttachRendData = 0x9002
	attrMAPIProps      = 0x9003
	attrAttachment     = 0x9005
)

// MAPI property IDs used during decoding.
const (
	MAPISubject         = 0x0037 // PR_SUBJECT
	MAPISenderName      = 0x0C1A // PR_SENDER_NAME
	MAPISenderEmail     = 0x0C1F // PR_SENDER_EMAIL_ADDRESS
	MAPIDisplayTo       = 0x0E04 // PR_DISPLAY_TO
	MAPIDisplayCc       = 0x0E03 // PR_DISPLAY_CC
	MAPIBody            = 0x1000 // PR_BODY
	MAPIRtfCompressed   = 0x1009 // PR_RTF_COMPRESSED
	MAPIBodyHTML        = 0x1013 // PR_BODY_HTML
	MAPIAttachDataObj   = 0x3701 // PR_ATTACH_DATA_OBJ
	MAPIAttachFilename  = 0x3704 // PR_ATTACH_FILENAME
	MAPIAttachMethod    = 0x3705 // PR_ATTACH_METHOD
	MAPIAttachLongFname = 0x3707 // PR_ATTACH_LONG_FILENAME
	MAPIAttachMimeTag   = 0x370E // PR_ATTACH_MIME_TAG
	MAPIAttachContentID = 0x3712 // PR_ATTACH_CONTENT_ID
)

// Attachment method constants from PR_ATTACH_METHOD.
const (
	AttachByValue     = 1
	AttachEmbeddedMsg = 5
	AttachOLE         = 6
)

// ErrBadSignature is returned when the input is not a valid TNEF stream.
var ErrBadSignature = errors.New("not a valid TNEF file")
