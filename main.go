package main

import (
	"archive/zip"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/fatih/color"
	"github.com/nwaples/rardecode/v2"
	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/schollz/progressbar/v3"
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
	ext := strings.ToLower(filepath.Ext(archivePath))
	switch ext {
	case ".cbr":
		return extractAndRenameCBR(archivePath, extractDir)
	case ".pdf":
		return extractAndRenamePDF(archivePath, extractDir)
	case ".cbz":
		return extractAndRenameCBZ(archivePath, extractDir)
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}
}

func extractAndRenameCBR(cbrPath, extractDir string) error {
	file, err := os.Open(cbrPath)
	if err != nil {
		return fmt.Errorf("failed to open CBR file %s: %w", cbrPath, err)
	}
	defer file.Close()

	r, err := rardecode.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create RAR reader for %s: %w", cbrPath, err)
	}

	if err := os.MkdirAll(extractDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read RAR entry: %w", err)
		}

		if header.IsDir {
			continue
		}

		cleanName := filepath.Clean(header.Name)
		relPath, err := filepath.Rel(extractDir, filepath.Join(extractDir, cleanName))
		if err != nil || strings.HasPrefix(relPath, "..") {
			log.Printf("skipping potentially unsafe file: %s", header.Name)
			continue
		}

		destPath := filepath.Join(extractDir, relPath)
		if err := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(destPath), err)
		}

		outFile, err := os.Create(destPath)
		defer outFile.Close()
		if err != nil {
			return fmt.Errorf("failed to create file %s: %w", destPath, err)
		}

		_, err = io.Copy(outFile, r)
		if err != nil {
			return fmt.Errorf("failed to extract file %s: %w", header.Name, err)
		}
	}

	if err := renameFilesWithLeadingZeros(extractDir); err != nil {
		return fmt.Errorf("failed to rename files: %w", err)
	}

	return nil
}

// Fallback to RAR extraction if .cbz file is actually a misnamed .cbr
func extractAndRenameCBZ(cbzPath, extractDir string) error {
	err := unzipCBZ(cbzPath, extractDir)
	if err != nil {
		if errors.Is(err, zip.ErrFormat) {
			log.Printf("ZIP extraction failed for %s, attempting RAR extraction (file may be misnamed .cbr as .cbz)", filepath.Base(cbzPath))

			file, rarErr := os.Open(cbzPath)
			if rarErr != nil {
				return fmt.Errorf("failed to extract CBZ file: %w (also failed to open for RAR attempt: %v)", err, rarErr)
			}
			defer func() {
				if closeErr := file.Close(); closeErr != nil {
					log.Printf("error closing file %s: %v", cbzPath, closeErr)
				}
			}()

			r, rarErr := rardecode.NewReader(file)
			if rarErr != nil {
				return fmt.Errorf("failed to extract as ZIP or RAR: ZIP error: %w, RAR error: %v", err, rarErr)
			}

			if rarErr := os.MkdirAll(extractDir, os.ModePerm); rarErr != nil {
				return fmt.Errorf("failed to create extract directory: %w", rarErr)
			}

			for {
				header, rarErr := r.Next()
				if rarErr == io.EOF {
					break
				}
				if rarErr != nil {
					return fmt.Errorf("failed to read RAR entry: %w", rarErr)
				}

				if header.IsDir {
					continue
				}

				cleanName := filepath.Clean(header.Name)

				// Path traversal protection
				if filepath.IsAbs(cleanName) {
					log.Printf("skipping potentially unsafe absolute path: %s", header.Name)
					continue
				}
				parts := strings.Split(cleanName, string(os.PathSeparator))
				unsafePath := false
				for _, part := range parts {
					if part == ".." {
						unsafePath = true
						break
					}
				}
				if unsafePath {
					log.Printf("skipping potentially unsafe file: %s", header.Name)
					continue
				}

				destPath := filepath.Join(extractDir, cleanName)
				if rarErr := os.MkdirAll(filepath.Dir(destPath), os.ModePerm); rarErr != nil {
					return fmt.Errorf("failed to create directory %s: %w", filepath.Dir(destPath), rarErr)
				}

				outFile, rarErr := os.Create(destPath)
				if rarErr != nil {
					return fmt.Errorf("failed to create file %s: %w", destPath, rarErr)
				}

				_, rarErr = io.Copy(outFile, r)
				if closeErr := outFile.Close(); closeErr != nil {
					return fmt.Errorf("failed to close file %s: %w", destPath, closeErr)
				}
				if rarErr != nil {
					return fmt.Errorf("failed to extract file %s: %w", header.Name, rarErr)
				}
			}

			log.Printf("Successfully extracted %s as RAR format (misnamed file)", filepath.Base(cbzPath))
		} else {
			return fmt.Errorf("failed to extract CBZ file: %w", err)
		}
	}

	if err := renameFilesWithLeadingZeros(extractDir); err != nil {
		return fmt.Errorf("failed to rename files: %w", err)
	}

	return nil
}

