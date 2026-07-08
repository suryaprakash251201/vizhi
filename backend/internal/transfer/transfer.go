package transfer

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

var (
	ErrInvalidPath      = errors.New("path traversal denied")
	ErrUploadIncomplete = errors.New("upload incomplete — checksum mismatch")
	ErrChunkOutOfOrder  = errors.New("chunk out of order")
)

type UploadSession struct {
	ID           string    `json:"id"`
	FileName     string    `json:"file_name"`
	FilePath     string    `json:"-"`
	TotalSize    int64     `json:"total_size"`
	TotalChunks  int       `json:"total_chunks"`
	ReceivedSize int64     `json:"received_size"`
	Received     int       `json:"received_chunks"`
	Chunks       []bool    `json:"-"`
	Checksum     string    `json:"checksum_sha256"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	mu           sync.Mutex
}

type TransferManager struct {
	baseDir   string
	maxSize   int64
	chunkSize int64
	sessions  map[string]*UploadSession
	mu        sync.RWMutex
}

func NewTransferManager(baseDir string, maxSize, chunkSize int64) *TransferManager {
	os.MkdirAll(baseDir, 0750)
	return &TransferManager{
		baseDir:   baseDir,
		maxSize:   maxSize,
		chunkSize: chunkSize,
		sessions:  make(map[string]*UploadSession),
	}
}

func (tm *TransferManager) safePath(base, requested string) (string, error) {
	clean := filepath.Clean(requested)
	if strings.Contains(clean, "..") || strings.HasPrefix(clean, "/") {
		return "", ErrInvalidPath
	}
	full := filepath.Join(base, clean)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", err
	}
	absBase, err := filepath.Abs(base)
	if err != nil {
		return "", err
	}
	if !strings.HasPrefix(abs, absBase) {
		return "", ErrInvalidPath
	}
	return abs, nil
}

func (tm *TransferManager) InitUpload(fileName string, totalSize int64) (*UploadSession, error) {
	if totalSize > tm.maxSize {
		return nil, fmt.Errorf("file too large: %d bytes exceeds limit of %d bytes", totalSize, tm.maxSize)
	}

	fileName = filepath.Base(fileName)
	if fileName == "" || fileName == "." || fileName == "/" {
		return nil, fmt.Errorf("invalid file name")
	}

	safePath, err := tm.safePath(tm.baseDir, fileName)
	if err != nil {
		return nil, err
	}

	totalChunks := int(totalSize / tm.chunkSize)
	if totalSize%tm.chunkSize != 0 {
		totalChunks++
	}

	session := &UploadSession{
		ID:          uuid.New().String(),
		FileName:    fileName,
		FilePath:    safePath,
		TotalSize:   totalSize,
		TotalChunks: totalChunks,
		Chunks:      make([]bool, totalChunks),
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	tm.mu.Lock()
	tm.sessions[session.ID] = session
	tm.mu.Unlock()

	return session, nil
}

func (tm *TransferManager) WriteChunk(sessionID string, chunkIndex int, data []byte, checksum string) error {
	tm.mu.RLock()
	session, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	if chunkIndex < 0 || chunkIndex >= session.TotalChunks {
		return fmt.Errorf("chunk index %d out of range [0, %d)", chunkIndex, session.TotalChunks)
	}

	if checksum != "" {
		h := sha256.Sum256(data)
		if hex.EncodeToString(h[:]) != checksum {
			return fmt.Errorf("checksum mismatch for chunk %d", chunkIndex)
		}
	}

	chunkPath := fmt.Sprintf("%s.chunk.%04d", session.FilePath, chunkIndex)
	if err := os.WriteFile(chunkPath, data, 0640); err != nil {
		return fmt.Errorf("write chunk %d: %w", chunkIndex, err)
	}

	session.Chunks[chunkIndex] = true
	session.ReceivedSize += int64(len(data))
	session.Received++
	session.UpdatedAt = time.Now()

	return nil
}

func (tm *TransferManager) FinalizeUpload(sessionID string) error {
	tm.mu.RLock()
	session, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()
	if !ok {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.mu.Lock()
	defer session.mu.Unlock()

	for i, received := range session.Chunks {
		if !received {
			return fmt.Errorf("missing chunk %d/%d", i, session.TotalChunks)
		}
	}

	out, err := os.Create(session.FilePath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}
	defer out.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(out, hasher)

	for i := 0; i < session.TotalChunks; i++ {
		chunkPath := fmt.Sprintf("%s.chunk.%04d", session.FilePath, i)
		data, err := os.ReadFile(chunkPath)
		if err != nil {
			return fmt.Errorf("read chunk %d: %w", i, err)
		}
		if _, err := writer.Write(data); err != nil {
			return fmt.Errorf("write chunk %d to output: %w", i, err)
		}
		os.Remove(chunkPath)
	}

	session.Checksum = hex.EncodeToString(hasher.Sum(nil))

	go func() {
		time.Sleep(5 * time.Minute)
		tm.mu.Lock()
		delete(tm.sessions, sessionID)
		tm.mu.Unlock()
	}()

	return nil
}

func (tm *TransferManager) GetSession(sessionID string) (*UploadSession, error) {
	tm.mu.RLock()
	session, ok := tm.sessions[sessionID]
	tm.mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}
	return session, nil
}

func (tm *TransferManager) Download(path string) (io.ReadCloser, int64, error) {
	safePath, err := tm.safePath(tm.baseDir, path)
	if err != nil {
		return nil, 0, err
	}

	stat, err := os.Stat(safePath)
	if err != nil {
		return nil, 0, fmt.Errorf("file not found: %w", err)
	}
	if stat.IsDir() {
		return nil, 0, fmt.Errorf("path is a directory")
	}

	f, err := os.Open(safePath)
	if err != nil {
		return nil, 0, fmt.Errorf("open file: %w", err)
	}

	return f, stat.Size(), nil
}

func (tm *TransferManager) DeleteUploadedFile(path string) error {
	safePath, err := tm.safePath(tm.baseDir, path)
	if err != nil {
		return err
	}
	return os.Remove(safePath)
}

func (tm *TransferManager) ListUploads() ([]FileInfo, error) {
	entries, err := os.ReadDir(tm.baseDir)
	if err != nil {
		return nil, fmt.Errorf("read upload dir: %w", err)
	}

	var files []FileInfo
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, FileInfo{
			Name:    e.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})
	}
	return files, nil
}

type FileInfo struct {
	Name    string    `json:"name"`
	Size    int64     `json:"size"`
	ModTime time.Time `json:"mod_time"`
}

func FormatJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
