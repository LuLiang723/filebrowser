package fbhttp

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/mholt/archives"
	"github.com/filebrowser/filebrowser/v2/fileutils"
)

type archiveRequest struct {
	Items  []string `json:"items"`
	Format string   `json:"format"`
}

func withPermArchive(fn handleFunc) handleFunc {
	return withUser(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
		if !d.user.Perm.Create {
			return http.StatusForbidden, nil
		}
		return fn(w, r, d)
	})
}

func getAlgorithm(format string) (archives.Archival, error) {
	switch format {
	case "zip":
		return archives.Zip{}, nil
	case "tar":
		return archives.Tar{}, nil
	case "targz":
		return archives.CompressedArchive{Compression: archives.Gz{}, Archival: archives.Tar{}}, nil
	case "tarbz2":
		return archives.CompressedArchive{Compression: archives.Bz2{}, Archival: archives.Tar{}}, nil
	case "tarxz":
		return archives.CompressedArchive{Compression: archives.Xz{}, Archival: archives.Tar{}}, nil
	case "tarlz4":
		return archives.CompressedArchive{Compression: archives.Lz4{}, Archival: archives.Tar{}}, nil
	case "tarsz":
		return archives.CompressedArchive{Compression: archives.Sz{}, Archival: archives.Tar{}}, nil
	case "tarbr":
		return archives.CompressedArchive{Compression: archives.Brotli{}, Archival: archives.Tar{}}, nil
	case "tarzst":
		return archives.CompressedArchive{Compression: archives.Zstd{}, Archival: archives.Tar{}}, nil
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

func collectFiles(d *data, path, commonPath string) ([]archives.FileInfo, error) {
	if !d.Check(path) {
		return nil, nil
	}

	info, err := d.user.Fs.Stat(path)
	if err != nil {
		return nil, err
	}

	var archiveFiles []archives.FileInfo

	if path != commonPath {
		nameInArchive := strings.TrimPrefix(path, commonPath)
		nameInArchive = strings.TrimPrefix(nameInArchive, string(filepath.Separator))
		nameInArchive = filepath.ToSlash(nameInArchive)

		archiveFiles = append(archiveFiles, archives.FileInfo{
			FileInfo:      info,
			NameInArchive: nameInArchive,
			Open: func() (fs.File, error) {
				return d.user.Fs.Open(path)
			},
		})
	}

	if info.IsDir() {
		f, err := d.user.Fs.Open(path)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		names, err := f.Readdirnames(0)
		if err != nil {
			return nil, err
		}

		for _, name := range names {
			fPath := filepath.Join(path, name)
			subFiles, err := collectFiles(d, fPath, commonPath)
			if err != nil {
				continue
			}
			archiveFiles = append(archiveFiles, subFiles...)
		}
	}

	return archiveFiles, nil
}

var archiveHandler = withPermArchive(func(w http.ResponseWriter, r *http.Request, d *data) (int, error) {
	var req archiveRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			return http.StatusBadRequest, err
		}
		defer r.Body.Close()
	}

	archiver, err := getAlgorithm(req.Format)
	if err != nil {
		return http.StatusBadRequest, err
	}

	dstPath := filepath.Clean(r.URL.Path)

	var cleanedItems []string
	for _, item := range req.Items {
		cleanItem := filepath.Clean(strings.TrimPrefix(strings.TrimPrefix(item, "/files"), "/api/resources"))
		cleanedItems = append(cleanedItems, cleanItem)
	}
	commonDir := fileutils.CommonPrefix(filepath.Separator, cleanedItems...)

	var allFiles []archives.FileInfo
	for _, fname := range cleanedItems {
		files, err := collectFiles(d, fname, commonDir)
		if err != nil {
			continue
		}
		allFiles = append(allFiles, files...)
	}

	out, err := d.user.Fs.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	if err != nil {
		return http.StatusInternalServerError, err
	}

	archiveErr := archiver.Archive(r.Context(), out, allFiles)
	out.Close()

	if archiveErr != nil {
		log.Printf("%v", archiveErr)
		d.user.Fs.Remove(dstPath)
		return http.StatusInternalServerError, archiveErr
	}

	return http.StatusOK, nil
})