package media

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/DagMT/client-portal/internal/tenant"
	"github.com/DagMT/client-portal/internal/utils"
)

const (
	maxImageBytes = 5 << 20  // 5 MB
	maxVideoBytes = 10 << 20 // 10 MB
)

var imageMIME = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/webp":    true,
	"image/gif":     true,
	"image/svg+xml": true,
}

var videoMIME = map[string]bool{
	"video/mp4":       true,
	"video/webm":      true,
	"video/ogg":       true,
	"video/quicktime": true,
}

type Handler struct {
	httpClient *http.Client
}

func NewHandler(httpClient *http.Client) *Handler {
	return &Handler{httpClient: httpClient}
}

// InitBucket handles POST /api/media/init-bucket
//
// @Summary     Ensure the client's default storage bucket exists
// @Tags        media
// @Produce     json
// @Success     200 {object} map[string]string
// @Failure     401 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/media/init-bucket [post]
// @Security    CookieAuth
func (h *Handler) InitBucket(w http.ResponseWriter, r *http.Request) {
	cfg, ok := tenant.ConfigFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	bucketName := bucketNameFromSiteURL(cfg.SiteURL)
	slog.Info("init-bucket", "tenant", cfg.TenantID, "bucket", bucketName, "supabase_url", cfg.SupabaseURL)

	if err := h.createBucket(r.Context(), cfg.SupabaseURL, cfg.ServiceRoleKey, bucketName); err != nil {
		slog.Error("init-bucket failed", "tenant", cfg.TenantID, "bucket", bucketName, "error", err)
		utils.RespondError(w, http.StatusInternalServerError, "failed to initialise storage bucket")
		return
	}

	slog.Info("init-bucket ok", "tenant", cfg.TenantID, "bucket", bucketName)
	utils.RespondJSON(w, http.StatusOK, map[string]string{"bucket": bucketName})
}

// Upload handles POST /api/media/upload
//
// Accepts multipart/form-data with fields:
//   - file   — the binary file
//   - bucket — target bucket name
//   - path   — folder path within the bucket (optional, "" = root)
//
// @Summary     Upload a file to the client's Supabase Storage
// @Tags        media
// @Accept      mpfd
// @Produce     json
// @Success     200 {object} map[string]string
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/media/upload [post]
// @Security    CookieAuth
func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	cfg, ok := tenant.ConfigFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	if err := r.ParseMultipartForm(maxVideoBytes); err != nil {
		utils.RespondError(w, http.StatusBadRequest, "file too large or invalid form data")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		utils.RespondError(w, http.StatusBadRequest, "file field required")
		return
	}
	defer file.Close()

	mimeType := header.Header.Get("Content-Type")
	isVideo := videoMIME[mimeType]
	if !imageMIME[mimeType] && !isVideo {
		utils.RespondError(w, http.StatusBadRequest, "unsupported file type")
		return
	}
	maxBytes := int64(maxImageBytes)
	if isVideo {
		maxBytes = maxVideoBytes
	}
	if header.Size > maxBytes {
		limitMB := maxBytes >> 20
		utils.RespondError(w, http.StatusBadRequest, fmt.Sprintf("file exceeds %d MB limit", limitMB))
		return
	}

	bucket := strings.TrimSpace(r.FormValue("bucket"))
	if bucket == "" {
		bucket = bucketNameFromSiteURL(cfg.SiteURL)
	}
	folder := strings.Trim(r.FormValue("path"), "/")
	filename := filepath.Base(header.Filename)
	storagePath := filename
	if folder != "" {
		storagePath = folder + "/" + filename
	}

	publicURL, err := h.uploadToStorage(r.Context(), cfg.SupabaseURL, cfg.ServiceRoleKey, bucket, storagePath, mimeType, file, header.Size)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "upload failed")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]string{
		"url":  publicURL,
		"path": storagePath,
	})
}

