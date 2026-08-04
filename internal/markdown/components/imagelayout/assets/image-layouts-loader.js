// image-layouts-loader.js
// 把 image-layout 网格/瀑布/自定义布局里的「远程图」加载与页面其它功能解耦：
//
//   - 远程 <img> 在构建期输出为 data-src（无 src），浏览器不会在 innerHTML 注入时
//     并发拉取，避免解码爆发抢占主线程、卡住滚动与雨点动效。
//   - IntersectionObserver 只在图片进入视口附近（rootMargin 300px）时才入队加载。
//   - 全局并发上限 2 + requestIdleCallback 错峰，确保任意时刻主线程最多处理少量
//     解码任务，滚动/动画始终跟手。
//   - MutationObserver 兜底：deep-post 等异步注入的 HTML 也能被接管。
//   - 加载完成后移除 data-src，触发 CSS opacity 淡入；本地图不受影响（无 data-src）。
//   - 无 IO 支持时降级为直接加载，保证可访问性。
(function () {
  var SELECTOR = ".image-layouts-image-cell img[data-src]";
  var ROOT_MARGIN = "300px 0px";
  var MAX_CONCURRENT = 2;
  var STAGGER = 30; // 每张入队后的最小错峰间隔(ms)，进一步分散解码

  var queue = [];
  var active = 0;
  var io = null;
  var scanTimer = null;

  function finishOne() {
    active--;
    pump();
  }

  function loadImage(img) {
    var url = img.getAttribute("data-src");
    if (!url) {
      finishOne();
      return;
    }
    // 先挂事件再赋 src，避免缓存命中时事件丢失
    img.addEventListener("load", function onLoad() {
      img.removeEventListener("load", onLoad);
      img.removeAttribute("data-src"); // 触发 CSS 淡入
      finishOne();
    });
    img.addEventListener("error", function onError() {
      img.removeEventListener("error", onError);
      img.removeAttribute("data-src"); // 仍淡入（显示 broken），不卡队列
      finishOne();
    });
    img.src = url;
  }

  function pump() {
    while (active < MAX_CONCURRENT && queue.length > 0) {
      var img = queue.shift();
      active++;
      if (window.requestIdleCallback) {
        // requestIdleCallback 把解码挪到空闲时段，避开动画帧
        (function (el) {
          requestIdleCallback(function () { loadImage(el); }, { timeout: 200 });
        })(img);
      } else {
        (function (el) { setTimeout(function () { loadImage(el); }, STAGGER); })(img);
      }
    }
  }

  function onIntersect(entries) {
    for (var i = 0; i < entries.length; i++) {
      var e = entries[i];
      if (!e.isIntersecting) continue;
      var img = e.target;
      io.unobserve(img);
      queue.push(img);
    }
    pump();
  }

  function ensureIO() {
    if (io || typeof IntersectionObserver === "undefined") return io;
    io = new IntersectionObserver(onIntersect, { rootMargin: ROOT_MARGIN });
    return io;
  }

  function scan() {
    var imgs = document.querySelectorAll(SELECTOR);
    if (imgs.length === 0) return;
    if (!ensureIO()) {
      // 无 IO 支持：直接全部加载（保底可访问性）
      imgs.forEach(function (img) {
        if (img.getAttribute("data-src")) {
          queue.push(img);
        }
      });
      pump();
      return;
    }
    imgs.forEach(function (img) { io.observe(img); });
  }

  function scheduleScan() {
    if (scanTimer) return;
    scanTimer = setTimeout(function () {
      scanTimer = null;
      scan();
    }, 30);
  }

  function init() {
    if (document.readyState !== "loading") {
      scheduleScan();
    } else {
      document.addEventListener("DOMContentLoaded", scheduleScan);
    }
    // 监听异步注入的 image-layout 块（如 deep-post 的 innerHTML 赋值）
    if (typeof MutationObserver !== "undefined") {
      var mo = new MutationObserver(function (muts) {
        for (var i = 0; i < muts.length; i++) {
          if (muts[i].addedNodes && muts[i].addedNodes.length > 0) {
            scheduleScan();
            return;
          }
        }
      });
      mo.observe(document.documentElement, { childList: true, subtree: true });
    }
  }

  init();
})();
