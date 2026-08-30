(function () {
  var observer = null;

  function C() {
    if (observer) {
      observer.disconnect();
      observer = null;
    }
  }

  function R() {
    var b = document.querySelector(".post-body");
    var t = document.getElementById("post-toc");
    if (!b || !t) return;
    C();
    var h = b.querySelectorAll("h2,h3");
    if (h.length < 2) {
      t.innerHTML = "";
      t.style.display = "none";
      return;
    }
    t.innerHTML = "";
    var n = document.createElement("nav");
    n.className = "toc-nav";
    var l = document.createElement("ol");
    l.className = "toc-list";
    for (var i = 0; i < h.length; i++) {
      var e = h[i];
      var id = "toc-" + i;
      e.id = id;
      var li = document.createElement("li");
      li.className = "toc-item toc-item--" + e.tagName.toLowerCase();
      var a = document.createElement("a");
      a.className = "toc-link";
      a.href = "#" + id;
      a.textContent = e.textContent.trim();
      a.addEventListener("click", function (v) {
        v.preventDefault();
        var d = document.getElementById(this.getAttribute("href").slice(1));
        if (d) d.scrollIntoView({
          behavior: window.matchMedia("(prefers-reduced-motion: reduce)").matches ? "auto" : "smooth",
          block: "start",
        });
      });
      li.appendChild(a);
      l.appendChild(li);
    }
    n.appendChild(l);
    t.appendChild(n);
    var ls = t.querySelectorAll(".toc-link");
    observer = new IntersectionObserver(
      function (es) {
        for (var j = 0; j < es.length; j++) {
          var en = es[j];
          var idx = parseInt(en.target.id.replace("toc-", ""), 10);
          if (en.isIntersecting) {
            for (var k = 0; k < ls.length; k++)
              ls[k].classList.remove("toc-link--active");
            if (ls[idx]) ls[idx].classList.add("toc-link--active");
          }
        }
      },
      { rootMargin: "-80px 0px -70% 0px" },
    );
    for (var m = 0; m < h.length; m++) observer.observe(h[m]);
    t.style.display = "block";
  }
  window.initPostTOC = R;
  document.addEventListener("DOMContentLoaded", R);
})();
