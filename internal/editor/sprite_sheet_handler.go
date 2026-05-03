package editor

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/kijimaD/ruins/internal/raw"
)

type spriteSheetItem struct {
	Index  int
	Sheet  raw.SpriteSheet
	Active bool
}

type spriteSheetEditData struct {
	Index int
	Sheet raw.SpriteSheet
}

type spriteSheetsData struct {
	Items []spriteSheetItem
	Edit  *spriteSheetEditData
}

func (s *Server) handleSpriteSheets(w http.ResponseWriter, _ *http.Request) {
	s.renderSpriteSheets(w, -1)
}

func (s *Server) renderSpriteSheets(w http.ResponseWriter, activeIndex int) {
	sheets := s.store.SpriteSheets()
	rows := make([]spriteSheetItem, len(sheets))
	for i, sh := range sheets {
		rows[i] = spriteSheetItem{Index: i, Sheet: sh, Active: i == activeIndex}
	}
	data := spriteSheetsData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(sheets) {
		data.Edit = &spriteSheetEditData{Index: activeIndex, Sheet: sheets[activeIndex]}
	}
	if err := s.templates.ExecuteTemplate(w, "sprite-sheets", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) renderSpriteSheetPartial(w http.ResponseWriter, activeIndex int) {
	sheets := s.store.SpriteSheets()
	rows := make([]spriteSheetItem, len(sheets))
	for i, sh := range sheets {
		rows[i] = spriteSheetItem{Index: i, Sheet: sh, Active: i == activeIndex}
	}
	data := spriteSheetsData{Items: rows}
	if activeIndex >= 0 && activeIndex < len(sheets) {
		ed := spriteSheetEditData{Index: activeIndex, Sheet: sheets[activeIndex]}
		if err := s.templates.ExecuteTemplate(w, "sprite-sheet-edit", ed); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if _, err := fmt.Fprint(w, `<div class="text-secondary mt-5 text-center">スプライトシートを選択してください</div>`); err != nil {
			log.Printf("レスポンス書き込みに失敗: %v", err)
		}
	}
	if err := s.templates.ExecuteTemplate(w, "ss-list-oob", data); err != nil {
		log.Printf("サイドバーOOBレンダリングに失敗: %v", err)
	}
	if err := s.templates.ExecuteTemplate(w, "ss-count-oob", data); err != nil {
		log.Printf("件数OOBレンダリングに失敗: %v", err)
	}
}

func (s *Server) handleSpriteSheetEdit(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	ss, err := s.store.SpriteSheetByIndex(index)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	data := spriteSheetEditData{Index: index, Sheet: ss}
	if err := s.templates.ExecuteTemplate(w, "sprite-sheet-edit", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) findSpriteSheetIndex(name string) int {
	for i, sh := range s.store.SpriteSheets() {
		if sh.Name == name {
			return i
		}
	}
	return -1
}

func (s *Server) handleSpriteSheetUpdate(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	ss := raw.SpriteSheet{Name: r.FormValue("name"), Path: r.FormValue("path")}
	if err := s.store.UpdateSpriteSheet(index, ss); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderSpriteSheetPartial(w, s.findSpriteSheetIndex(ss.Name))
}

func (s *Server) handleSpriteSheetCreate(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "名前は必須です", http.StatusBadRequest)
		return
	}
	ss := raw.SpriteSheet{Name: name}
	if err := s.store.AddSpriteSheet(ss); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderSpriteSheetPartial(w, s.findSpriteSheetIndex(name))
}

func (s *Server) handleSpriteSheetDelete(w http.ResponseWriter, r *http.Request) {
	index, err := strconv.Atoi(r.PathValue("index"))
	if err != nil {
		http.Error(w, "無効なインデックス", http.StatusBadRequest)
		return
	}
	if err := s.store.DeleteSpriteSheet(index); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	s.renderSpriteSheetPartial(w, -1)
}

// cutterCell はスプライトカッターの1セル分のデータ
type cutterCell struct {
	Index int
	Row   int
	Col   int
}

// cutterData はスプライトカッターテンプレートに渡すデータ
type cutterData struct {
	Uploaded bool
	Cols     int
	Rows     int
	CellSize int
	Cells    []cutterCell
}

