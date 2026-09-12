// Package importupload управляет временными сессиями загрузки больших файлов.
// Он отвечает только за транспорт и сборку частей, не знает формат выгрузки и БД.
package importupload

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	defaultChunkSize   int64 = 4 * 1024 * 1024
	defaultMaxFileSize int64 = 512 * 1024 * 1024
	uploadTTL                = 24 * time.Hour
)

var uploadIDPattern = regexp.MustCompile(`^[a-f0-9]{32}$`)

type Service struct {
	root        string
	chunkSize   int64
	maxFileSize int64
}

type Session struct {
	ID          string `json:"uploadId"`
	Filename    string `json:"filename"`
	Size        int64  `json:"size"`
	ChunkSize   int64  `json:"chunkSize"`
	ChunksTotal int    `json:"chunksTotal"`
}

type Upload struct {
	Session
	dir     string
	files   []*os.File
	readers []io.Reader
}

func New(root string) *Service {
	if strings.TrimSpace(root) == "" {
		root = filepath.Join(os.TempDir(), "clever-dashboard-imports")
	}
	return &Service{root: root, chunkSize: defaultChunkSize, maxFileSize: defaultMaxFileSize}
}

func (s *Service) Create(filename string, size int64) (*Session, error) {
	filename = filepath.Base(filename)
	if filename == "" || filename == "." {
		return nil, fmt.Errorf("требуется имя файла")
	}
	if size <= 0 {
		return nil, fmt.Errorf("файл пуст")
	}
	if size > s.maxFileSize {
		return nil, fmt.Errorf("размер файла превышает лимит %d МБ", s.maxFileSize/(1024*1024))
	}
	if err := os.MkdirAll(s.root, 0o700); err != nil {
		return nil, fmt.Errorf("создать каталог загрузок: %w", err)
	}
	s.cleanupStale()
	id, err := randomID()
	if err != nil {
		return nil, fmt.Errorf("создать идентификатор загрузки: %w", err)
	}
	session := &Session{
		ID: id, Filename: filename, Size: size, ChunkSize: s.chunkSize,
		ChunksTotal: int((size + s.chunkSize - 1) / s.chunkSize),
	}
	dir := s.sessionDir(id)
	if err := os.Mkdir(dir, 0o700); err != nil {
		return nil, fmt.Errorf("создать сессию загрузки: %w", err)
	}
	encoded, err := json.Marshal(session)
	if err != nil {
		_ = os.RemoveAll(dir)
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "meta.json"), encoded, 0o600); err != nil {
		_ = os.RemoveAll(dir)
		return nil, fmt.Errorf("сохранить сессию загрузки: %w", err)
	}
	return session, nil
}

func (s *Service) SaveChunk(id string, index int, reader io.Reader) error {
	session, dir, err := s.loadSession(id)
	if err != nil {
		return err
	}
	if index < 0 || index >= session.ChunksTotal {
		return fmt.Errorf("неверный номер части %d", index)
	}
	expected := session.ChunkSize
	if remaining := session.Size - int64(index)*session.ChunkSize; remaining < expected {
		expected = remaining
	}
	temporary, err := os.CreateTemp(dir, ".chunk-")
	if err != nil {
		return fmt.Errorf("создать временную часть: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	written, copyErr := io.Copy(temporary, io.LimitReader(reader, expected+1))
	closeErr := temporary.Close()
	if copyErr != nil {
		return fmt.Errorf("записать часть: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("закрыть часть: %w", closeErr)
	}
	if written != expected {
		return fmt.Errorf("неверный размер части %d: получено %d, ожидалось %d", index, written, expected)
	}
	if err := os.Rename(temporaryName, chunkPath(dir, index)); err != nil {
		return fmt.Errorf("сохранить часть: %w", err)
	}
	now := time.Now()
	_ = os.Chtimes(dir, now, now)
	return nil
}

func (s *Service) Open(id string) (*Upload, error) {
	session, dir, err := s.loadSession(id)
	if err != nil {
		return nil, err
	}
	upload := &Upload{Session: *session, dir: dir}
	for index := 0; index < session.ChunksTotal; index++ {
		path := chunkPath(dir, index)
		info, err := os.Stat(path)
		if err != nil {
			upload.closeFiles()
			return nil, fmt.Errorf("часть %d не загружена", index)
		}
		expected := session.ChunkSize
		if remaining := session.Size - int64(index)*session.ChunkSize; remaining < expected {
			expected = remaining
		}
		if info.Size() != expected {
			upload.closeFiles()
			return nil, fmt.Errorf("часть %d повреждена", index)
		}
		file, err := os.Open(path)
		if err != nil {
			upload.closeFiles()
			return nil, fmt.Errorf("открыть часть %d: %w", index, err)
		}
		upload.files = append(upload.files, file)
		upload.readers = append(upload.readers, file)
	}
	return upload, nil
}

func (u *Upload) Reader() io.Reader { return io.MultiReader(u.readers...) }

// Close закрывает части и удаляет временную сессию после попытки импорта.
func (u *Upload) Close() error {
	closeErr := u.closeFiles()
	removeErr := os.RemoveAll(u.dir)
	if closeErr != nil {
		return closeErr
	}
	return removeErr
}

func (u *Upload) closeFiles() error {
	var firstErr error
	for _, file := range u.files {
		if err := file.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	u.files = nil
	u.readers = nil
	return firstErr
}

func (s *Service) Delete(id string) error {
	if !uploadIDPattern.MatchString(id) {
		return fmt.Errorf("неверный идентификатор загрузки")
	}
	return os.RemoveAll(s.sessionDir(id))
}

func (s *Service) loadSession(id string) (*Session, string, error) {
	if !uploadIDPattern.MatchString(id) {
		return nil, "", fmt.Errorf("неверный идентификатор загрузки")
	}
	dir := s.sessionDir(id)
	encoded, err := os.ReadFile(filepath.Join(dir, "meta.json"))
	if err != nil {
		return nil, "", fmt.Errorf("сессия загрузки не найдена")
	}
	var session Session
	if err := json.Unmarshal(encoded, &session); err != nil || session.ID != id {
		return nil, "", fmt.Errorf("сессия загрузки повреждена")
	}
	return &session, dir, nil
}

func (s *Service) sessionDir(id string) string { return filepath.Join(s.root, id) }

func (s *Service) cleanupStale() {
	entries, err := os.ReadDir(s.root)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-uploadTTL)
	for _, entry := range entries {
		if !entry.IsDir() || !uploadIDPattern.MatchString(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(s.sessionDir(entry.Name()))
		}
	}
}

func chunkPath(dir string, index int) string {
	return filepath.Join(dir, fmt.Sprintf("%06d.part", index))
}

func randomID() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
