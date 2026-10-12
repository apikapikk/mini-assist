// Package web menyediakan berkas statis dan template HTML yang di-embed ke binary Go.
package web

import "embed"

// StaticFiles berisi aset statis seperti CSS, JS, ikon, dan manifest PWA.
//
//go:embed static/*
var StaticFiles embed.FS

// Templates berisi file HTML untuk frontend (index, login, dll).
//
//go:embed templates/*
var Templates embed.FS
