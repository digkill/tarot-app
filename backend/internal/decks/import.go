package decks

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"

	xdraw "golang.org/x/image/draw"
)

const maxCardWidth = 720

type ImportResult struct {
	Cards int
	Back  bool
	Cover bool
	Skip  int
}

func ImportZip(zipPath, destDir string) (ImportResult, error) {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return ImportResult{}, fmt.Errorf("open zip: %w", err)
	}
	defer zr.Close()
	return importFiles(destDir, func(yield func(name string, r io.Reader) error) error {
		for _, f := range zr.File {
			if f.FileInfo().IsDir() {
				continue
			}
			rc, err := f.Open()
			if err != nil {
				return err
			}
			err = yield(f.Name, rc)
			_ = rc.Close()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func ImportDir(srcDir, destDir string) (ImportResult, error) {
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return ImportResult{}, fmt.Errorf("read dir: %w", err)
	}
	return importFiles(destDir, func(yield func(name string, r io.Reader) error) error {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			f, err := os.Open(filepath.Join(srcDir, e.Name()))
			if err != nil {
				return err
			}
			err = yield(e.Name(), f)
			_ = f.Close()
			if err != nil {
				return err
			}
		}
		return nil
	})
}

func importFiles(destDir string, walk func(func(name string, r io.Reader) error) error) (ImportResult, error) {
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return ImportResult{}, err
	}
	var res ImportResult
	err := walk(func(name string, r io.Reader) error {
		key, ok := MapFilename(name)
		if !ok {
			res.Skip++
			return nil
		}
		payload, err := io.ReadAll(r)
		if err != nil {
			return fmt.Errorf("read %s: %w", name, err)
		}
		jpg, err := toJPEG(payload)
		if err != nil {
			return fmt.Errorf("convert %s: %w", name, err)
		}
		out := filepath.Join(destDir, key+".jpg")
		if err = os.WriteFile(out, jpg, 0o644); err != nil {
			return err
		}
		switch key {
		case FileBack:
			res.Back = true
		case FileCover:
			res.Cover = true
		default:
			res.Cards++
		}
		return nil
	})
	if err != nil {
		return res, err
	}
	coverPath := filepath.Join(destDir, FileCover+".jpg")
	if _, err := os.Stat(coverPath); os.IsNotExist(err) {
		fool := filepath.Join(destDir, "the_fool.jpg")
		if _, err = os.Stat(fool); err == nil {
			_ = copyFile(fool, coverPath)
			res.Cover = true
		}
	}
	return res, nil
}

func toJPEG(payload []byte) ([]byte, error) {
	img, err := decodeImage(payload)
	if err != nil {
		return nil, err
	}
	img = scaleMaxWidth(img, maxCardWidth)
	var buf bytes.Buffer
	if err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 82}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func decodeImage(payload []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(payload))
	if err == nil {
		return img, nil
	}
	if img, err2 := png.Decode(bytes.NewReader(payload)); err2 == nil {
		return img, nil
	}
	if img, err2 := jpeg.Decode(bytes.NewReader(payload)); err2 == nil {
		return img, nil
	}
	return nil, err
}

func scaleMaxWidth(src image.Image, maxW int) image.Image {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= maxW || maxW <= 0 {
		return src
	}
	nh := h * maxW / w
	dst := image.NewRGBA(image.Rect(0, 0, maxW, nh))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, b, xdraw.Over, nil)
	return dst
}

func copyFile(src, dst string) error {
	in, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, in, 0o644)
}

func SanitizeMediaFile(name string) (string, bool) {
	base := pathBase(name)
	key := strings.TrimSuffix(base, filepath.Ext(base))
	if key == FileBack || key == FileCover || IsCardKey(key) {
		if strings.ToLower(filepath.Ext(base)) == ".jpg" {
			return key + ".jpg", true
		}
	}
	return "", false
}

func pathBase(name string) string {
	name = strings.ReplaceAll(name, "\\", "/")
	return filepath.Base(name)
}
