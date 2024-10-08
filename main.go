package main

import (
	"archive/zip"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

func unzipCBZ(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return fmt.Errorf("failed to open zip file %s: %w", src, err)
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)
		if f.FileInfo().IsDir() {
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(fpath), err)
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", fpath, err)
		}

		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in zip %s: %w", f.Name, err)
		}

		_, err = io.Copy(outFile, rc)
		outFile.Close()
		rc.Close()

		if err != nil {
			return fmt.Errorf("failed to copy file %s: %w", fpath, err)
		}
	}
	return nil
}

func renameFilesWithLeadingZeros(dir string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		numberRegex := regexp.MustCompile(`(\d+)`)
		imageExtensions := map[string]bool{
			".jpg": true, ".jpeg": true, ".png": true,
			".gif": true, ".bmp": true, ".tiff": true,
		}

		ext := filepath.Ext(info.Name())
		if !imageExtensions[ext] {
			return nil
		}

		oldName := info.Name()
		newName := numberRegex.ReplaceAllStringFunc(oldName, func(match string) string {
			num, err := strconv.Atoi(match)
			if err != nil {
				return match
			}
			return fmt.Sprintf("%03d", num)
		})

		if oldName != newName {
			oldPath := path
			newPath := filepath.Join(filepath.Dir(path), newName)
			if err := os.Rename(oldPath, newPath); err != nil {
				return err
			}
			fmt.Printf("Renamed: %s -> %s\n", oldPath, newPath)
		}

		return nil
	})

	return err
}

func zipFiles(filename string, baseDir string) error {
	newZipFile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer newZipFile.Close()

	zipWriter := zip.NewWriter(newZipFile)
	defer zipWriter.Close()

	err = filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		relPath, err := filepath.Rel(baseDir, path)
		if err != nil {
			return err
		}

		fileToZip, err := os.Open(path)
		if err != nil {
			return err
		}
		defer fileToZip.Close()

		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		header.Name = relPath
		header.Method = zip.Deflate

		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = io.Copy(writer, fileToZip)
		return err
	})

	return err
}

func extractAndRenameArchive(archivePath, extractDir string) error {
	if strings.HasSuffix(strings.ToLower(archivePath), ".cbr") {
		return extractAndRenameCBR(archivePath, extractDir)
	} else {
		return extractAndRenameCBZ(archivePath, extractDir)
	}
}

func extractAndRenameCBR(cbrPath, extractDir string) error {
	tempDir, err := os.MkdirTemp("", "cbr_extract")
	if err != nil {
		return fmt.Errorf("failed to create temporary directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	cmd := exec.Command("unrar", "x", cbrPath, tempDir)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to extract CBR file: %w", err)
	}

	if err := renameFilesWithLeadingZeros(tempDir); err != nil {
		return fmt.Errorf("failed to rename files: %w", err)
	}

	if err := os.MkdirAll(extractDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	if err := copyDir(tempDir, extractDir); err != nil {
		return fmt.Errorf("failed to copy renamed files: %w", err)
	}

	return nil
}

func extractAndRenameCBZ(cbzPath, extractDir string) error {
	if err := unzipCBZ(cbzPath, extractDir); err != nil {
		return fmt.Errorf("failed to extract CBZ file: %w", err)
	}

	if err := renameFilesWithLeadingZeros(extractDir); err != nil {
		return fmt.Errorf("failed to rename files: %w", err)
	}

	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(dstPath, info.Mode())
		}

		return copyFile(path, dstPath)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func main() {
	dir := "./"

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("failed to read directory %s: %v", dir, err)
	}

	var wg sync.WaitGroup

	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == ".cbz" || ext == ".cbr" {
			wg.Add(1)
			go func(file os.DirEntry) {
				defer wg.Done()

				filePath := filepath.Join(dir, file.Name())
				extractDir := filepath.Join(dir, strings.TrimSuffix(file.Name(), ext)+"_extracted")
				newCBZPath := filepath.Join(dir, strings.TrimSuffix(file.Name(), ext)+".cbz")

				fmt.Printf("Processing %s...\n", filePath)

				// Extract and rename for both CBR and CBZ
				if err := extractAndRenameArchive(filePath, extractDir); err != nil {
					fmt.Printf("failed to extract and rename %s: %v\n", filePath, err)
					return
				}

				if ext == ".cbr" {
					fmt.Printf("Compressing files into %s...\n", newCBZPath)
					if err := zipFiles(newCBZPath, extractDir); err != nil {
						fmt.Printf("failed to zip files into %s: %v\n", newCBZPath, err)
						return
					}

					fmt.Printf("Removing original CBR file...\n")
					if err := os.Remove(filePath); err != nil {
						fmt.Printf("failed to remove original CBR file %s: %v\n", filePath, err)
					}

					fmt.Printf("Successfully converted %s to %s\n", filePath, newCBZPath)
				} else {
					// For CBZ, we just need to update the original file
					fmt.Printf("Updating original CBZ file %s...\n", filePath)
					if err := os.Remove(filePath); err != nil {
						fmt.Printf("failed to remove original CBZ file %s: %v\n", filePath, err)
						return
					}
					if err := zipFiles(filePath, extractDir); err != nil {
						fmt.Printf("failed to update CBZ file %s: %v\n", filePath, err)
						return
					}
					fmt.Printf("Successfully updated %s\n", filePath)
				}

				fmt.Printf("Cleaning up temporary files...\n")
				os.RemoveAll(extractDir)
			}(file)
		}
	}

	wg.Wait()
}
