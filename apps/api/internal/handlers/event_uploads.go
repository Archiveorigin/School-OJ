package handlers

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/services"
	"strconv"
	"strings"
	"sync"
	"time"
)

const eventUploadChunkSize int64 = 1 << 20

var eventUploadLock sync.Mutex

func eventUploadRoot() string { return filepath.Join(os.TempDir(), "school-oj-event-uploads") }
func eventUploadDir(user uint, p eventPaper, id string) string {
	return filepath.Join(eventUploadRoot(), fmt.Sprint(user), p.kind, fmt.Sprint(p.id), id)
}

// Uploads are private to the author AND event. The public problem/ticket
// upload route is deliberately not reused to bypass its permission checks.
func (s Server) uploadEventTestChunk(c *gin.Context) {
	p, ok := s.eventPaperByParams(c, true)
	if !ok {
		return
	}
	revision, err := strconv.Atoi(c.Query("revision"))
	if err != nil {
		c.JSON(400, gin.H{"error": "缺少场次修订号"})
		return
	}
	if err = requireEventDraft(p, revision); err != nil {
		eventError(c, err)
		return
	}
	id := c.Param("upload_id")
	index, err := strconv.Atoi(c.Param("chunk_index"))
	if !problemTestUploadIDPattern.MatchString(id) || err != nil || index < 0 || index >= 128 {
		c.JSON(400, gin.H{"error": "分片编号无效"})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, eventUploadChunkSize)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(400, gin.H{"error": "单个分片不能超过 1 MiB"})
		return
	}
	u, _ := middleware.CurrentUser(c)
	eventUploadLock.Lock()
	defer eventUploadLock.Unlock()
	cleanupEventUploads()
	dir := eventUploadDir(u.ID, p, id)
	path := filepath.Join(dir, fmt.Sprintf("%03d.part", index))
	if existing, e := os.ReadFile(path); e == nil {
		if !bytes.Equal(existing, body) {
			c.JSON(409, gin.H{"error": "分片内容已存在且不同，请重新上传"})
			return
		}
		c.JSON(201, gin.H{"received": len(body)})
		return
	}
	var total int64
	var count int
	_ = filepath.WalkDir(eventUploadRoot(), func(name string, entry fs.DirEntry, e error) error {
		if e == nil && !entry.IsDir() && name != path {
			if info, e := entry.Info(); e == nil {
				total += info.Size()
				count++
			}
		}
		return nil
	})
	if total+int64(len(body)) > 512<<20 || count >= 4096 {
		c.JSON(429, gin.H{"error": "上传暂存空间繁忙，请稍后重试"})
		return
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		eventError(c, err)
		return
	}
	temp, err := os.CreateTemp(dir, ".chunk-")
	if err != nil {
		eventError(c, err)
		return
	}
	name := temp.Name()
	defer os.Remove(name)
	_, err = temp.Write(body)
	closeErr := temp.Close()
	if err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, path)
	}
	if err != nil {
		eventError(c, err)
		return
	}
	now := time.Now()
	_ = os.Chtimes(dir, now, now)
	c.JSON(201, gin.H{"received": len(body)})
}

func consumeEventTestUploads(user uint, p eventPaper, refs []services.TestUploadReference) ([]services.TestPointUploadFile, func(), error) {
	noop := func() {}
	if len(refs) > 400 {
		return nil, noop, eventFail(400, "测试点文件数量超出限制")
	}
	eventUploadLock.Lock()
	defer eventUploadLock.Unlock()
	var files []services.TestPointUploadFile
	var total int64
	seen := map[string]bool{}
	// Validate the entire plan before claiming any upload directories.
	for _, r := range refs {
		chunks := (r.Size + eventUploadChunkSize - 1) / eventUploadChunkSize
		if chunks == 0 {
			chunks = 1
		}
		total += r.Size
		if !problemTestUploadIDPattern.MatchString(r.ID) || seen[r.ID] || r.Size < 0 || r.ChunkCount != int(chunks) || chunks > 128 || total > services.MaxProblemTestFilesSize || r.Name == "" || filepath.Base(r.Name) != r.Name {
			return nil, noop, eventFail(400, "分片文件信息无效或总计超过 128 MiB")
		}
		seen[r.ID] = true
	}
	claimed := []string{}
	cleanup := func() {
		for _, dir := range claimed {
			_ = os.RemoveAll(dir)
		}
	}
	fail := func(e error) ([]services.TestPointUploadFile, func(), error) { cleanup(); return nil, noop, e }
	for _, r := range refs {
		dir := eventUploadDir(user, p, r.ID)
		claimedDir, err := os.MkdirTemp(filepath.Dir(dir), ".consume-")
		if err != nil {
			return fail(eventFail(400, "分片已过期，请重新上传"))
		}
		// Rename into a unique private directory; retries cannot consume it twice.
		_ = os.Remove(claimedDir)
		if err = os.Rename(dir, claimedDir); err != nil {
			return fail(eventFail(400, "分片不完整、已过期或正在使用，请重新上传"))
		}
		claimed = append(claimed, claimedDir)
		assembled := filepath.Join(claimedDir, "assembled")
		dst, err := os.OpenFile(assembled, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		if err != nil {
			return fail(err)
		}
		var size int64
		for i := 0; i < r.ChunkCount; i++ {
			part, e := os.Open(filepath.Join(claimedDir, fmt.Sprintf("%03d.part", i)))
			if e != nil {
				_ = dst.Close()
				return fail(eventFail(400, "分片不完整，请重新上传"))
			}
			n, e := io.Copy(dst, io.LimitReader(part, eventUploadChunkSize+1))
			_ = part.Close()
			size += n
			if e != nil || n > eventUploadChunkSize || i < r.ChunkCount-1 && n != eventUploadChunkSize {
				_ = dst.Close()
				return fail(eventFail(400, "分片长度不匹配"))
			}
		}
		if err = dst.Close(); err != nil {
			return fail(err)
		}
		if size != r.Size {
			return fail(eventFail(400, "分片总长度不匹配"))
		}
		files = append(files, services.TestPointUploadFile{Name: r.Name, Path: assembled})
	}
	return files, cleanup, nil
}

func cleanupEventUploads() {
	root := eventUploadRoot()
	cutoff := time.Now().Add(-time.Hour)
	_ = filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || !entry.IsDir() || (!problemTestUploadIDPattern.MatchString(entry.Name()) && !strings.HasPrefix(entry.Name(), ".consume-")) {
			return nil
		}
		if info, e := entry.Info(); e == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(path)
			return filepath.SkipDir
		}
		return nil
	})
}
