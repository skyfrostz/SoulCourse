package handler

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"hash/crc32"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"subject-choice-forum/backend/internal/config"
	"subject-choice-forum/backend/internal/http/middleware"

	"github.com/gin-gonic/gin"
)

func TestAdminUploadImageReturnsDimensions(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	uploadDir := filepath.Join(t.TempDir(), "uploads")
	router := gin.New()
	router.Use(middleware.RequestID())
	adminHandler := NewAdminHandler(config.Config{MediaUploadDir: uploadDir}, nil, nil, nil)
	router.POST("/api/v1/admin/uploads/images", adminHandler.UploadImage)

	body, contentType := multipartImageBody(t, "banner.png", testPNG(t))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/uploads/images", body)
	request.Header.Set("Content-Type", contentType)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("upload status = %d body=%s", recorder.Code, recorder.Body.String())
	}
	var response struct {
		Data struct {
			URL         string `json:"url"`
			ContentType string `json:"contentType"`
			Width       int    `json:"width"`
			Height      int    `json:"height"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Data.ContentType != "image/png" || response.Data.Width != 2 || response.Data.Height != 2 || response.Data.URL == "" {
		t.Fatalf("unexpected upload response: %+v", response.Data)
	}
}

func TestAdminUploadImageRejectsOversizedDimensions(t *testing.T) {
	t.Parallel()
	gin.SetMode(gin.TestMode)

	uploadDir := filepath.Join(t.TempDir(), "uploads")
	router := gin.New()
	router.Use(middleware.RequestID())
	adminHandler := NewAdminHandler(config.Config{MediaUploadDir: uploadDir}, nil, nil, nil)
	router.POST("/api/v1/admin/uploads/images", adminHandler.UploadImage)

	body, contentType := multipartImageBody(t, "huge.png", minimalPNG(12001, 1))
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/admin/uploads/images", body)
	request.Header.Set("Content-Type", contentType)
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("upload status = %d, want %d body=%s", recorder.Code, http.StatusBadRequest, recorder.Body.String())
	}
	if !bytes.Contains(recorder.Body.Bytes(), []byte("image_dimensions_too_large")) {
		t.Fatalf("response missing dimension error: %s", recorder.Body.String())
	}
}

func TestAdminUploadImageErrorBranches(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("missing file", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.RequestID())
		router.POST("/upload", NewAdminHandler(config.Config{MediaUploadDir: t.TempDir()}, nil, nil, nil).UploadImage)
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/upload", bytes.NewBufferString("not-multipart"))
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "missing_file") {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("unsupported file", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.RequestID())
		router.POST("/upload", NewAdminHandler(config.Config{MediaUploadDir: t.TempDir()}, nil, nil, nil).UploadImage)
		body, contentType := multipartImageBody(t, "payload.txt", []byte("not an image"))
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/upload", body)
		request.Header.Set("Content-Type", contentType)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusBadRequest || !strings.Contains(recorder.Body.String(), "unsupported_file_type") {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})

	t.Run("upload directory unavailable", func(t *testing.T) {
		root := t.TempDir()
		blocked := filepath.Join(root, "blocked")
		if err := os.WriteFile(blocked, []byte("file"), 0600); err != nil {
			t.Fatal(err)
		}
		router := gin.New()
		router.Use(middleware.RequestID())
		router.POST("/upload", NewAdminHandler(config.Config{MediaUploadDir: blocked}, nil, nil, nil).UploadImage)
		body, contentType := multipartImageBody(t, "image.png", testPNG(t))
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, "/upload", body)
		request.Header.Set("Content-Type", contentType)
		router.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusInternalServerError || !strings.Contains(recorder.Body.String(), "upload_dir_failed") {
			t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
		}
	})
}

func multipartImageBody(t *testing.T, fileName string, imageBytes []byte) (*bytes.Buffer, string) {
	t.Helper()
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(imageBytes); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return body, writer.FormDataContentType()
}

func minimalPNG(width int, height int) []byte {
	output := bytes.NewBuffer([]byte{137, 80, 78, 71, 13, 10, 26, 10})
	writePNGChunk(output, "IHDR", func() []byte {
		data := make([]byte, 13)
		binary.BigEndian.PutUint32(data[0:4], uint32(width))
		binary.BigEndian.PutUint32(data[4:8], uint32(height))
		data[8] = 8
		data[9] = 2
		return data
	}())
	writePNGChunk(output, "IEND", nil)
	return output.Bytes()
}

func writePNGChunk(output *bytes.Buffer, chunkType string, data []byte) {
	_ = binary.Write(output, binary.BigEndian, uint32(len(data)))
	output.WriteString(chunkType)
	output.Write(data)
	checksum := crc32.NewIEEE()
	checksum.Write([]byte(chunkType))
	checksum.Write(data)
	_ = binary.Write(output, binary.BigEndian, checksum.Sum32())
}
