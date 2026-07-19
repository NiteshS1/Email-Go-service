package routes

import "embed"

//go:embed spec/swagger.json spec/swagger.yaml
var specFS embed.FS
