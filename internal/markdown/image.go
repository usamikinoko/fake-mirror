package markdown

import (
	"fmt"

	"rainhush/internal/imagedim"

	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/util"
)

type imageRenderer struct{}

func (r *imageRenderer) RegisterFuncs(reg renderer.NodeRendererFuncRegisterer) {
	reg.Register(ast.KindImage, r.renderImage)
}

// renderImage 输出 <img>：
//   - 本地图片构建期探测固有尺寸并输出 width/height，浏览器加载前即按比例占位，
//     弱网/慢加载下不产生布局抖动（与 image-layout 组件行为一致）；
//   - 所有正文图片追加 loading="lazy" decoding="async"，避免图片与首屏关键资源抢带宽。
func (r *imageRenderer) renderImage(w util.BufWriter, source []byte, node ast.Node, entering bool) (ast.WalkStatus, error) {
	if !entering {
		return ast.WalkContinue, nil
	}
	n := node.(*ast.Image)
	w.WriteString(`<img src="`)
	w.Write(util.EscapeHTML(n.Destination))
	w.WriteString(`" alt="`)
	w.Write(util.EscapeHTML(n.Text(source)))
	w.WriteString(`"`)
	if len(n.Title) > 0 {
		w.WriteString(` title="`)
		w.Write(util.EscapeHTML(n.Title))
		w.WriteString(`"`)
	}
	if width, height, ok := localImageSize(string(n.Destination)); ok {
		fmt.Fprintf(w, ` width="%d" height="%d"`, width, height)
	}
	w.WriteString(` loading="lazy" decoding="async">`)
	return ast.WalkSkipChildren, nil
}

// localImageSize 将图片地址映射到本地 static/ 路径并探测固有尺寸；远程图返回 ok=false。
func localImageSize(dest string) (int, int, bool) {
	rel, ok := imagedim.LocalPath(dest)
	if !ok {
		return 0, 0, false
	}
	return imagedim.Size(rel)
}
