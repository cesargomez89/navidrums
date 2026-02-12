package web

import "embed"

//go:embed templates/* templates/components/* static/*
var Files embed.FS
