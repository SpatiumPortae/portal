package tools

import (
	"archive/tar"
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ZinoKader/portal/constants"
	"github.com/klauspost/pgzip"
)

func ReadFiles(fileNames []string) ([]*os.File, error) {
	var files []*os.File
	for _, fileName := range fileNames {
		f, err := os.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("file '%s' not found", fileName)
		}
		files = append(files, f)
	}
	return files, nil
}

// ArchiveAndCompressFiles tars and gzip-compresses files into a temporary file, returning it
// along with the resulting size
func ArchiveAndCompressFiles(files []*os.File) (*os.File, int64, error) {
	// chained writers -> writing to tw writes to gw -> writes to temporary file
	tempFile, err := os.CreateTemp(os.TempDir(), constants.SEND_TEMP_FILE_NAME_PREFIX)
	if err != nil {
		return nil, 0, err
	}
	tempFileWriter := bufio.NewWriter(tempFile)
	gw := pgzip.NewWriter(tempFileWriter)
	tw := tar.NewWriter(gw)

	for _, file := range files {
		err := addToTarArchive(tw, file)
		if err != nil {
			return nil, 0, err
		}
	}
	tw.Close()
	gw.Close()
	tempFileWriter.Flush()
	fileInfo, err := tempFile.Stat()
	if err != nil {
		return nil, 0, err
	}

	tempFile.Seek(0, io.SeekStart)
	return tempFile, fileInfo.Size(), nil
}

// DecompressAndUnarchiveBytes gzip-decompresses and un-tars files into the current working directory
// and returns the names and decompressed size of the created files
func DecompressAndUnarchiveBytes(reader io.Reader) ([]string, int64, error) {
	// chained readers -> gr reads from reader -> tr reads from gr
	gr, err := pgzip.NewReader(reader)
	if err != nil {
		return nil, 0, err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)

	var createdFiles []string
	var decompressedSize int64
	for {
		header, err := tr.Next()

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		if header == nil {
			continue
		}

		cwd, err := os.Getwd()
		if err != nil {
			return nil, 0, err
		}

		fileTarget := filepath.Join(cwd, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(fileTarget); err != nil {
				if err := os.MkdirAll(fileTarget, 0755); err != nil {
					return nil, 0, err
				}
			}
		case tar.TypeReg:
			f, err := os.OpenFile(fileTarget, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return nil, 0, err
			}
			if _, err := io.Copy(f, tr); err != nil {
				return nil, 0, err
			}
			fileInfo, err := f.Stat()
			if err != nil {
				return nil, 0, err
			}
			decompressedSize += fileInfo.Size()
			createdFiles = append(createdFiles, header.Name)
			f.Close()
		}
	}

	return createdFiles, decompressedSize, nil
}

// Traverses files and directories (recursively) for total size in bytes
func FilesTotalSize(files []*os.File) (int64, error) {
	var size int64
	for _, file := range files {
		err := filepath.Walk(file.Name(), func(_ string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				size += info.Size()
			}
			return err
		})
		if err != nil {
			return 0, err
		}
	}
	return size, nil
}

// Credits to: https://gist.github.com/mimoo/25fc9716e0f1353791f5908f94d6e726
func addToTarArchive(tw *tar.Writer, file *os.File) error {
	return filepath.Walk(file.Name(), func(file string, fi os.FileInfo, err error) error {
		header, e := tar.FileInfoHeader(fi, file)
		if e != nil {
			return err
		}
		header.Name = filepath.ToSlash(file)

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !fi.IsDir() {
			data, err := os.Open(file)
			if err != nil {
				return err
			}
			defer data.Close()
			if _, err := io.Copy(tw, data); err != nil {
				return err
			}
		}
		return nil
	})
}

// optimistically remove files created by portal with the specified prefix
func RemoveTemporaryFiles(prefix string) {
	tempFiles, err := os.ReadDir(os.TempDir())
	if err != nil {
		return
	}
	for _, tempFile := range tempFiles {
		fileInfo, err := tempFile.Info()
		if err != nil {
			continue
		}
		fileName := fileInfo.Name()
		if strings.HasPrefix(fileName, prefix) {
			os.Remove(filepath.Join(os.TempDir(), fileName))
		}
	}
}

// Credits to (legendary Mr. Nilsson): https://yourbasic.org/golang/formatting-byte-size-to-human-readable-format/
func ByteCountSI(b int64) string {
	const unit = 1000
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB",
		float64(b)/float64(div), "kMGTPE"[exp])
}
