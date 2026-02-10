package web

import "embed"

//go:embed templates/* templates/components/*
var Files embed.FS
