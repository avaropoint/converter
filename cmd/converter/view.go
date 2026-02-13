// view.go implements the CLI "view" command that displays the structure
// and metadata of a TNEF file.

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/avaropoint/converter/formats"
	"github.com/avaropoint/converter/parsers/tnef"
)

// cmdView decodes a TNEF file and prints its structure to stdout.
func cmdView(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading %s: %v\n", path, err)
		os.Exit(1)
	}
	conv := formats.Detect(filepath.Base(path), data)
	if conv == nil {
		fmt.Fprintf(os.Stderr, "Unsupported file format: %s\n", filepath.Base(path))
		os.Exit(1)
	}
	if fi, err := os.Stat(path); err == nil {
		fmt.Printf("File:        %s (%s)\n", filepath.Base(path), humanSize(int(fi.Size())))
	} else {
		fmt.Printf("File:        %s\n", filepath.Base(path))
	}
	fmt.Printf("Format:      %s\n", conv.Name())
	fmt.Println(strings.Repeat("─", 60))
	msg, err := tnef.Decode(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error decoding: %v\n", err)
		os.Exit(1)
	}
	printMessage(msg, "")
}

// methodStr returns a human-readable label for an attachment method constant.
func methodStr(m int) string {
	switch m {
	case tnef.AttachByValue:
		return "file"
	case tnef.AttachEmbeddedMsg:
		return "embedded message"
	case tnef.AttachOLE:
		return "OLE object"
	default:
		return fmt.Sprintf("method=%d", m)
	}
}

// printMessage recursively prints a decoded TNEF message and its attachments.
func printMessage(msg *tnef.Message, indent string) {
	divider := indent + strings.Repeat("─", 60-len(indent))
	fields := []struct {
		label string
		value string
	}{
		{"Subject", msg.GetAttrString(tnef.MAPISubject)},
		{"From", msg.GetAttrString(tnef.MAPISenderName)},
		{"From Email", msg.GetAttrString(tnef.MAPISenderEmail)},
		{"To", msg.GetAttrString(tnef.MAPIDisplayTo)},
		{"CC", msg.GetAttrString(tnef.MAPIDisplayCc)},
	}
	for _, f := range fields {
		if f.value != "" {
			fmt.Printf("%s%-13s%s\n", indent, f.label+":", f.value)
		}
	}
	if len(msg.Body) > 0 {
		fmt.Printf("%sBody:        Plain text (%s)\n", indent, humanSize(len(msg.Body)))
	}
	if len(msg.BodyHTML) > 0 {
		fmt.Printf("%sBody HTML:   Yes (%s)\n", indent, humanSize(len(msg.BodyHTML)))
	}
	if len(msg.BodyRTF) > 0 {
		if len(msg.BodyRTFHTML) > 0 {
			fmt.Printf("%sBody RTF:    Yes (%s, encapsulated HTML: %s)\n", indent, humanSize(len(msg.BodyRTF)), humanSize(len(msg.BodyRTFHTML)))
		} else {
			fmt.Printf("%sBody RTF:    Yes (%s)\n", indent, humanSize(len(msg.BodyRTF)))
		}
	}
	if len(msg.Attachments) == 0 {
		fmt.Printf("%sAttachments: None\n", indent)
		return
	}
	fmt.Printf("%sAttachments: %d item(s)\n", indent, len(msg.Attachments))
	fmt.Println(divider)
	for i, att := range msg.Attachments {
		name := att.Filename()
		fmt.Printf("%s  %d. %-36s %8s  [%s]\n", indent, i+1, name, humanSize(len(att.Data)), methodStr(att.Method))
		if att.EmbeddedMsg != nil {
			fmt.Printf("%s     └─ Embedded message:\n", indent)
			printMessage(att.EmbeddedMsg, indent+"        ")
		}
	}
}
