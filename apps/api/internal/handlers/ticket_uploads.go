package handlers

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	"github.com/gin-gonic/gin"
)

func cleanupTicketMultipart(c *gin.Context) {
	if c.Request.MultipartForm != nil {
		_ = c.Request.MultipartForm.RemoveAll()
	}
}

// Inspect in place: never extract ZIP paths or retain decompressed test data in memory.
// The original package remains a private attachment for the reviewer.
func inspectTicketZIP(ctx context.Context, src io.ReaderAt, size int64) (int64, error) {
	z, err := zip.NewReader(src, size)
	if err != nil {
		return 0, errors.New("无法读取 ZIP 测试包")
	}
	if len(z.File) > 1024 {
		return 0, errors.New("测试包最多包含 1024 个条目")
	}
	seen, pairs := map[string]bool{}, map[string]uint8{}
	var total int64
	for _, f := range z.File {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		name := strings.ReplaceAll(f.Name, "\\", "/")
		if strings.HasPrefix(name, "/") || strings.ContainsAny(name, ":\x00") || path.Clean(name) == ".." || strings.HasPrefix(path.Clean(name), "../") {
			return 0, errors.New("测试包包含不安全的文件路径")
		}
		if f.FileInfo().IsDir() {
			continue
		}
		if !f.Mode().IsRegular() {
			return 0, errors.New("测试包不支持链接或特殊文件")
		}
		name = strings.ToLower(path.Clean(name))
		if seen[name] {
			return 0, fmt.Errorf("测试包文件名重复：%s", name)
		}
		seen[name] = true
		if f.UncompressedSize64 > uint64(problemTicketAttachmentLimit) {
			return 0, errors.New("测试包内单个文件不能超过 32 MiB")
		}
		total += int64(f.UncompressedSize64)
		if total > ticketTotalAttachmentLimit {
			return 0, errors.New("测试包解压后不能超过 96 MiB")
		}
		ext := path.Ext(name)
		if ext == ".in" || ext == ".out" {
			key := strings.TrimSuffix(name, ext)
			if ext == ".in" {
				pairs[key] |= 1
			} else {
				pairs[key] |= 2
			}
		}
		r, err := f.Open()
		if err != nil {
			return 0, errors.New("测试包存在无法读取的条目")
		}
		n, readErr := io.Copy(io.Discard, io.LimitReader(ticketContextReader{ctx, r}, int64(f.UncompressedSize64)+1))
		_ = r.Close()
		if readErr != nil || n != int64(f.UncompressedSize64) {
			return 0, errors.New("测试包校验失败，文件损坏或大小不符")
		}
	}
	if len(pairs) == 0 {
		return 0, errors.New("测试包需要包含同名配对的 .in 和 .out 文件")
	}
	for name, mask := range pairs {
		if mask != 3 {
			return 0, fmt.Errorf("测试点 %s 缺少配对的 .in 或 .out 文件", name)
		}
	}
	return total, nil
}

type ticketContextReader struct {
	ctx    context.Context
	reader io.Reader
}

func (r ticketContextReader) Read(p []byte) (int, error) {
	if err := r.ctx.Err(); err != nil {
		return 0, err
	}
	return r.reader.Read(p)
}
