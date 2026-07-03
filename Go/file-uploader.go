package main

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/lifecycle"
)

const defaultBucketName = "default-bucket"

func main() {
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
	}

	if err := ensureBucket(); err != nil {
		fmt.Printf("Warning: bucket setup: %v\n", err)
	}

	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "OK")
	})
	http.HandleFunc("/rooms", createRoom)
	http.HandleFunc("/upload", uploadFile)
	http.HandleFunc("/download", downloadFile)
	http.HandleFunc("/files", filesHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8000"
	}
	if port[0] != ':' {
		port = ":" + port
	}

	fmt.Printf("Server starting on %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Server failed to start: %v\n", err)
		os.Exit(1)
	}
}

func newMinioClient() (*minio.Client, error) {
	endpoint := os.Getenv("LOCAL_IP")
	accessKeyID := os.Getenv("ACCESS_KEY")
	secretAccessKey := os.Getenv("SECRET_KEY")
	if endpoint == "" || accessKeyID == "" || secretAccessKey == "" {
		return nil, fmt.Errorf("missing environment variables")
	}
	return minio.New(endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure:       false,
		BucketLookup: minio.BucketLookupPath,
	})
}

func setCORS(w http.ResponseWriter, methods string) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", methods)
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func objectKey(room, filename string) string {
	return "rooms/" + room + "/" + filename
}

// ponytail: modulo bias negligible for room codes (31 chars, 256%31=8)
const roomChars = "ABCDEFGHJKMNPQRSTUVWXYZ23456789"

func generateRoomCode() (string, error) {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", err
	}
	for i, b := range buf {
		buf[i] = roomChars[int(b)%len(roomChars)]
	}
	return string(buf[:]), nil
}

func validateRoomCode(code string) bool {
	if len(code) != 6 {
		return false
	}
	for _, c := range code {
		switch {
		case c >= 'A' && c <= 'Z' && c != 'I' && c != 'L' && c != 'O':
		case c >= '2' && c <= '9':
		default:
			return false
		}
	}
	return true
}

func validateFilename(name string) error {
	if name == "" {
		return fmt.Errorf("filename required")
	}
	if strings.Contains(name, "..") || strings.Contains(name, "/") || strings.Contains(name, "\\") {
		return fmt.Errorf("invalid filename")
	}
	return nil
}

func ensureBucket() error {
	client, err := newMinioClient()
	if err != nil {
		return err
	}
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, defaultBucketName)
	if err != nil {
		return err
	}
	if !exists {
		if err := client.MakeBucket(ctx, defaultBucketName, minio.MakeBucketOptions{}); err != nil {
			return err
		}
	}
	lc := &lifecycle.Configuration{
		Rules: []lifecycle.Rule{{
			ID:         "expire-rooms",
			Status:     "Enabled",
			RuleFilter: lifecycle.Filter{Prefix: "rooms/"},
			Expiration: lifecycle.Expiration{Days: lifecycle.ExpirationDays(1)},
		}},
	}
	return client.SetBucketLifecycle(ctx, defaultBucketName, lc)
}

