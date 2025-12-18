package blp

import (
	"encoding/binary"
	"image"
)

func decodeDXT1(w, h int, data []byte) (image.Image, error) {
	bw := (w + 3) / 4
	bh := (h + 3) / 4

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	offset := 0

	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			// Each DXT1 block is 8 bytes
			c0 := binary.LittleEndian.Uint16(data[offset:])
			c1 := binary.LittleEndian.Uint16(data[offset+2:])
			indices := binary.LittleEndian.Uint32(data[offset+4:])
			offset += 8

			colors := colorPalette(c0, c1)

			for py := 0; py < 4; py++ {
				for px := 0; px < 4; px++ {
					x := bx*4 + px
					y := by*4 + py
					if x >= w || y >= h {
						continue
					}

					i := (indices >> uint(2*(py*4+px))) & 0x03
					c := colors[i]
					set(img, x, y, c[0], c[1], c[2], 255)
				}
			}
		}
	}

	return img, nil
}

func decodeDXT5(w, h int, data []byte) (image.Image, error) {
	bw := (w + 3) / 4
	bh := (h + 3) / 4

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	offset := 0

	for by := 0; by < bh; by++ {
		for bx := 0; bx < bw; bx++ {
			// Alpha block (8 bytes)
			a0 := data[offset]
			a1 := data[offset+1]

			var alphaBits uint64
			for i := 0; i < 6; i++ {
				alphaBits |= uint64(data[offset+2+i]) << (8 * i)
			}

			// Color block (8 bytes)
			c0 := binary.LittleEndian.Uint16(data[offset+8:])
			c1 := binary.LittleEndian.Uint16(data[offset+10:])
			indices := binary.LittleEndian.Uint32(data[offset+12:])
			offset += 16

			alpha := alphaPalette(a0, a1)
			colors := colorPalette(c0, c1)

			for py := 0; py < 4; py++ {
				for px := 0; px < 4; px++ {
					x := bx*4 + px
					y := by*4 + py
					if x >= w || y >= h {
						continue
					}

					p := py*4 + px
					a := alpha[(alphaBits>>(3*p))&0x07]
					c := colors[(indices>>(2*p))&0x03]

					set(img, x, y, c[0], c[1], c[2], a)
				}
			}
		}
	}

	return img, nil
}
