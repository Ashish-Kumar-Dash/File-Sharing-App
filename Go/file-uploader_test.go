package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

const (
	headerContentType = "Content-Type"
	testRoom          = "ABCD23"
)

type storedObject struct {
	data []byte
	ct   string
}

type mockStorage struct {
	mu      sync.RWMutex
	buckets map[string]map[string]storedObject
}

func newMockStorage() *mockStorage {
	return &mockStorage{buckets: make(map[string]map[string]storedObject)}
}

func (m *mockStorage) bucketExists(bucket string) bool { _, ok := m.buckets[bucket]; return ok }

func (m *mockStorage) createBucket(bucket string) {
	if !m.bucketExists(bucket) {
		m.buckets[bucket] = make(map[string]storedObject)
	}
}

func (m *mockStorage) putObject(bucket, object string, data []byte, ct string) {
	m.buckets[bucket][object] = storedObject{data: data, ct: ct}
}

func (m *mockStorage) getObject(bucket, object string) (storedObject, bool) {
	o, ok := m.buckets[bucket][object]
	return o, ok
}

func (m *mockStorage) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/")
	parts := strings.Split(path, "/")
	if len(parts) == 0 || parts[0] == "" {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	bucket := parts[0]
	object := ""
	if len(parts) > 1 {
		object = strings.Join(parts[1:], "/")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	switch r.Method {
	case http.MethodHead:
		if m.bucketExists(bucket) {
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case http.MethodPut:
		if object == "" {
			m.createBucket(bucket)
			w.WriteHeader(http.StatusOK)
		} else if m.bucketExists(bucket) {
			body, _ := io.ReadAll(r.Body)
			m.putObject(bucket, object, body, r.Header.Get(headerContentType))
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	case http.MethodGet:
		if object == "" && r.URL.Query().Get("list-type") == "2" {
			if !m.bucketExists(bucket) {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			prefix := r.URL.Query().Get("prefix")
			var contents strings.Builder
			for key := range m.buckets[bucket] {
				if strings.HasPrefix(key, prefix) {
					obj := m.buckets[bucket][key]
					contents.WriteString(fmt.Sprintf(
						"<Contents><Key>%s</Key><LastModified>%s</LastModified><Size>%d</Size><ETag>\"x\"</ETag><StorageClass>STANDARD</StorageClass></Contents>",
						key, time.Now().UTC().Format(time.RFC3339), len(obj.data),
					))
				}
			}
			w.Header().Set("Content-Type", "application/xml")
			fmt.Fprintf(w, `<?xml version="1.0" encoding="UTF-8"?><ListBucketResult xmlns="http://s3.amazonaws.com/doc/2006-03-01/"><Name>%s</Name><Prefix>%s</Prefix><IsTruncated>false</IsTruncated><MaxKeys>1000</MaxKeys>%s</ListBucketResult>`,
				bucket, prefix, contents.String())
			return
		}
		if object == "" && r.URL.RawQuery == "location=" {
			if m.bucketExists(bucket) {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte("<LocationConstraint>us-east-1</LocationConstraint>"))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
			return
		}
		if !m.bucketExists(bucket) {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		obj, ok := m.getObject(bucket, object)
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if obj.ct != "" {
			w.Header().Set(headerContentType, obj.ct)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(obj.data)
	case http.MethodDelete:
		if object != "" && m.bucketExists(bucket) {
			delete(m.buckets[bucket], object)
			w.WriteHeader(http.StatusNoContent)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func createMultipartBody(t *testing.T, fieldName, filename string, content []byte) (string, *bytes.Buffer) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile(fieldName, filename)
	if err != nil {
		t.Fatalf("CreateFormFile error: %v", err)
	}
	if _, err := fw.Write(content); err != nil {
		t.Fatalf("writing file content failed: %v", err)
	}
	_ = w.Close()
	return w.FormDataContentType(), &b
}

func setupMock(t *testing.T) (*mockStorage, *httptest.Server) {
	storage := newMockStorage()
	storage.createBucket(defaultBucketName)
	server := httptest.NewServer(storage)
	t.Cleanup(server.Close)
	t.Setenv("LOCAL_IP", strings.TrimPrefix(server.URL, "http://"))
	t.Setenv("ACCESS_KEY", "dummy")
	t.Setenv("SECRET_KEY", "dummy")
	return storage, server
}

// --- Room ---

func TestCreateRoom(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/rooms", nil)
	w := httptest.NewRecorder()
	createRoom(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 got %d", res.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	code := body["room"]
	if !validateRoomCode(code) {
		t.Fatalf("invalid room code: %q", code)
	}
}

func TestValidateRoomCode(t *testing.T) {
	tests := []struct {
		code string
		want bool
	}{
		{"ABCD23", true},
		{"A2B3C4", true},
		{"ZZZZZZ", true},
		{"222222", true},
		{"abcd23", false},
		{"ABCDO3", false},
		{"ABCDI3", false},
		{"ABCDL3", false},
		{"ABCD03", false},
		{"ABCD13", false},
		{"ABC", false},
		{"ABCDEFG", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := validateRoomCode(tt.code); got != tt.want {
			t.Errorf("validateRoomCode(%q) = %v, want %v", tt.code, got, tt.want)
		}
	}
}

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"test.txt", false},
		{"photo.jpg", false},
		{"", true},
		{"../etc/passwd", true},
		{"foo/bar.txt", true},
		{"foo\\bar.txt", true},
		{"..hidden", true},
	}
	for _, tt := range tests {
		err := validateFilename(tt.name)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateFilename(%q) error=%v, wantErr=%v", tt.name, err, tt.wantErr)
		}
	}
}

// --- Upload ---

func TestUploadFileSuccess(t *testing.T) {
	storage, _ := setupMock(t)

	contentType, body := createMultipartBody(t, "file", "test.txt", []byte("hello world"))
	req := httptest.NewRequest(http.MethodPost, "/upload?room="+testRoom, body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()

	uploadFile(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200 got %d body=%s", res.StatusCode, string(b))
	}

	storage.mu.RLock()
	_, ok := storage.buckets[defaultBucketName][objectKey(testRoom, "test.txt")]
	storage.mu.RUnlock()
	if !ok {
		t.Fatal("file not stored at room-scoped key")
	}
}

func TestUploadFileMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/upload?room="+testRoom, nil)
	w := httptest.NewRecorder()
	uploadFile(w, req)
	if w.Result().StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 got %d", w.Result().StatusCode)
	}
}

func TestUploadInvalidRoom(t *testing.T) {
	contentType, body := createMultipartBody(t, "file", "test.txt", []byte("data"))
	req := httptest.NewRequest(http.MethodPost, "/upload?room=bad", body)
	req.Header.Set("Content-Type", contentType)
	w := httptest.NewRecorder()
	uploadFile(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Result().StatusCode)
	}
}

// --- Download ---

func TestDownloadFileSuccess(t *testing.T) {
	storage, _ := setupMock(t)

	key := objectKey(testRoom, "xyz.dat")
	storage.mu.Lock()
	storage.putObject(defaultBucketName, key, []byte("dummy"), "application/octet-stream")
	storage.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/download?room="+testRoom+"&filename=xyz.dat", nil)
	w := httptest.NewRecorder()
	downloadFile(w, req)
	res := w.Result()
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200 got %d body=%s", res.StatusCode, string(b))
	}
	body, _ := io.ReadAll(res.Body)
	if !strings.Contains(string(body), "xyz.dat") {
		t.Fatalf("presigned URL missing filename: %s", string(body))
	}
}

func TestDownloadMissingParams(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/download", nil)
	w := httptest.NewRecorder()
	downloadFile(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Result().StatusCode)
	}
}

// --- File listing ---

func TestListFiles(t *testing.T) {
	storage, _ := setupMock(t)

	storage.mu.Lock()
	storage.putObject(defaultBucketName, objectKey(testRoom, "a.txt"), []byte("aaa"), "text/plain")
	storage.putObject(defaultBucketName, objectKey(testRoom, "b.png"), []byte("bbb"), "image/png")
	storage.putObject(defaultBucketName, objectKey("XXXXXX", "c.txt"), []byte("ccc"), "text/plain")
	storage.mu.Unlock()

	req := httptest.NewRequest(http.MethodGet, "/files?room="+testRoom, nil)
	w := httptest.NewRecorder()
	filesHandler(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 200 got %d body=%s", res.StatusCode, string(b))
	}

	var files []fileInfo
	if err := json.NewDecoder(res.Body).Decode(&files); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %+v", len(files), files)
	}
	names := map[string]bool{}
	for _, f := range files {
		names[f.Name] = true
	}
	if !names["a.txt"] || !names["b.png"] {
		t.Fatalf("unexpected files: %+v", files)
	}
}

func TestListFilesEmpty(t *testing.T) {
	setupMock(t)

	req := httptest.NewRequest(http.MethodGet, "/files?room="+testRoom, nil)
	w := httptest.NewRecorder()
	filesHandler(w, req)
	res := w.Result()
	defer res.Body.Close()

	var files []fileInfo
	json.NewDecoder(res.Body).Decode(&files)
	if len(files) != 0 {
		t.Fatalf("expected empty list, got %d", len(files))
	}
}

// --- Delete ---

func TestDeleteFile(t *testing.T) {
	storage, _ := setupMock(t)

	key := objectKey(testRoom, "bye.txt")
	storage.mu.Lock()
	storage.putObject(defaultBucketName, key, []byte("data"), "text/plain")
	storage.mu.Unlock()

	req := httptest.NewRequest(http.MethodDelete, "/files?room="+testRoom+"&name=bye.txt", nil)
	w := httptest.NewRecorder()
	filesHandler(w, req)
	res := w.Result()
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(res.Body)
		t.Fatalf("expected 204 got %d body=%s", res.StatusCode, string(b))
	}

	storage.mu.RLock()
	_, exists := storage.buckets[defaultBucketName][key]
	storage.mu.RUnlock()
	if exists {
		t.Fatal("file should have been deleted")
	}
}