func createRoom(w http.ResponseWriter, r *http.Request) {
	setCORS(w, "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	code, err := generateRoomCode()
	if err != nil {
		http.Error(w, "Failed to generate room code", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"room": code})
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	setCORS(w, "POST, OPTIONS")
	if r.Method == http.MethodOptions {
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	room := r.URL.Query().Get("room")
	if !validateRoomCode(room) {
		http.Error(w, "Invalid room code", http.StatusBadRequest)
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse form: %v", err), http.StatusBadRequest)
		return
	}

	file, h, err := r.FormFile("file")
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get file: %v", err), http.StatusBadRequest)
		return
	}
	defer file.Close()

	if err := validateFilename(h.Filename); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	const maxFileSize = int64(1 << 30)
	if err := validateFileSize(file, maxFileSize); err != nil {
		http.Error(w, fmt.Sprintf("File size exceeds limit of %d bytes", maxFileSize), http.StatusBadRequest)
		return
	}

	if err := validateFileExtension(h.Filename); err != nil {
		http.Error(w, fmt.Sprintf("File %s not allowed", h.Filename), http.StatusBadRequest)
		return
	}

	minioClient, err := newMinioClient()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create MinIO client: %v", err), http.StatusInternalServerError)
		return
	}

	key := objectKey(room, h.Filename)
	if _, err := minioClient.PutObject(context.Background(), defaultBucketName, key, file, h.Size, minio.PutObjectOptions{
		ContentType: "application/octet-stream",
	}); err != nil {
		http.Error(w, fmt.Sprintf("Failed to upload: %v", err), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func downloadFile(w http.ResponseWriter, r *http.Request) {
	setCORS(w, "GET, OPTIONS")
	if r.Method == http.MethodOptions {
		return
	}

	room := r.URL.Query().Get("room")
	if !validateRoomCode(room) {
		http.Error(w, "Invalid room code", http.StatusBadRequest)
		return
	}

	filename := r.URL.Query().Get("filename")
	if err := validateFilename(filename); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	minioClient, err := newMinioClient()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create MinIO client: %v", err), http.StatusInternalServerError)
		return
	}

	key := objectKey(room, filename)
	reqParams := make(url.Values)
	reqParams.Set("response-content-disposition", fmt.Sprintf("attachment; filename=%q", filename))
	presignedURL, err := minioClient.PresignedGetObject(context.Background(), defaultBucketName, key, 24*time.Hour, reqParams)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate presigned URL: %v", err), http.StatusInternalServerError)
		return
	}

	io.WriteString(w, presignedURL.String())
}

type fileInfo struct {
	Name        string    `json:"name"`
	Size        int64     `json:"size"`
	ContentType string    `json:"contentType"`
	UploadedAt  time.Time `json:"uploadedAt"`
}

func filesHandler(w http.ResponseWriter, r *http.Request) {
	setCORS(w, "GET, DELETE, OPTIONS")
	if r.Method == http.MethodOptions {
		return
	}

	room := r.URL.Query().Get("room")
	if !validateRoomCode(room) {
		http.Error(w, "Invalid room code", http.StatusBadRequest)
		return
	}

	switch r.Method {
	case http.MethodGet:
		listFiles(w, room)
	case http.MethodDelete:
		deleteFile(w, r, room)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func listFiles(w http.ResponseWriter, room string) {
	client, err := newMinioClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	prefix := "rooms/" + room + "/"
	var files []fileInfo
	for obj := range client.ListObjects(context.Background(), defaultBucketName, minio.ListObjectsOptions{Prefix: prefix}) {
		if obj.Err != nil {
			http.Error(w, obj.Err.Error(), http.StatusInternalServerError)
			return
		}
		name := strings.TrimPrefix(obj.Key, prefix)
		if name == "" {
			continue
		}
		files = append(files, fileInfo{
			Name:        name,
			Size:        obj.Size,
			ContentType: obj.ContentType,
			UploadedAt:  obj.LastModified,
		})
	}
	if files == nil {
		files = []fileInfo{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func deleteFile(w http.ResponseWriter, r *http.Request, room string) {
	name := r.URL.Query().Get("name")
	if err := validateFilename(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	client, err := newMinioClient()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	key := objectKey(room, name)
	if err := client.RemoveObject(context.Background(), defaultBucketName, key, minio.RemoveObjectOptions{}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func validateFileExtension(filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == "" {
		return fmt.Errorf("file extension not found")
	}
	allowedFileTypes := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true,
		".pdf": true, ".txt": true, ".doc": true, ".docx": true,
		".zip": true, ".dat": true,
	}
	if !allowedFileTypes[ext] {
		return fmt.Errorf("file: %s with type %s not allowed", filename, ext)
	}
	return nil
}

func validateFileSize(file multipart.File, maxFileSize int64) error {
	n, err := io.Copy(io.Discard, io.LimitReader(file, maxFileSize+1))
	if err != nil {
		return fmt.Errorf("error reading file: %v", err)
	}
	if n > maxFileSize {
		return fmt.Errorf("file size exceeds limit of %d bytes", maxFileSize)
	}
	file.Seek(0, io.SeekStart)
	return nil
}
