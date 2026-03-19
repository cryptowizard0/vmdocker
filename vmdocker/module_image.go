package vmdocker

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	runtimeSchema "github.com/cryptowizard0/vmdocker/vmdocker/runtimemanager/schema"
)

var dockerLookPath = exec.LookPath

func ensureModuleImageAvailable(ctx context.Context, moduleID string, image runtimeSchema.ImageInfo) error {
	if image.Source == "" && image.ArchiveFormat == "" {
		return nil
	}
	if image.Source != runtimeSchema.ImageSourceModuleData {
		return fmt.Errorf("unsupported image source %q", image.Source)
	}
	if image.ArchiveFormat != runtimeSchema.ImageArchiveDockerSaveGZ {
		return fmt.Errorf("unsupported image archive format %q", image.ArchiveFormat)
	}

	cliBin, err := dockerBinary()
	if err != nil {
		return err
	}

	matched, err := imageMatchesRef(ctx, cliBin, image.Name, image.SHA)
	if err == nil && matched {
		return nil
	}
	if err == nil && !matched {
		log.Info("local image tag exists but sha mismatched, reloading from module", "module", moduleID, "image", image.Name, "expected_sha", image.SHA)
	}

	if err := ensureImageTaggedByID(ctx, cliBin, image.SHA, image.Name); err == nil {
		return nil
	}

	if err := dockerLoadArchive(ctx, cliBin, moduleID); err != nil {
		return err
	}
	if err := ensureImageTaggedByID(ctx, cliBin, image.SHA, image.Name); err != nil {
		return fmt.Errorf("loaded image from module %s but failed to tag/verify %s: %w", moduleID, image.Name, err)
	}
	return nil
}

func dockerBinary() (string, error) {
	cliBin, err := dockerLookPath("docker")
	if err != nil {
		return "", fmt.Errorf("docker CLI is not available: %w", err)
	}
	return cliBin, nil
}

func imageMatchesRef(ctx context.Context, cliBin, ref, expectedID string) (bool, error) {
	actualID, err := inspectLocalImageID(ctx, cliBin, ref)
	if err != nil {
		return false, err
	}
	return actualID == expectedID, nil
}

func inspectLocalImageID(ctx context.Context, cliBin, ref string) (string, error) {
	cmd := exec.CommandContext(ctx, cliBin, "image", "inspect", "--format", "{{.Id}}", ref)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("inspect image %s failed: %w: %s", ref, err, strings.TrimSpace(string(output)))
	}
	return strings.TrimSpace(string(output)), nil
}

func ensureImageTaggedByID(ctx context.Context, cliBin, imageID, imageName string) error {
	if _, err := inspectLocalImageID(ctx, cliBin, imageID); err != nil {
		return err
	}
	if matched, err := imageMatchesRef(ctx, cliBin, imageName, imageID); err == nil && matched {
		return nil
	}
	cmd := exec.CommandContext(ctx, cliBin, "image", "tag", imageID, imageName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("tag image %s as %s failed: %w: %s", imageID, imageName, err, strings.TrimSpace(string(output)))
	}
	matched, err := imageMatchesRef(ctx, cliBin, imageName, imageID)
	if err != nil {
		return err
	}
	if !matched {
		return fmt.Errorf("image %s does not match expected id %s after tagging", imageName, imageID)
	}
	return nil
}

func dockerLoadArchive(ctx context.Context, cliBin, moduleID string) error {
	modulePath, err := resolveModuleFilePath(moduleID)
	if err != nil {
		return fmt.Errorf("read module file for %s failed: %w", moduleID, err)
	}

	file, err := os.Open(modulePath)
	if err != nil {
		return fmt.Errorf("open module file %s failed: %w", modulePath, err)
	}
	defer file.Close()

	dataReader, err := newModuleDataReader(file)
	if err != nil {
		return fmt.Errorf("read module %s payload stream failed: %w", moduleID, err)
	}

	base64Reader := base64.NewDecoder(base64.RawURLEncoding, dataReader)
	reader, err := gzip.NewReader(base64Reader)
	if err != nil {
		return fmt.Errorf("open gzip payload failed: %w", err)
	}
	defer reader.Close()

	cmd := exec.CommandContext(ctx, cliBin, "image", "load")
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("open docker load stdin failed: %w", err)
	}
	copyErrCh := make(chan error, 1)
	go func() {
		_, copyErr := io.Copy(stdin, reader)
		closeErr := stdin.Close()
		if copyErr == nil {
			copyErr = closeErr
		}
		copyErrCh <- copyErr
	}()
	output, err := cmd.CombinedOutput()
	copyErr := <-copyErrCh
	if copyErr != nil && !errors.Is(copyErr, io.EOF) {
		return fmt.Errorf("stream docker image load payload failed: %w", copyErr)
	}
	if err != nil {
		return fmt.Errorf("docker image load failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func moduleFilePath(moduleID string) string {
	return filepath.Join("mod", fmt.Sprintf("mod-%s.json", moduleID))
}

func legacyModuleFilePath(moduleID string) string {
	return fmt.Sprintf("mod-%s.json", moduleID)
}

func resolveModuleFilePath(moduleID string) (string, error) {
	candidates := []string{
		moduleFilePath(moduleID),
		legacyModuleFilePath(moduleID),
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
	}
	return "", os.ErrNotExist
}

func newModuleDataReader(file *os.File) (io.Reader, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(file)
	inObject := 0
	expectingKey := false
	for {
		tok, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, fmt.Errorf("module data field not found")
			}
			return nil, err
		}
		switch v := tok.(type) {
		case json.Delim:
			switch v {
			case '{':
				inObject++
				expectingKey = true
			case '}':
				if inObject > 0 {
					inObject--
				}
				expectingKey = inObject > 0
			case '[':
				expectingKey = false
			case ']':
				expectingKey = inObject > 0
			}
		case string:
			if inObject > 0 && expectingKey {
				if v == "data" {
					return newJSONStringValueReader(file, decoder.InputOffset())
				}
				expectingKey = false
				continue
			}
			if inObject > 0 {
				expectingKey = true
			}
		default:
			if inObject > 0 && !expectingKey {
				expectingKey = true
			}
		}
	}
}

