package main

import "embed"

//go:embed all:frontend/dist
var assets embed.FS

//go:embed db/migrations/*.sql
var migrations embed.FS
