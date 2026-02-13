package web

import "embed"

// StaticFS holds the embedded static files (HTML, CSS, JS).
//
//go:embed static
var StaticFS embed.FS
