package blp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"image"
	"io/fs"
	"os"
)

var (
	ErrBadBLP         = errors.New("blp: bad file")
	ErrUnsupportedBLP = errors.New("blp: unsupported format")
)

// BLP2 header layout (minimum size = 1172 bytes)
type blpHeader struct {
	Magic         [4]byte
	Version       uint32
	ColorEncoding uint8
	AlphaDepth    uint8
	Format        uint8
	Mips          uint8
	Width         uint32
	Height        uint32
	Offsets       [16]uint32
	Sizes         [16]uint32
	Palette       [256]uint32
}

/* =======================
   Public API
   ======================= */

// DecodeBLP loads and decodes a BLP file from disk.
func DecodeBLP(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodeBLP(data, path)
}

// DecodeBLPFromFS loads and decodes a BLP file from an fs.FS.
func DecodeBLPFromFS(fsys fs.FS, path string) (image.Image, error) {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return nil, err
	}
	return decodeBLP(data, path)
}

// DecodeBLPFromBytes decodes a BLP file from raw bytes.
func DecodeBLPFromBytes(data []byte) (image.Image, error) {
	return decodeBLP(data, "")
}

/* =======================
   Core decoder
   ======================= */

func decodeBLP(data []byte, path string) (image.Image, error) {
	h, err := parseHeader(data)
	if err != nil {
		return nil, wrapErr(err, path)
	}

	off := int(h.Offsets[0])
	sz := int(h.Sizes[0])

	if off <= 0 || sz <= 0 || off+sz > len(data) {
		return nil, wrapErr(ErrBadBLP, path)
	}

	mip := data[off : off+sz]

	switch h.ColorEncoding {
	case 2: // DXTC
		if h.Format == 7 {
			return decodeDXT5(int(h.Width), int(h.Height), mip)
		}
		return decodeDXT1(int(h.Width), int(h.Height), mip)

	case 3, 4: // ARGB
		return decodeARGB(int(h.Width), int(h.Height), mip)

	default:
		return nil, wrapErr(ErrUnsupportedBLP, path)
	}
}

/* =======================
   Header parsing
   ======================= */

func parseHeader(b []byte) (*blpHeader, error) {
	// BLP2 header size:
	// 4   magic
	// 16  fixed fields
	// 64  offsets
	// 64  sizes
	// 1024 palette
	// ----
	// 1172 bytes
	if len(b) < 1172 {
		return nil, fmt.Errorf(
			"%w: file too small (%d bytes)",
			ErrBadBLP, len(b),
		)
	}

	h := &blpHeader{}

	copy(h.Magic[:], b[0:4])
	if string(h.Magic[:]) != "BLP2" {
		return nil, fmt.Errorf(
			"%w: bad magic %q",
			ErrBadBLP, h.Magic,
		)
	}

	h.Version = binary.LittleEndian.Uint32(b[4:8])
	h.ColorEncoding = b[8]
	h.AlphaDepth = b[9]
	h.Format = b[10]
	h.Mips = b[11]
	h.Width = binary.LittleEndian.Uint32(b[12:16])
	h.Height = binary.LittleEndian.Uint32(b[16:20])

	o := 20
	for i := 0; i < 16; i++ {
		h.Offsets[i] = binary.LittleEndian.Uint32(b[o:])
		o += 4
	}
	for i := 0; i < 16; i++ {
		h.Sizes[i] = binary.LittleEndian.Uint32(b[o:])
		o += 4
	}
	for i := 0; i < 256; i++ {
		h.Palette[i] = binary.LittleEndian.Uint32(b[o:])
		o += 4
	}

	return h, nil
}

/* =======================
   Utilities
   ======================= */

func wrapErr(err error, path string) error {
	if path == "" {
		return err
	}
	return fmt.Errorf("%w: %s", err, path)
}
