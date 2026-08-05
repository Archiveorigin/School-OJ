package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"school-oj/apps/api/internal/middleware"
	"school-oj/apps/api/internal/services"
)

const problemTestUploadChunkSize int64 = 4 << 20

var problemTestUploadIDPattern = regexp.MustCompile(`^[a-f0-9-]{20,64}$`)

func (s Server) uploadProblemTestChunk(c *gin.Context) {
	user, _ := middleware.CurrentUser(c)
	if !canCreateProblems(user) {
		c.JSON(http.StatusForbidden, gin.H{"error": "problem author permission is required"})
		return
	}
	uploadID := c.Param("upload_id")
	chunkIndex, err := strconv.Atoi(c.Param("chunk_index"))
	if !problemTestUploadIDPattern.MatchString(uploadID) || err != nil || chunkIndex < 0 || chunkIndex >= 1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid upload chunk"})
		return
	}

	cleanupExpiredProblemTestUploads(user.ID)
	dir := problemTestUploadDir(user.ID, uploadID)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	path := filepath.Join(dir, fmt.Sprintf("%06d.part", chunkIndex))
	dst, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	written, copyErr := io.Copy(dst, io.LimitReader(c.Request.Body, problemTestUploadChunkSize+1))
	closeErr := dst.Close()
	if copyErr != nil || closeErr != nil || written <= 0 || written > problemTestUploadChunkSize {
		_ = os.Remove(path)
		if copyErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": copyErr.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "upload chunk is empty or too large"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"received": written})
}

func consumeProblemTestUploads(userID uint, refs []services.TestUploadReference) ([]services.TestPointUploadFile, func(), error) {
	cleanup := func() {
		for _, ref := range refs {
			if problemTestUploadIDPattern.MatchString(ref.ID) {
				_ = os.RemoveAll(problemTestUploadDir(userID, ref.ID))
			}
		}
	}
	if len(refs) == 0 {
		return nil, cleanup, nil
	}
	seen := map[string]bool{}
	uploads := make([]services.TestPointUploadFile, 0, len(refs))
	var totalSize int64
	for _, ref := range refs {
		if !problemTestUploadIDPattern.MatchString(ref.ID) || seen[ref.ID] || ref.ChunkCount <= 0 || ref.ChunkCount > 1024 || ref.Size <= 0 {
			cleanup()
			return nil, func() {}, fmt.Errorf("invalid test upload reference")
		}
		seen[ref.ID] = true
		dir := problemTestUploadDir(userID, ref.ID)
		assembledPath := filepath.Join(dir, "assembled.upload")
		dst, err := os.OpenFile(assembledPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
		if err != nil {
			cleanup()
			return nil, func() {}, err
		}
		var assembledSize int64
		for index := 0; index < ref.ChunkCount; index++ {
			partPath := filepath.Join(dir, fmt.Sprintf("%06d.part", index))
			part, err := os.Open(partPath)
			if err != nil {
				_ = dst.Close()
				cleanup()
				return nil, func() {}, fmt.Errorf("test upload is incomplete")
			}
			copied, copyErr := io.Copy(dst, io.LimitReader(part, problemTestUploadChunkSize+1))
			_ = part.Close()
			if copyErr != nil || copied <= 0 || copied > problemTestUploadChunkSize {
				_ = dst.Close()
				cleanup()
				return nil, func() {}, fmt.Errorf("invalid test upload chunk")
			}
			assembledSize += copied
		}
		if err := dst.Close(); err != nil {
			cleanup()
			return nil, func() {}, err
		}
		totalSize += assembledSize
		if assembledSize != ref.Size || totalSize > services.MaxProblemTestFilesSize {
			cleanup()
			return nil, func() {}, fmt.Errorf("test files are too large or incomplete")
		}
		uploads = append(uploads, services.TestPointUploadFile{Name: filepath.Base(ref.Name), Path: assembledPath})
	}
	return uploads, cleanup, nil
}

func problemTestUploadDir(userID uint, uploadID string) string {
	return filepath.Join(os.TempDir(), "school-oj-test-uploads", strconv.FormatUint(uint64(userID), 10), uploadID)
}

func cleanupExpiredProblemTestUploads(userID uint) {
	root := filepath.Dir(problemTestUploadDir(userID, "placeholder"))
	entries, err := os.ReadDir(root)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-24 * time.Hour)
	for _, entry := range entries {
		if !entry.IsDir() || !problemTestUploadIDPattern.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(root, entry.Name()))
		}
	}
}
