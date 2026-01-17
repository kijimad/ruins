package gamelog

import "image/color"

// LogFragment は色付きテキストの断片
type LogFragment struct {
	Color color.RGBA `json:"color"`
	Text  string     `json:"text"`
}
