package imagelayout

import (
	"strings"

	"rainhush/internal/imagedim"
)

// imageDimensions 解析图片地址并探测本地图片固有尺寸（像素）。
// 远程图、data URI 与探测失败（格式不支持 / 文件缺失）返回 ok=false，
// 渲染时不输出 width/height。
func imageDimensions(link string) (w, h int, ok bool) {
	if strings.HasPrefix(link, "data:") {
		return 0, 0, false
	}
	rel, ok := imagedim.LocalPath(link)
	if !ok {
		return 0, 0, false
	}
	return imagedim.Size(rel)
}
