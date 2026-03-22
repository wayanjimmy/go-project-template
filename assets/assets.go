package assets

import "embed"

// PublicBuildFS contains embedded frontend assets for admin-tools.
//
//go:embed all:public/build
var PublicBuildFS embed.FS
