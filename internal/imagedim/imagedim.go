// Package imagedim 探测本地图片文件的固有尺寸（像素）。
// 供 markdown 图片渲染与 image-layout 组件共享，避免两处重复实现。
package imagedim

import (
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Size 读取本地图片文件头探测固有尺寸（像素）。仅读取前 4KB，开销可忽略；
// 解析失败（格式不支持 / 文件缺失）返回 ok=false。
func Size(path string) (w, h int, ok bool) {
	f, err := os.Open(filepath.FromSlash(path))
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()

	head := make([]byte, 4096)
	n, err := io.ReadFull(f, head)
	if err != nil && err != io.ErrUnexpectedEOF && err != io.EOF {
		return 0, 0, false
	}
	return dimsFromHead(head[:n])
}

// LocalPath 把图片地址解析为项目内 static/images/ 目录下的本地路径：
//
//	/images/x.png  -> static/images/x.png（站点图片根约定）
//	images/x.png   -> static/images/x.png
//	static/images/x.png -> static/images/x.png
//
// 超出 static/images/ 的路径（含目录穿越/符号链接逃逸）、远程 URL 与 data URI
// 一律返回 ok=false。
func LocalPath(dest string) (string, bool) {
	var path string
	switch {
	case strings.HasPrefix(dest, "/images/"):
		path = "static/images/" + strings.TrimPrefix(dest, "/images/")
	case strings.HasPrefix(dest, "images/"):
		path = "static/" + dest
	case strings.HasPrefix(dest, "static/"):
		path = dest
	default:
		return "", false
	}

	root, err := filepath.Abs("static/images")
	if err != nil {
		return "", false
	}
	candidate, err := filepath.Abs(filepath.FromSlash(path))
	if err != nil {
		return "", false
	}
	rel, err := filepath.Rel(root, candidate)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	if real, err := filepath.EvalSymlinks(candidate); err == nil {
		rootReal, rootErr := filepath.EvalSymlinks(root)
		if rootErr != nil {
			return "", false
		}
		realRel, relErr := filepath.Rel(rootReal, real)
		if relErr != nil || realRel == ".." || strings.HasPrefix(realRel, ".."+string(filepath.Separator)) {
			return "", false
		}
	}
	return path, true
}

func dimsFromHead(b []byte) (w, h int, ok bool) {
	switch {
	case len(b) >= 24 && b[0] == 0x89 && b[1] == 'P' && b[2] == 'N' && b[3] == 'G':
		return int(be32(b[16:20])), int(be32(b[20:24])), true
	case len(b) >= 10 && string(b[0:4]) == "GIF8":
		return int(le16(b[6:8])), int(le16(b[8:10])), true
	case len(b) >= 2 && b[0] == 0xFF && b[1] == 0xD8:
		return jpegDims(b)
	case len(b) >= 12 && string(b[0:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return webpDims(b)
	case len(b) >= 26 && string(b[0:2]) == "BM":
		return int(le32(b[18:22])), int(absLe32(b[22:26])), true
	case strings.HasPrefix(strings.ToLower(string(b)), "<svg") ||
		strings.Contains(string(b[:min(len(b), 512)]), "<svg"):
		return svgDims(b)
	}
	return 0, 0, false
}

func jpegDims(b []byte) (w, h int, ok bool) {
	i := 2
	for i+3 < len(b) {
		if b[i] != 0xFF {
			i++
			continue
		}
		marker := b[i+1]
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			i += 2
			continue
		}
		segLen := int(be16(b[i+2 : i+4]))
		if isSOF(marker) && i+7 < len(b) {
			return int(be16(b[i+7 : i+9])), int(be16(b[i+5 : i+7])), true
		}
		i += 2 + segLen
	}
	return 0, 0, false
}

func isSOF(marker byte) bool {
	return (marker >= 0xC0 && marker <= 0xC3) ||
		(marker >= 0xC5 && marker <= 0xC7) ||
		(marker >= 0xC9 && marker <= 0xCB) ||
		(marker >= 0xCD && marker <= 0xCF)
}

func webpDims(b []byte) (w, h int, ok bool) {
	switch string(b[12:16]) {
	case "VP8 ":
		if len(b) < 30 {
			return 0, 0, false
		}
		return int(le16(b[26:28]) & 0x3FFF), int(le16(b[28:30]) & 0x3FFF), true
	case "VP8L":
		if len(b) < 25 {
			return 0, 0, false
		}
		bits := le32(b[21:25])
		return int(bits&0x3FFF) + 1, int((bits>>14)&0x3FFF) + 1, true
	case "VP8X":
		if len(b) < 30 {
			return 0, 0, false
		}
		return int(le24(b[24:27])) + 1, int(le24(b[27:30])) + 1, true
	}
	return 0, 0, false
}

var (
	svgSizeRe    = regexp.MustCompile(`(?i)(?:width|height)\s*=\s*"([0-9.]+)(?:px)?"`)
	svgViewboxRe = regexp.MustCompile(`(?i)viewBox\s*=\s*"[-\s0-9.]+"`)
)

func svgDims(b []byte) (w, h int, ok bool) {
	s := string(b)
	matches := svgSizeRe.FindAllStringSubmatch(s, 2)
	if len(matches) == 2 {
		sw, errW := strconv.Atoi(matches[0][1])
		sh, errH := strconv.Atoi(matches[1][1])
		if errW == nil && errH == nil && sw > 0 && sh > 0 {
			return sw, sh, true
		}
	}
	if m := svgViewboxRe.FindString(s); m != "" {
		start := strings.IndexByte(m, '"')
		if start != -1 {
			fields := strings.Fields(m[start+1 : len(m)-1])
			if len(fields) >= 4 {
				if vw, err1 := strconv.Atoi(fields[2]); err1 == nil && vw > 0 {
					if vh, err2 := strconv.Atoi(fields[3]); err2 == nil && vh > 0 {
						return vw, vh, true
					}
				}
			}
		}
	}
	return 0, 0, false
}

func be32(b []byte) uint32 { return binary.BigEndian.Uint32(b) }
func be16(b []byte) uint16 { return binary.BigEndian.Uint16(b) }
func le16(b []byte) uint16 { return binary.LittleEndian.Uint16(b) }
func le32(b []byte) uint32 { return binary.LittleEndian.Uint32(b) }
func le24(b []byte) uint32 { return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 }
func absLe32(b []byte) uint32 {
	v := le32(b)
	if v&0x80000000 != 0 {
		return ^v + 1
	}
	return v
}
