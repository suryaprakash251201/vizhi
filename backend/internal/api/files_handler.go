package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"vizhi/backend/internal/transfer"

	"github.com/go-chi/chi/v5"
)

type FilesHandler struct {
	tm *transfer.TransferManager
}

func NewFilesHandler(tm *transfer.TransferManager) *FilesHandler {
	return &FilesHandler{tm: tm}
}

func (h *FilesHandler) List(w http.ResponseWriter, r *http.Request) {
	files, err := h.tm.ListUploads()
	if err != nil {
		http.Error(w, `{"error":"failed to list files"}`, http.StatusInternalServerError)
		return
	}
	if files == nil {
		files = []transfer.FileInfo{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (h *FilesHandler) InitUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		FileName  string `json:"file_name"`
		TotalSize int64  `json:"total_size"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.FileName == "" || req.TotalSize <= 0 {
		http.Error(w, `{"error":"file_name and total_size required"}`, http.StatusBadRequest)
		return
	}

	session, err := h.tm.InitUpload(req.FileName, req.TotalSize)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(session)
}

func (h *FilesHandler) UploadChunk(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, `{"error":"failed to parse multipart form"}`, http.StatusBadRequest)
		return
	}

	sessionID := r.FormValue("session_id")
	chunkIdxStr := r.FormValue("chunk_index")
	checksum := r.FormValue("checksum_sha256")

	if sessionID == "" || chunkIdxStr == "" {
		http.Error(w, `{"error":"session_id and chunk_index required"}`, http.StatusBadRequest)
		return
	}

	chunkIndex, err := strconv.Atoi(chunkIdxStr)
	if err != nil {
		http.Error(w, `{"error":"invalid chunk_index"}`, http.StatusBadRequest)
		return
	}

	file, _, err := r.FormFile("chunk")
	if err != nil {
		http.Error(w, `{"error":"chunk file field required"}`, http.StatusBadRequest)
		return
	}
	defer file.Close()

	data, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, `{"error":"failed to read chunk data"}`, http.StatusInternalServerError)
		return
	}

	if err := h.tm.WriteChunk(sessionID, chunkIndex, data, checksum); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "chunk_received"})
}

func (h *FilesHandler) FinalizeUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if err := h.tm.FinalizeUpload(req.SessionID); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "upload_complete"})
}

func (h *FilesHandler) Download(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	if path == "" {
		http.Error(w, `{"error":"path required"}`, http.StatusBadRequest)
		return
	}

	reader, size, err := h.tm.Download(path)
	if err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusNotFound)
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, path))
	w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
	http.ServeContent(w, r, path, time.Now(), reader.(io.ReadSeeker))
}

func (h *FilesHandler) Delete(w http.ResponseWriter, r *http.Request) {
	path := chi.URLParam(r, "path")
	if path == "" {
		http.Error(w, `{"error":"path required"}`, http.StatusBadRequest)
		return
	}

	if err := h.tm.DeleteUploadedFile(path); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