func (s *Server) handleCutter(w http.ResponseWriter, _ *http.Request) {
	data := s.buildCutterData()
	if err := s.templates.ExecuteTemplate(w, "cutter", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *Server) handleCutterUpload(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "ファイルのパースに失敗", http.StatusBadRequest)
		return
	}
	file, _, err := r.FormFile("sheet")
	if err != nil {
		http.Error(w, "ファイルの読み込みに失敗", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := file.Close(); err != nil {
			log.Printf("アップロードファイルのクローズに失敗: %v", err)
		}
	}()

	img, err := png.Decode(file)
	if err != nil {
		http.Error(w, "PNG画像のデコードに失敗", http.StatusBadRequest)
		return
	}
	s.uploadedSheet = img

	w.Header().Set("HX-Redirect", "/cutter")
	w.WriteHeader(http.StatusOK)
}

// handleCutterPreview はアップロード済み画像をPNGで返す
func (s *Server) handleCutterPreview(w http.ResponseWriter, _ *http.Request) {
	if s.uploadedSheet == nil {
		http.NotFound(w, nil)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "no-cache")
	if err := png.Encode(w, s.uploadedSheet); err != nil {
		log.Printf("プレビュー画像の書き込みに失敗: %v", err)
	}
}

func (s *Server) handleCutterSave(w http.ResponseWriter, r *http.Request) {
	if s.uploadedSheet == nil {
		http.Error(w, "画像がアップロードされていません", http.StatusBadRequest)
		return
	}
	if s.outputDir == "" {
		http.Error(w, "出力先が設定されていません", http.StatusInternalServerError)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "フォームのパースに失敗", http.StatusBadRequest)
		return
	}

	bounds := s.uploadedSheet.Bounds()
	cellSize := 32
	cols := bounds.Dx() / cellSize
	rows := bounds.Dy() / cellSize
	saved := 0

	for row := range rows {
		for col := range cols {
			idx := row*cols + col
			name := strings.TrimSpace(r.FormValue(fmt.Sprintf("name_%d", idx)))
			if name == "" {
				continue
			}
			rect := image.Rect(col*cellSize, row*cellSize, (col+1)*cellSize, (row+1)*cellSize)
			cell := image.NewRGBA(image.Rect(0, 0, cellSize, cellSize))
			draw.Draw(cell, cell.Bounds(), s.uploadedSheet, rect.Min, draw.Src)

			// 完全に透明なセルはスキップする
			if isTransparent(cell) {
				continue
			}

			filename := filepath.Base(name) + "_.png"
			path := filepath.Join(s.outputDir, filename)
			if err := savePNG(path, cell); err != nil {
				http.Error(w, fmt.Sprintf("%s の保存に失敗: %v", name, err), http.StatusInternalServerError)
				return
			}
			saved++
		}
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if _, err := fmt.Fprintf(w, `<div class="alert alert-success">%d 個のスプライトを保存しました</div>`, saved); err != nil {
		log.Printf("レスポンス書き込みに失敗: %v", err)
	}
}

func (s *Server) buildCutterData() cutterData {
	if s.uploadedSheet == nil {
		return cutterData{Uploaded: false}
	}
	bounds := s.uploadedSheet.Bounds()
	cellSize := 32
	cols := bounds.Dx() / cellSize
	rows := bounds.Dy() / cellSize
	cells := make([]cutterCell, 0, cols*rows)
	for row := range rows {
		for col := range cols {
			cells = append(cells, cutterCell{
				Index: row*cols + col,
				Row:   row,
				Col:   col,
			})
		}
	}
	return cutterData{
		Uploaded: true,
		Cols:     cols,
		Rows:     rows,
		CellSize: cellSize,
		Cells:    cells,
	}
}

// isTransparent は画像が完全に透明かどうかを判定する
func isTransparent(img *image.RGBA) bool {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				return false
			}
		}
	}
	return true
}

// clampUint8 はintを0-255の範囲にクランプしてuint8に変換する
func clampUint8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func savePNG(path string, img image.Image) error {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}
