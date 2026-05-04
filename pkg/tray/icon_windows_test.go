//go:build tray && windows

package tray

import (
	"encoding/binary"
	"testing"
)

// pngToICO mutates package state (init replaces iconData), so we test the
// wrapper directly with a known-shape PNG.
func TestPNGToICOHeader(t *testing.T) {
	t.Parallel()

	// Minimal PNG signature is enough — pngToICO doesn't parse the contents.
	png := []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a, 0xde, 0xad, 0xbe, 0xef}

	ico := pngToICO(png)

	const headerLen = 22
	if len(ico) != headerLen+len(png) {
		t.Fatalf("ico length = %d, want %d", len(ico), headerLen+len(png))
	}

	// ICONDIR: reserved=0, type=1, count=1
	if ico[0] != 0 || ico[1] != 0 {
		t.Errorf("ICONDIR reserved bytes = %v, want [0 0]", ico[:2])
	}

	if got := binary.LittleEndian.Uint16(ico[2:4]); got != 1 {
		t.Errorf("ICONDIR type = %d, want 1", got)
	}

	if got := binary.LittleEndian.Uint16(ico[4:6]); got != 1 {
		t.Errorf("ICONDIR count = %d, want 1", got)
	}

	// ICONDIRENTRY at offset 6
	if ico[6] != 44 || ico[7] != 44 {
		t.Errorf("entry width/height = %d/%d, want 44/44", ico[6], ico[7])
	}

	if got := binary.LittleEndian.Uint16(ico[10:12]); got != 1 {
		t.Errorf("entry planes = %d, want 1", got)
	}

	if got := binary.LittleEndian.Uint16(ico[12:14]); got != 32 {
		t.Errorf("entry bpp = %d, want 32", got)
	}

	if got := binary.LittleEndian.Uint32(ico[14:18]); got != uint32(len(png)) {
		t.Errorf("entry image size = %d, want %d", got, len(png))
	}

	if got := binary.LittleEndian.Uint32(ico[18:22]); got != headerLen {
		t.Errorf("entry image offset = %d, want %d", got, headerLen)
	}

	// Image data follows the header.
	for i, b := range png {
		if ico[headerLen+i] != b {
			t.Errorf("image byte %d = 0x%02x, want 0x%02x", i, ico[headerLen+i], b)
			break
		}
	}
}