func TestDeletePathTraversal(t *testing.T) {
	setupMock(t)

	req := httptest.NewRequest(http.MethodDelete, "/files?room="+testRoom+"&name=../etc/passwd", nil)
	w := httptest.NewRecorder()
	filesHandler(w, req)
	if w.Result().StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 got %d", w.Result().StatusCode)
	}
}

// --- Validation ---

type mockMultipartFile struct {
	*bytes.Reader
}

func (m *mockMultipartFile) Close() error { return nil }

func TestValidateFileSize(t *testing.T) {
	maxFileSize := int64(1 << 30)
	tests := []struct {
		contentSize int64
		wantErr     bool
	}{
		{100, false},
		{maxFileSize - 1, false},
		{maxFileSize, false},
		{maxFileSize + 1, true},
	}
	for _, tt := range tests {
		file := &mockMultipartFile{bytes.NewReader(make([]byte, tt.contentSize))}
		err := validateFileSize(file, maxFileSize)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateFileSize(size=%d) error=%v, wantErr=%v", tt.contentSize, err, tt.wantErr)
		}
	}
}

func TestValidateFileExtension(t *testing.T) {
	tests := []struct {
		ext     string
		wantErr bool
	}{
		{".png", false}, {".zip", false}, {".xlsx", true}, {".docx", false},
		{".doc", false}, {".xls", true}, {".ppt", true}, {".mp4", true},
		{".pdf", false}, {".jpeg", false}, {".jpg", false}, {".txt", false},
		{".gif", false}, {".dat", false},
	}
	for _, tt := range tests {
		err := validateFileExtension("testfile" + tt.ext)
		if (err != nil) != tt.wantErr {
			t.Errorf("validateFileExtension(%s) error=%v, wantErr=%v", tt.ext, err, tt.wantErr)
		}
	}
}
