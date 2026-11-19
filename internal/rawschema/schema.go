package rawschema

import "embed"

// SchemaFiles は raw のスキーマの CUE のファイル
//
//go:embed *.cue
var SchemaFiles embed.FS
