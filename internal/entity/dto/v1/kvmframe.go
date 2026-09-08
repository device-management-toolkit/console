package dto

import "time"

// KVMFrame is a single captured KVM screen frame returned as raw pixel data.
//
// Decoding/rendering is intentionally left to the caller (e.g. an agentic AI /
// VLM). The pixel data is a full-screen framebuffer in the requested pixel
// format (RGB332, 1 byte per pixel) laid out row-major, top-to-bottom, base64
// encoded. Each RGB332 byte packs colour as: bits 5-7 red (0-7), bits 2-4 green
// (0-7), bits 0-1 blue (0-3).
type KVMFrame struct {
	GUID               string    `json:"guid"`
	Width              int       `json:"width"`
	Height             int       `json:"height"`
	BytesPerPixel      int       `json:"bytesPerPixel"`
	PixelFormat        string    `json:"pixelFormat"`
	Encoding           string    `json:"encoding"`
	RectangleCount     int       `json:"rectangleCount"`
	RequestedMaxWidth  int       `json:"requestedMaxWidth,omitempty"`
	RequestedMaxHeight int       `json:"requestedMaxHeight,omitempty"`
	DataBase64         string    `json:"dataBase64"`
	CapturedAt         time.Time `json:"capturedAt"`
}
