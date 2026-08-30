(function () {
  var requestID = 0;

  function isNavigable(link, event) {
    if (!link || link.target && link.target !== "_self" || link.hasAttribute("download")) return false;
    if (event && (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey)) return false;
    var url = new URL(link.href, location.href);
    if (url.origin !== location.origin || url.protocol !== location.protocol) return false;
    return url.pathname !== location.pathname || url.search !== location.search;
  }

  function updatePage(doc) {
    var currentMain = document.querySelector("main");
    var nextMain = doc.querySelector("main");
    if (!currentMain || !nextMain) return false;

    var currentHeader = document.querySelector("header");
    var nextHeader = doc.querySelector("header");
    var currentFooter = document.querySelector("footer");
    var nextFooter = doc.querySelector("footer");
    if (currentHeader && nextHeader) currentHeader.replaceWith(nextHeader);
    currentMain.replaceWith(nextMain);
    if (currentFooter && nextFooter) currentFooter.replaceWith(nextFooter);

    document.title = doc.title;
    document.documentElement.lang = doc.documentElement.lang || "zh-CN";
    window.dispatchEvent(new Event("pagechange"));
    if (window.syncThemeButton) {
      window.syncThemeButton(document.documentElement.getAttribute("data-theme") || "dark");
    }
    return true;
  }

  function visit(url, replace) {
    var id = ++requestID;
    var main = document.querySelector("main");
    if (main) main.setAttribute("aria-busy", "true");
    fetch(url.href, { credentials: "same-origin" }).then(function (response) {
      if (!response.ok) throw new Error("navigation failed");
      return response.text();
    }).then(function (html) {
      if (id !== requestID) return;
      var doc = new DOMParser().parseFromString(html, "text/html");
      if (!updatePage(doc)) throw new Error("invalid page");
      if (replace) history.replaceState(null, "", url.href);
      else history.pushState(null, "", url.href);
      if (url.hash) {
        var target = document.getElementById(decodeURIComponent(url.hash.slice(1)));
        if (target) target.scrollIntoView({ behavior: "auto", block: "start" });
      } else {
        window.scrollTo(0, 0);
      }
    }).catch(function () {
      if (id === requestID) location.href = url.href;
    }).then(function () {
      if (id !== requestID) return;
      var current = document.querySelector("main");
      if (current) current.removeAttribute("aria-busy");
    });
  }

  document.addEventListener("click", function (event) {
    var link = event.target.closest("a");
    if (!isNavigable(link, event)) return;
    event.preventDefault();
    visit(new URL(link.href, location.href), false);
  });

  window.addEventListener("popstate", function () {
    visit(new URL(location.href), true);
  });
})();
