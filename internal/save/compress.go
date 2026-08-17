package save

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

// gzipBytes は data を gzip 圧縮する。gzip は magic 1f 8b の標準形式で、
// デスクトップのセーブファイルは gunzip や zcat でそのまま解凍できる。
func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gw, err := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if err != nil {
		return nil, fmt.Errorf("failed to init gzip writer: %w", err)
	}
	if _, err := gw.Write(data); err != nil {
		return nil, fmt.Errorf("failed to compress save data: %w", err)
	}
	if err := gw.Close(); err != nil {
		return nil, fmt.Errorf("failed to finalize compression: %w", err)
	}
	return buf.Bytes(), nil
}

// gunzipBytes は gzipBytes の逆変換を行う。
func gunzipBytes(data []byte) ([]byte, error) {
	gr, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to init gzip reader: %w", err)
	}
	defer func() { _ = gr.Close() }()
	out, err := io.ReadAll(gr)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress save data: %w", err)
	}
	return out, nil
}
