package core

import (
  	"fmt"
  	"io"
  	"mime/multipart"
  	"net/http"
  	"os"
  	"path/filepath"
  )

// UploadedFile holds metadata about a successfully uploaded file.
type UploadedFile struct {
  	Filename    string // original filename from the client
  	ContentType string // MIME type
  	Size        int64  // bytes written
  }

// MaxUploadSize is the default maximum allowed upload size (10 MB).
// Override before calling UploadFile to change the limit.
var MaxUploadSize int64 = 10 << 20 // 10 MB

// UploadFile reads a single file from the multipart form field named key,
// writes it to destDir, and returns metadata about the saved file.
// The destination filename is sanitised to its base name to prevent path traversal.
//
// Example:
//
//	file, err := core.UploadFile(r, "avatar", "./uploads")
//	if err != nil {
//	    return nil, err // already a 400/413
//	}
//	return map[string]string{"filename": file.Filename}, nil
func UploadFile(r *http.Request, key, destDir string) (*UploadedFile, error) {
  	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
      		return nil, ErrBadRequest("request body too large or not multipart/form-data")
      	}

  	file, header, err := r.FormFile(key)
  	if err != nil {
      		return nil, ErrBadRequest(fmt.Sprintf("missing or invalid file field %q", key))
      	}
  	defer file.Close()

  	if err := os.MkdirAll(destDir, 0o755); err != nil {
      		return nil, fmt.Errorf("create upload dir: %w", err)
      	}

  	filename := filepath.Base(header.Filename) // sanitise
  	dest := filepath.Join(destDir, filename)

  	out, err := os.Create(dest)
  	if err != nil {
      		return nil, fmt.Errorf("create file %q: %w", dest, err)
      	}
  	defer out.Close()

  	n, err := io.Copy(out, io.LimitReader(file, MaxUploadSize))
  	if err != nil {
      		return nil, fmt.Errorf("write file: %w", err)
      	}

  	return &UploadedFile{
      		Filename:    filename,
      		ContentType: header.Header.Get("Content-Type"),
      		Size:        n,
      	}, nil
  }

// UploadFiles reads all files from the multipart form field named key
// and writes each to destDir. Returns metadata for all uploaded files.
func UploadFiles(r *http.Request, key, destDir string) ([]*UploadedFile, error) {
  	if err := r.ParseMultipartForm(MaxUploadSize); err != nil {
      		return nil, ErrBadRequest("request body too large or not multipart/form-data")
      	}

  	form := r.MultipartForm
  	if form == nil || form.File[key] == nil {
      		return nil, ErrBadRequest(fmt.Sprintf("missing file field %q", key))
      	}

  	var results []*UploadedFile
  	for _, fh := range form.File[key] {
      		uf, err := saveFormFile(fh, destDir)
      		if err != nil {
            			return results, err
            		}
      		results = append(results, uf)
      	}
  	return results, nil
  }

func saveFormFile(fh *multipart.FileHeader, destDir string) (*UploadedFile, error) {
  	f, err := fh.Open()
  	if err != nil {
      		return nil, fmt.Errorf("open uploaded file: %w", err)
      	}
  	defer f.Close()

  	if err := os.MkdirAll(destDir, 0o755); err != nil {
      		return nil, fmt.Errorf("create upload dir: %w", err)
      	}

  	filename := filepath.Base(fh.Filename)
  	dest := filepath.Join(destDir, filename)

  	out, err := os.Create(dest)
  	if err != nil {
      		return nil, fmt.Errorf("create file %q: %w", dest, err)
      	}
  	defer out.Close()

  	n, err := io.Copy(out, io.LimitReader(f, MaxUploadSize))
  	if err != nil {
      		return nil, fmt.Errorf("write file: %w", err)
      	}

  	return &UploadedFile{
      		Filename:    filename,
      		ContentType: fh.Header.Get("Content-Type"),
      		Size:        n,
      	}, nil
  }
