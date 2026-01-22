package maptemplate

import (
	"fmt"
	"os"
	"strings"

	"github.com/pelletier/go-toml/v2"
)

// FacilityTemplate は施設テンプレート定義
type FacilityTemplate struct {
	Type     string   `toml:"type"`     // 施設タイプ（例: "military_factory"）
	Name     string   `toml:"name"`     // 表示名（例: "軍需工場・小型"）
	Weight   int      `toml:"weight"`   // 出現確率の重み
	Size     [2]int   `toml:"size"`     // [幅, 高さ]
	Entrance [2]int   `toml:"entrance"` // 入口座標 [x, y]
	Palettes []string `toml:"palettes"` // 使用するパレットID
	Map      string   `toml:"map"`      // ASCIIマップ
}

// FacilityTemplateFile はTOMLファイルのルート構造
type FacilityTemplateFile struct {
	Facilities []FacilityTemplate `toml:"facility"`
}

// TemplateLoader はテンプレート定義の読み込みを担当する
type TemplateLoader struct{}

// NewTemplateLoader はTemplateLoaderを生成する
func NewTemplateLoader() *TemplateLoader {
	return &TemplateLoader{}
}

// LoadFromFile はTOMLファイルから施設テンプレート定義を読み込む
func (l *TemplateLoader) LoadFromFile(path string) ([]FacilityTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("テンプレートファイル読み込みエラー: %w", err)
	}

	var file FacilityTemplateFile
	if err := toml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("テンプレートTOMLパースエラー: %w", err)
	}

	for i := range file.Facilities {
		if err := l.validate(&file.Facilities[i]); err != nil {
			return nil, fmt.Errorf("テンプレート%d検証エラー: %w", i, err)
		}
	}

	return file.Facilities, nil
}

// validate はテンプレート定義の妥当性を検証する
func (l *TemplateLoader) validate(t *FacilityTemplate) error {
	if t.Type == "" {
		return fmt.Errorf("施設タイプが空です")
	}

	if t.Name == "" {
		return fmt.Errorf("施設名が空です")
	}

	if t.Weight <= 0 {
		return fmt.Errorf("重みは正の整数である必要があります: %d", t.Weight)
	}

	if t.Size[0] <= 0 || t.Size[1] <= 0 {
		return fmt.Errorf("サイズは正の整数である必要があります: %v", t.Size)
	}

	// マップの検証
	return l.validateMap(t)
}

// validateMap はマップの妥当性を検証する
func (l *TemplateLoader) validateMap(t *FacilityTemplate) error {
	if t.Map == "" {
		return fmt.Errorf("マップが空です")
	}

	lines := strings.Split(strings.TrimSpace(t.Map), "\n")
	if len(lines) == 0 {
		return fmt.Errorf("マップが空です")
	}

	// 最初の行の長さを基準とする
	expectedWidth := len(lines[0])
	if expectedWidth == 0 {
		return fmt.Errorf("マップの行が空です")
	}

	// すべての行が同じ長さであることを確認
	for i, line := range lines {
		if len(line) != expectedWidth {
			return fmt.Errorf("マップの行%dの長さが不一致です: 期待%d、実際%d", i, expectedWidth, len(line))
		}
	}

	// マップサイズと定義サイズの一致確認
	actualWidth := expectedWidth
	actualHeight := len(lines)
	if actualWidth != t.Size[0] || actualHeight != t.Size[1] {
		return fmt.Errorf("マップの実サイズ[%d, %d]が定義サイズ%vと一致しません", actualWidth, actualHeight, t.Size)
	}

	return nil
}

// GetMapLines はマップを行の配列として取得する
func (t *FacilityTemplate) GetMapLines() []string {
	return strings.Split(strings.TrimSpace(t.Map), "\n")
}

// GetCharAt は指定座標の文字を取得する
func (t *FacilityTemplate) GetCharAt(x, y int) (string, error) {
	lines := t.GetMapLines()
	if y < 0 || y >= len(lines) {
		return "", fmt.Errorf("y座標が範囲外です: %d", y)
	}
	if x < 0 || x >= len(lines[y]) {
		return "", fmt.Errorf("x座標が範囲外です: %d", x)
	}
	return string(lines[y][x]), nil
}
