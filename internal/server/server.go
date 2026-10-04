package server

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/M5Devs/Total-Connect/internal/core"
	"github.com/M5Devs/Total-Connect/internal/models"
	"github.com/M5Devs/Total-Connect/web"
)

// Server handles HTTP requests for the web UI and REST API.
type Server struct {
	engine core.StorageEngine
	mux    *http.ServeMux
}

// NewServer creates a new Server instance using the given StorageEngine.
func NewServer(engine core.StorageEngine) *Server {
	s := &Server{
		engine: engine,
		mux:    http.NewServeMux(),
	}
	s.routes()
	return s
}

// ServeHTTP implements http.Handler.
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Enable CORS for remote/LAN or tunnel clients
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	s.mux.ServeHTTP(w, r)
}

func (s *Server) routes() {
	// Static web assets
	fileServer := http.FileServer(http.FS(web.Assets))
	s.mux.Handle("GET /", fileServer)

	// API routes
	s.mux.HandleFunc("GET /api/remotes", s.handleListRemotes)
	s.mux.HandleFunc("POST /api/remotes/create", s.handleCreateRemote)
	s.mux.HandleFunc("POST /api/remotes/delete", s.handleDeleteRemote)
	s.mux.HandleFunc("GET /api/entries", s.handleListEntries)
	s.mux.HandleFunc("POST /api/copy", s.handleCopy)
	s.mux.HandleFunc("POST /api/move", s.handleMove)
	s.mux.HandleFunc("POST /api/mkdir", s.handleMkdir)
	s.mux.HandleFunc("POST /api/delete", s.handleDelete)
}

func renderJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func renderError(w http.ResponseWriter, status int, msg string) {
	renderJSON(w, status, map[string]string{"error": msg})
}

func (s *Server) handleListRemotes(w http.ResponseWriter, r *http.Request) {
	remotes, err := s.engine.ListRemotes(r.Context())
	if err != nil {
		renderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	renderJSON(w, http.StatusOK, map[string]interface{}{
		"remotes": remotes,
	})
}

type createRemoteRequest struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Parameters map[string]string `json:"parameters"`
}

func (s *Server) handleCreateRemote(w http.ResponseWriter, r *http.Request) {
	var req createRemoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(req.Type)

	if req.Name == "" {
		renderError(w, http.StatusBadRequest, "remote name is required")
		return
	}
	if req.Type == "" {
		renderError(w, http.StatusBadRequest, "remote type is required")
		return
	}

	if req.Parameters == nil {
		req.Parameters = make(map[string]string)
	}

	if err := s.engine.CreateRemote(r.Context(), req.Name, req.Type, req.Parameters); err != nil {
		renderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	renderJSON(w, http.StatusCreated, map[string]string{
		"status": "ok",
		"name":   req.Name,
	})
}

type deleteRemoteRequest struct {
	Name string `json:"name"`
}

func (s *Server) handleDeleteRemote(w http.ResponseWriter, r *http.Request) {
	var req deleteRemoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		renderError(w, http.StatusBadRequest, "remote name is required")
		return
	}

	if err := s.engine.DeleteRemote(r.Context(), req.Name); err != nil {
		renderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListEntries(w http.ResponseWriter, r *http.Request) {
	remote := r.URL.Query().Get("remote")
	path := r.URL.Query().Get("path")

	fullPath := buildFullPath(remote, path)

	entries, err := s.engine.ListEntries(r.Context(), fullPath)
	if err != nil {
		renderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// Ensure directory sorting (folders first, alphabetical case-insensitive)
	var dirs []models.FileItem
	var files []models.FileItem

	for _, item := range entries {
		if item.IsDir {
			dirs = append(dirs, item)
		} else {
			files = append(files, item)
		}
	}

	sort.SliceStable(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})
	sort.SliceStable(files, func(i, j int) bool {
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})

	sorted := append(dirs, files...)
	if sorted == nil {
		sorted = []models.FileItem{}
	}

	renderJSON(w, http.StatusOK, sorted)
}

type copyMoveRequest struct {
	SrcRemote string `json:"srcRemote"`
	SrcPath   string `json:"srcPath"`
	DstRemote string `json:"dstRemote"`
	DstPath   string `json:"dstPath"`
}

func (s *Server) handleCopy(w http.ResponseWriter, r *http.Request) {
	var req copyMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	src := buildFullPath(req.SrcRemote, req.SrcPath)
	dst := buildFullPath(req.DstRemote, req.DstPath)

	if err := s.engine.Copy(r.Context(), src, dst); err != nil {
		renderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleMove(w http.ResponseWriter, r *http.Request) {
	var req copyMoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	src := buildFullPath(req.SrcRemote, req.SrcPath)
	dst := buildFullPath(req.DstRemote, req.DstPath)

	if err := s.engine.Move(r.Context(), src, dst); err != nil {
		renderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

type pathRequest struct {
	Remote string `json:"remote"`
	Path   string `json:"path"`
}

func (s *Server) handleMkdir(w http.ResponseWriter, r *http.Request) {
	var req pathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fullPath := buildFullPath(req.Remote, req.Path)

	if err := s.engine.Mkdir(r.Context(), fullPath); err != nil {
		renderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDelete(w http.ResponseWriter, r *http.Request) {
	var req pathRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		renderError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	fullPath := buildFullPath(req.Remote, req.Path)

	if err := s.engine.Delete(r.Context(), fullPath); err != nil {
		renderError(w, http.StatusInternalServerError, err.Error())
		return
	}

	renderJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func buildFullPath(remote, path string) string {
	if remote == "" || remote == "local" {
		if path == "" {
			return "."
		}
		return path
	}

	// Remote storage (e.g. "myremote:")
	r := strings.TrimSuffix(remote, ":") + ":"
	if path == "" || path == "." {
		return r
	}
	return r + strings.TrimPrefix(path, "/")
}
