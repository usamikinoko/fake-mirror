package imagedim

import (
	"os"
	"testing"
)

func TestDimsFromHead(t *testing.T) {
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0x0D, 'I', 'H', 'D', 'R'}, make([]byte, 8)...)
	png[16], png[17], png[18], png[19] = 0, 0, 3, 0x20 // 800
	png[20], png[21], png[22], png[23] = 0, 0, 2, 0x58 // 600
	if w, h, ok := dimsFromHead(png); !ok || w != 800 || h != 600 {
		t.Errorf("png = %dx%d ok=%v, want 800x600", w, h, ok)
	}

	gif := []byte("GIF89a" + string([]byte{0x40, 0x01, 0xF0, 0x00})) // 320x240 LE
	if w, h, ok := dimsFromHead(gif); !ok || w != 320 || h != 240 {
		t.Errorf("gif = %dx%d ok=%v, want 320x240", w, h, ok)
	}

	jpg := make([]byte, 20)
	jpg[0], jpg[1] = 0xFF, 0xD8
	jpg[2], jpg[3] = 0xFF, 0xC0 // SOF0
	jpg[4], jpg[5] = 0x00, 0x11
	jpg[7], jpg[8] = 0x02, 0xD0  // 720
	jpg[9], jpg[10] = 0x05, 0x00 // 1280
	if w, h, ok := dimsFromHead(jpg); !ok || w != 1280 || h != 720 {
		t.Errorf("jpeg = %dx%d ok=%v, want 1280x720", w, h, ok)
	}

	webpLossy := make([]byte, 30)
	copy(webpLossy, "RIFF\x00\x00\x00\x00WEBPVP8 ")
	webpLossy[26], webpLossy[27] = 100, 0 // w=100
	webpLossy[28], webpLossy[29] = 50, 0  // h=50
	if w, h, ok := dimsFromHead(webpLossy); !ok || w != 100 || h != 50 {
		t.Errorf("webp lossy = %dx%d ok=%v, want 100x50", w, h, ok)
	}

	webpLossless := make([]byte, 25)
	copy(webpLossless, "RIFF\x00\x00\x00\x00WEBPVP8L")
	bits := uint32(89) | uint32(69)<<14 // w=90, h=70
	webpLossless[21] = byte(bits)
	webpLossless[22] = byte(bits >> 8)
	webpLossless[23] = byte(bits >> 16)
	webpLossless[24] = byte(bits >> 24)
	if w, h, ok := dimsFromHead(webpLossless); !ok || w != 90 || h != 70 {
		t.Errorf("webp lossless = %dx%d ok=%v, want 90x70", w, h, ok)
	}

	bmp := make([]byte, 26)
	copy(bmp, "BM")
	bmp[18], bmp[19], bmp[20], bmp[21] = 200, 0, 0, 0 // w=200 LE
	bmp[22], bmp[23], bmp[24], bmp[25] = 100, 0, 0, 0 // h=100
	if w, h, ok := dimsFromHead(bmp); !ok || w != 200 || h != 100 {
		t.Errorf("bmp = %dx%d ok=%v, want 200x100", w, h, ok)
	}

	svg := []byte(`<svg xmlns="http://www.w3.org/2000/svg" width="500" height="300"><rect/></svg>`)
	if w, h, ok := dimsFromHead(svg); !ok || w != 500 || h != 300 {
		t.Errorf("svg = %dx%d ok=%v, want 500x300", w, h, ok)
	}

	svgViewBox := []byte(`<svg viewBox="0 0 400 250"><rect/></svg>`)
	if w, h, ok := dimsFromHead(svgViewBox); !ok || w != 400 || h != 250 {
		t.Errorf("svg viewBox = %dx%d ok=%v, want 400x250", w, h, ok)
	}

	if _, _, ok := dimsFromHead([]byte("not an image")); ok {
		t.Error("garbage accepted")
	}
}

func TestSizeFromFile(t *testing.T) {
	dir := t.TempDir()
	p := dir + "/test.png"
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0D, 0x0A, 0x1A, 0x0A, 0, 0, 0, 0x0D, 'I', 'H', 'D', 'R'}, make([]byte, 8)...)
	png[16], png[17], png[18], png[19] = 0, 0, 0x01, 0x2C // 300
	png[20], png[21], png[22], png[23] = 0, 0, 0x00, 0x78 // 120
	if err := os.WriteFile(p, png, 0644); err != nil {
		t.Fatal(err)
	}
	if w, h, ok := Size(p); !ok || w != 300 || h != 120 {
		t.Errorf("Size = %dx%d ok=%v, want 300x120", w, h, ok)
	}
	if _, _, ok := Size(dir + "/missing.png"); ok {
		t.Error("missing file should not resolve")
	}
}

func TestLocalPath(t *testing.T) {
	cases := []struct {
		in   string
		want string
		ok   bool
	}{
		{"/images/a.png", "static/images/a.png", true},
		{"images/a.png", "static/images/a.png", true},
		{"static/images/b.png", "static/images/b.png", true},
		{"https://example.com/a.png", "", false},
		{"data:image/svg+xml,abc", "", false},
	}
	for _, c := range cases {
		got, ok := LocalPath(c.in)
		if ok != c.ok || got != c.want {
			t.Errorf("LocalPath(%q) = (%q, %v), want (%q, %v)", c.in, got, ok, c.want, c.ok)
		}
	}
}
