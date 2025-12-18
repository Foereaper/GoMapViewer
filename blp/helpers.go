package blp

import "image"

/* =======================
   Pixel helpers
   ======================= */

// set writes an RGBA pixel directly into an image.RGBA.
func set(img *image.RGBA, x, y int, r, g, b, a uint8) {
	i := y*img.Stride + x*4
	img.Pix[i+0] = r
	img.Pix[i+1] = g
	img.Pix[i+2] = b
	img.Pix[i+3] = a
}

// rgb565 converts a 16-bit RGB565 value to 8-bit RGB.
func rgb565(c uint16) (r, g, b uint8) {
	r = uint8((c >> 11) & 0x1F)
	g = uint8((c >> 5) & 0x3F)
	b = uint8(c & 0x1F)

	// Expand to full 8-bit range
	r = (r << 3) | (r >> 2)
	g = (g << 2) | (g >> 4)
	b = (b << 3) | (b >> 2)

	return
}

/* =======================
   DXT palettes
   ======================= */

// colorPalette builds the 4-color palette used by DXT1/DXT5 blocks.
func colorPalette(c0, c1 uint16) [4][3]uint8 {
	r0, g0, b0 := rgb565(c0)
	r1, g1, b1 := rgb565(c1)

	return [4][3]uint8{
		{r0, g0, b0},
		{r1, g1, b1},
		{
			uint8((2*int(r0) + int(r1)) / 3),
			uint8((2*int(g0) + int(g1)) / 3),
			uint8((2*int(b0) + int(b1)) / 3),
		},
		{
			uint8((int(r0) + 2*int(r1)) / 3),
			uint8((int(g0) + 2*int(g1)) / 3),
			uint8((int(b0) + 2*int(b1)) / 3),
		},
	}
}

// alphaPalette builds the 8-entry alpha palette used by DXT5.
func alphaPalette(a0, a1 uint8) [8]uint8 {
	var p [8]uint8
	p[0], p[1] = a0, a1

	if a0 > a1 {
		// 8-step interpolation
		for i := 2; i < 8; i++ {
			p[i] = uint8(((8-i)*int(a0) + (i-1)*int(a1)) / 7)
		}
	} else {
		// 6-step interpolation + endpoints
		for i := 2; i < 6; i++ {
			p[i] = uint8(((6-i)*int(a0) + (i-1)*int(a1)) / 5)
		}
		p[6] = 0
		p[7] = 255
	}

	return p
}

/* =======================
   ARGB8888 decoding
   ======================= */

// decodeARGB decodes raw ARGB8888 pixel data.
func decodeARGB(w, h int, data []byte) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	pixels := w * h
	for i := 0; i < pixels; i++ {
		o := i * 4
		a := data[o+0]
		r := data[o+1]
		g := data[o+2]
		b := data[o+3]

		set(img, i%w, i/w, r, g, b, a)
	}

	return img, nil
}