func extractAndRenamePDF(pdfPath, extractDir string) error {
	if err := os.MkdirAll(extractDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create extract directory: %w", err)
	}

	if err := api.ExtractImagesFile(pdfPath, extractDir, nil, nil); err != nil {
		return fmt.Errorf("failed to extract images from PDF %s: %w", pdfPath, err)
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

func extractTomeNumber(filename string) string {
	tomeRegex := regexp.MustCompile(`(?i)(?:tome|t)[.\s]*(\d+)`)
	matches := tomeRegex.FindStringSubmatch(filename)
	if len(matches) > 1 {
		num, err := strconv.Atoi(matches[1])
		if err == nil {
			return fmt.Sprintf("%02d", num)
		}
		return matches[1]
	}

	numRegex := regexp.MustCompile(`\d+`)
	matches = numRegex.FindStringSubmatch(filename)
	if len(matches) > 0 {
		num, err := strconv.Atoi(matches[0])
		if err == nil {
			return fmt.Sprintf("%02d", num)
		}
		return matches[0]
	}

	return ""
}

func renameFile(oldPath, seriesName string) (string, error) {
	ext := strings.ToLower(filepath.Ext(oldPath))
	if ext != ".cbz" && ext != ".cbr" && ext != ".pdf" {
		return oldPath, nil
	}

	filename := filepath.Base(oldPath)
	tomeNumber := extractTomeNumber(filename)

	if tomeNumber == "" {
		return oldPath, fmt.Errorf("unable to find tome number in %s", filename)
	}

	newName := fmt.Sprintf("%s T%s%s", seriesName, tomeNumber, ".cbz")
	newPath := filepath.Join(filepath.Dir(oldPath), newName)

	if newPath == oldPath {
		return oldPath, nil
	}

	return newPath, nil
}

var (
	infoColor  = color.New(color.FgCyan).SprintFunc()
	successColor = color.New(color.FgGreen).SprintFunc()
	errorColor = color.New(color.FgRed).SprintFunc()
)

func logMessage(mu *sync.Mutex, message string) {
	mu.Lock()
	defer mu.Unlock()
	fmt.Printf("• %s\n", message)
}

func main() {
	seriesNamePtr := flag.String("name", "", "Name of the series to rename the files to")
	flag.Parse()

	seriesName := *seriesNamePtr

	if seriesName == "" && flag.NArg() > 0 {
		seriesName = strings.Join(flag.Args(), " ")
	}

	dir := "./"

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Failed to read directory %s: %v", dir, err)
	}

	var cbFiles []os.DirEntry
	for _, file := range files {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == ".cbz" || ext == ".cbr" || ext == ".pdf" {
			cbFiles = append(cbFiles, file)
		}
	}

	if len(cbFiles) == 0 {
		fmt.Println("No CBR/CBZ/PDF files found in the current directory.")
		return
	}

	bar := progressbar.NewOptions(len(cbFiles),
		progressbar.OptionShowCount(),
		progressbar.OptionSetDescription("Processing comic archives"),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)

	var wg sync.WaitGroup
	var mu sync.Mutex

	fmt.Printf("Found %d comic archives to process.\n", len(cbFiles))
	if seriesName != "" {
		fmt.Printf("Series name set to: %s\n", seriesName)
	}
	fmt.Println("Starting processing...")

	for _, file := range cbFiles {
		ext := strings.ToLower(filepath.Ext(file.Name()))
		if ext == ".cbz" || ext == ".cbr" || ext == ".pdf" {
			wg.Add(1)
			go func(file os.DirEntry) {
				defer wg.Done()

				filePath := filepath.Join(dir, file.Name())
				extractDir := filepath.Join(dir, strings.TrimSuffix(file.Name(), ext)+"_extracted")

				newFilePath := filePath
				var newCBZPath string

				if seriesName != "" {
					var err error
					newFilePath, err = renameFile(filePath, seriesName)
					if err != nil {
						logMessage(&mu, errorColor(fmt.Sprintf("Error with %s: %v", file.Name(), err)))
						return
					}

					if newFilePath != filePath {
						newExt := filepath.Ext(newFilePath)
						extractDir = filepath.Join(dir, strings.TrimSuffix(filepath.Base(newFilePath), newExt)+"_extracted")
						logMessage(&mu, infoColor(fmt.Sprintf("Renaming: %s → %s", filepath.Base(filePath), filepath.Base(newFilePath))))
					}
				}

				if ext == ".cbr" || ext == ".pdf" {
					newCBZPath = strings.TrimSuffix(newFilePath, ext) + ".cbz"
				} else {
					newCBZPath = newFilePath
				}

				logMessage(&mu, infoColor(fmt.Sprintf("Processing: %s", filepath.Base(filePath))))

				if err := extractAndRenameArchive(filePath, extractDir); err != nil {
					logMessage(&mu, errorColor(fmt.Sprintf("Error extracting %s: %v", filepath.Base(filePath), err)))
					return
				}

				if ext == ".cbr" || ext == ".pdf" {
					logMessage(&mu, infoColor(fmt.Sprintf("Converting to CBZ: %s", filepath.Base(newCBZPath))))

					if err := zipFiles(newCBZPath, extractDir); err != nil {
						logMessage(&mu, errorColor(fmt.Sprintf("Error creating CBZ for %s: %v", filepath.Base(filePath), err)))
						return
					}

					if err := os.Remove(filePath); err != nil {
						logMessage(&mu, errorColor(fmt.Sprintf("Error removing original %s: %v", filepath.Base(filePath), err)))
					}

					logMessage(&mu, successColor(fmt.Sprintf("Converted: %s → %s", filepath.Base(filePath), filepath.Base(newCBZPath))))
				} else {
					logMessage(&mu, infoColor(fmt.Sprintf("Updating CBZ: %s", filepath.Base(newFilePath))))

					if newFilePath != filePath {
						if err := zipFiles(newCBZPath, extractDir); err != nil {
							logMessage(&mu, errorColor(fmt.Sprintf("Error creating new CBZ %s: %v", filepath.Base(newFilePath), err)))
							return
						}
						if err := os.Remove(filePath); err != nil {
							logMessage(&mu, errorColor(fmt.Sprintf("Error removing original %s: %v", filepath.Base(filePath), err)))
						}
					} else {
						if err := os.Remove(filePath); err != nil {
							logMessage(&mu, errorColor(fmt.Sprintf("Error removing original %s: %v", filepath.Base(filePath), err)))
							return
						}
						if err := zipFiles(filePath, extractDir); err != nil {
							logMessage(&mu, errorColor(fmt.Sprintf("Error updating CBZ %s: %v", filepath.Base(filePath), err)))
							return
						}
						if err := zipFiles(filePath, extractDir); err != nil {
							logMessage(&mu, errorColor(fmt.Sprintf("Error updating CBZ %s: %v", filepath.Base(filePath), err)))
							return
						}
					}

					logMessage(&mu, successColor(fmt.Sprintf("Updated: %s", filepath.Base(newCBZPath))))
				}

				os.RemoveAll(extractDir)
				logMessage(&mu, successColor(fmt.Sprintf("✓ Completed: %s", filepath.Base(newCBZPath))))

				mu.Lock()
				bar.Add(1)
				mu.Unlock()
			}(file)
		}
	}

	wg.Wait()
	bar.Finish()
	fmt.Println("\n✅ All files processed successfully!")
}
