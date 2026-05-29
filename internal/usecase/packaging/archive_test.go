package packaging

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"errors"
	"io/fs"
	"testing"
)

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