type jsonStringValueReader struct {
	reader *bufio.Reader
	buf    []byte
	done   bool
	escape bool
}

func (r *jsonStringValueReader) Read(p []byte) (int, error) {
	if r.done {
		return 0, io.EOF
	}
	n := 0
	for n < len(p) {
		if len(r.buf) > 0 {
			copied := copy(p[n:], r.buf)
			r.buf = r.buf[copied:]
			n += copied
			if n == len(p) {
				return n, nil
			}
			continue
		}

		b, err := r.reader.ReadByte()
		if err != nil {
			if errors.Is(err, io.EOF) {
				if n > 0 {
					return n, fmt.Errorf("unexpected EOF while reading module data field")
				}
				return 0, fmt.Errorf("unexpected EOF while reading module data field")
			}
			if n > 0 {
				return n, err
			}
			return 0, err
		}

		if r.escape {
			decoded, err := decodeJSONStringEscape(r.reader, b)
			if err != nil {
				if n > 0 {
					return n, err
				}
				return 0, err
			}
			r.escape = false
			if len(decoded) == 0 {
				continue
			}
			r.buf = decoded
			continue
		}

		switch b {
		case '\\':
			r.escape = true
		case '"':
			r.done = true
			if n == 0 {
				return 0, io.EOF
			}
			return n, io.EOF
		default:
			p[n] = b
			n++
		}
	}
	return n, nil
}

func newJSONStringValueReader(file *os.File, offset int64) (io.Reader, error) {
	if _, err := file.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}
	reader := bufio.NewReader(file)
	if err := consumeJSONStringValueStart(reader); err != nil {
		return nil, err
	}
	return &jsonStringValueReader{reader: reader}, nil
}

func consumeJSONStringValueStart(reader *bufio.Reader) error {
	b, err := readJSONNonSpaceByte(reader)
	if err != nil {
		return err
	}
	if b != ':' {
		return fmt.Errorf("expected ':' before module data value, got %q", b)
	}
	b, err = readJSONNonSpaceByte(reader)
	if err != nil {
		return err
	}
	if b != '"' {
		return fmt.Errorf("expected '\"' to start module data value, got %q", b)
	}
	return nil
}

func readJSONNonSpaceByte(reader *bufio.Reader) (byte, error) {
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return 0, err
		}
		switch b {
		case ' ', '\n', '\r', '\t':
			continue
		default:
			return b, nil
		}
	}
}

func decodeJSONStringEscape(reader *bufio.Reader, esc byte) ([]byte, error) {
	switch esc {
	case '"', '\\', '/':
		return []byte{esc}, nil
	case 'b':
		return []byte{'\b'}, nil
	case 'f':
		return []byte{'\f'}, nil
	case 'n':
		return []byte{'\n'}, nil
	case 'r':
		return []byte{'\r'}, nil
	case 't':
		return []byte{'\t'}, nil
	case 'u':
		r, err := readUnicodeEscape(reader)
		if err != nil {
			return nil, err
		}
		return []byte(string(r)), nil
	default:
		return nil, fmt.Errorf("unsupported JSON escape sequence \\%c in module data field", esc)
	}
}

func readUnicodeEscape(reader *bufio.Reader) (rune, error) {
	hex, err := readExactBytes(reader, 4)
	if err != nil {
		return 0, err
	}
	value, err := strconv.ParseUint(string(hex), 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid unicode escape in module data field: %w", err)
	}
	first := rune(value)
	if !utf16.IsSurrogate(first) {
		return first, nil
	}

	slash, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}
	u, err := reader.ReadByte()
	if err != nil {
		return 0, err
	}
	if slash != '\\' || u != 'u' {
		return 0, fmt.Errorf("invalid surrogate pair in module data field")
	}
	secondHex, err := readExactBytes(reader, 4)
	if err != nil {
		return 0, err
	}
	secondValue, err := strconv.ParseUint(string(secondHex), 16, 16)
	if err != nil {
		return 0, fmt.Errorf("invalid unicode escape in module data field: %w", err)
	}
	second := rune(secondValue)
	decoded := utf16.DecodeRune(first, second)
	if decoded == utf8.RuneError {
		return 0, fmt.Errorf("invalid surrogate pair in module data field")
	}
	return decoded, nil
}

func readExactBytes(reader *bufio.Reader, n int) ([]byte, error) {
	buf := make([]byte, n)
	if _, err := io.ReadFull(reader, buf); err != nil {
		return nil, err
	}
	return buf, nil
}
