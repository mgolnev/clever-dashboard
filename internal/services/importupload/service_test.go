package importupload

import (
	"bytes"
	"io"
	"testing"
)

func TestUploadReassemblesChunks(t *testing.T) {
	service := &Service{root: t.TempDir(), chunkSize: 8, maxFileSize: 1024}
	data := []byte("0123456789abcdefghij")
	session, err := service.Create("orders.xls", int64(len(data)))
	if err != nil {
		t.Fatal(err)
	}
	if session.ChunksTotal != 3 {
		t.Fatalf("chunksTotal=%d, want 3", session.ChunksTotal)
	}
	for index := 0; index < session.ChunksTotal; index++ {
		start := index * int(session.ChunkSize)
		end := start + int(session.ChunkSize)
		if end > len(data) {
			end = len(data)
		}
		if err := service.SaveChunk(session.ID, index, bytes.NewReader(data[start:end])); err != nil {
			t.Fatalf("save chunk %d: %v", index, err)
		}
	}
	upload, err := service.Open(session.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := io.ReadAll(upload.Reader())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Fatalf("reassembled=%q, want %q", got, data)
	}
	if err := upload.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := service.loadSession(session.ID); err == nil {
		t.Fatal("session should be removed after Close")
	}
}

func TestUploadRejectsWrongChunkSize(t *testing.T) {
	service := &Service{root: t.TempDir(), chunkSize: 8, maxFileSize: 1024}
	session, err := service.Create("orders.xls", 10)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.SaveChunk(session.ID, 0, bytes.NewReader([]byte("short"))); err == nil {
		t.Fatal("expected wrong chunk size error")
	}
}
