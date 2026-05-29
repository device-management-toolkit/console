package packaging

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"io/fs"
	"testing"
)

// neverendingReader is a Reader that fills any buffer with a constant byte and
// never returns io.EOF, used to drive readLimited past the cap without
// allocating a large buffer.
type neverendingReader struct{}

func (neverendingReader) Read(p []byte) (int, error) {
	for i := range p {
		p[i] = 0xAB
	}

	return len(p), nil
}

func makeTarGz(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buf bytes.Buffer

	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	hdr := &tar.Header{Name: name, Mode: 0o755, Size: int64(len(content))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}

	if _, err := tw.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}

	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func makeZip(t *testing.T, name string, content []byte) []byte {
	t.Helper()

	var buf bytes.Buffer

	zw := zip.NewWriter(&buf)

	w, err := zw.Create(name)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := w.Write(content); err != nil {
		t.Fatal(err)
	}

	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}

	return buf.Bytes()
}

func TestExtractBinaryTarGz(t *testing.T) {
	t.Parallel()

	data := makeTarGz(t, "rpc", []byte("ELF-bytes"))

	name, content, err := extractBinary(data, "rpc-go_Linux_x86_64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}

	if name != "rpc" || string(content) != "ELF-bytes" {
		t.Fatalf("got (%q,%q)", name, content)
	}
}

func TestExtractBinaryZip(t *testing.T) {
	t.Parallel()

	data := makeZip(t, "rpc.exe", []byte("PE-bytes"))

	name, content, err := extractBinary(data, "rpc-go_Windows_x86_64.zip")
	if err != nil {
		t.Fatal(err)
	}

	if name != "rpc.exe" || string(content) != "PE-bytes" {
		t.Fatalf("got (%q,%q)", name, content)
	}
}

func TestExtractBinaryNotFound(t *testing.T) {
	t.Parallel()

	data := makeTarGz(t, "README.md", []byte("x"))

	_, _, err := extractBinary(data, "rpc-go_Linux_x86_64.tar.gz")
	if !errors.Is(err, ErrBinaryNotFound) {
		t.Fatalf("expected ErrBinaryNotFound, got %v", err)
	}
}

func TestBuildZip(t *testing.T) {
	t.Parallel()

	out, err := buildZip("rpc", []byte("bin"), []byte("cfg"))
	if err != nil {
		t.Fatal(err)
	}

	zr, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}

	found := map[string]string{}
	modes := map[string]fs.FileMode{}

	for _, f := range zr.File {
		rc, err := f.Open()
		if err != nil {
			t.Fatal(err)
		}

		var b bytes.Buffer
		if _, err := b.ReadFrom(rc); err != nil {
			t.Fatal(err)
		}

		rc.Close()

		found[f.Name] = b.String()
		modes[f.Name] = f.Mode()
	}

	if found["rpc"] != "bin" || found["config.yaml"] != "cfg" {
		t.Fatalf("unexpected zip contents: %v", found)
	}

	if modes["rpc"]&0o111 == 0 {
		t.Fatalf("expected rpc to be executable, mode = %v", modes["rpc"])
	}
}

func TestReadLimitedEntryTooLarge(t *testing.T) {
	t.Parallel()

	// Provide a reader that yields exactly maxArchiveBytes+10 bytes before being
	// cut off by io.LimitReader — enough to exceed the cap without allocating the
	// full buffer in memory.
	r := io.LimitReader(neverendingReader{}, maxArchiveBytes+10)

	_, err := readLimited(r)
	if !errors.Is(err, ErrEntryTooLarge) {
		t.Fatalf("expected ErrEntryTooLarge, got: %v", err)
	}
}
