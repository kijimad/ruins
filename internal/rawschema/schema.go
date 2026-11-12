package rawschema

import "embed"

//go:generate go run ./gen

// SchemaFiles は raw のスキーマの CUE のファイル
//
//go:embed *.cue
var SchemaFiles embed.FS
