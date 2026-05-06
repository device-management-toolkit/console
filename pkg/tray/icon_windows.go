//go:build tray && windows

package tray

import "encoding/binary"

// Windows systray expects ICO format, not raw PNG. Wrap the embedded PNG
// in a single-image ICO container at startup.
func init() {
	iconData = pngToICO(iconData)
}

// pngToICO wraps PNG bytes in a 1-image ICO container. The PNG is 44x44.
func pngToICO(png []byte) []byte {
	const (
		iconWidth  = 44
		iconHeight = 44
		dataOffset = 22 // ICONDIR (6) + ICONDIRENTRY (16)
	)

	ico := make([]byte, 0, dataOffset+len(png))

	// ICONDIR: reserved=0, type=1 (icon), count=1
	ico = append(ico, 0, 0, 1, 0, 1, 0)

	// ICONDIRENTRY
	ico = append(ico,
		iconWidth,  // width
		iconHeight, // height
		0,          // color count (0 for true color)
		0,          // reserved
	)
	ico = binary.LittleEndian.AppendUint16(ico, 1)                // planes
	ico = binary.LittleEndian.AppendUint16(ico, 32)               // bits per pixel
	ico = binary.LittleEndian.AppendUint32(ico, uint32(len(png))) // image size
	ico = binary.LittleEndian.AppendUint32(ico, dataOffset)       // image offset

	ico = append(ico, png...)

	return ico
}