// DeleteFile handles DELETE /api/media/file
//
// @Summary     Delete a file from the client's Supabase Storage
// @Tags        media
// @Accept      json
// @Produce     json
// @Param       body body object true "bucket and path"
// @Success     200 {object} map[string]bool
// @Failure     400 {object} utils.ErrorResponse
// @Failure     401 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/media/file [delete]
// @Security    CookieAuth
func (h *Handler) DeleteFile(w http.ResponseWriter, r *http.Request) {
	cfg, ok := tenant.ConfigFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req struct {
		Bucket string `json:"bucket"`
		Path   string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Path == "" {
		utils.RespondError(w, http.StatusBadRequest, "bucket and path required")
		return
	}
	if req.Bucket == "" {
		req.Bucket = bucketNameFromSiteURL(cfg.SiteURL)
	}

	if err := h.deleteFromStorage(r.Context(), cfg.SupabaseURL, cfg.ServiceRoleKey, req.Bucket, req.Path); err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "delete failed")
		return
	}

	utils.RespondJSON(w, http.StatusOK, map[string]bool{"deleted": true})
}

// ListFiles handles GET /api/media/files?bucket=X&path=Y
//
// @Summary     List files in a storage bucket path
// @Tags        media
// @Produce     json
// @Param       bucket query string false "bucket name"
// @Param       path   query string false "folder path"
// @Success     200 {array}  object
// @Failure     401 {object} utils.ErrorResponse
// @Failure     500 {object} utils.ErrorResponse
// @Router      /api/media/files [get]
// @Security    CookieAuth
func (h *Handler) ListFiles(w http.ResponseWriter, r *http.Request) {
	cfg, ok := tenant.ConfigFromContext(r.Context())
	if !ok {
		utils.RespondError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	bucket := strings.TrimSpace(r.URL.Query().Get("bucket"))
	if bucket == "" {
		bucket = bucketNameFromSiteURL(cfg.SiteURL)
	}
	prefix := strings.Trim(r.URL.Query().Get("path"), "/")

	body, _ := json.Marshal(map[string]any{
		"prefix":  prefix,
		"sortBy":  map[string]string{"column": "name", "order": "asc"},
		"limit":   1000,
		"offset":  0,
	})
	endpoint := cfg.SupabaseURL + "/storage/v1/object/list/" + bucket
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "list failed")
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+cfg.ServiceRoleKey)
	req.Header.Set("apikey", cfg.ServiceRoleKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		utils.RespondError(w, http.StatusInternalServerError, "list failed")
		return
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		slog.Error("list-files failed", "tenant", cfg.TenantID, "bucket", bucket, "error", string(respBody))
		utils.RespondError(w, http.StatusInternalServerError, "list failed")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(respBody)
}

// ── internal helpers ────────────────────────────────────────────────────────

func (h *Handler) createBucket(ctx context.Context, supabaseURL, serviceRoleKey, bucketName string) error {
	body, _ := json.Marshal(map[string]any{"id": bucketName, "name": bucketName, "public": true})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		supabaseURL+"/storage/v1/bucket", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("create bucket: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		return nil
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		// Supabase Storage returns HTTP 400 with statusCode "409" when the bucket already exists.
		if strings.Contains(string(body), `"409"`) || strings.Contains(string(body), "already exists") {
			return nil
		}
		return fmt.Errorf("create bucket returned %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (h *Handler) uploadToStorage(ctx context.Context, supabaseURL, serviceRoleKey, bucket, path, mimeType string, body io.Reader, size int64) (string, error) {
	endpoint := supabaseURL + "/storage/v1/object/" + bucket + "/" + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", mimeType)
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)
	req.Header.Set("x-upsert", "true")
	req.ContentLength = size

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("storage upload: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("storage upload returned %d", resp.StatusCode)
	}

	publicURL := supabaseURL + "/storage/v1/object/public/" + bucket + "/" + path
	return publicURL, nil
}

func (h *Handler) deleteFromStorage(ctx context.Context, supabaseURL, serviceRoleKey, bucket, path string) error {
	body, _ := json.Marshal(map[string]any{"prefixes": []string{path}})
	endpoint := supabaseURL + "/storage/v1/object/" + bucket
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+serviceRoleKey)
	req.Header.Set("apikey", serviceRoleKey)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("storage delete: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("storage delete returned %d", resp.StatusCode)
	}
	return nil
}

func bucketNameFromSiteURL(siteURL string) string {
	u, err := url.Parse(siteURL)
	if err != nil || u.Hostname() == "" {
		return "media"
	}
	name := strings.ToLower(u.Hostname())
	name = strings.ReplaceAll(name, ".", "-")
	var sb strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			sb.WriteRune(r)
		}
	}
	name = strings.Trim(sb.String(), "-")
	if len(name) > 63 {
		name = name[:63]
	}
	if len(name) < 3 {
		return "media"
	}
	return name
}
