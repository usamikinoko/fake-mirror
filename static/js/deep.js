(function () {
  var CFG = __DEEP_CFG__;
  var KEY = "rh" + ".deep" + ".2026";
  var STORE = "deep.auth";
  var AUTH = location.pathname.indexOf("/auth") === 0;
  var enc = new TextEncoder();
  var dec = new TextDecoder();
  var subtle = window.crypto && window.crypto.subtle;
  var next = function () {
    var n = new URLSearchParams(location.search).get("next");
    return n && n.indexOf("/deep") === 0 ? n : "/deep/";
  };
  var go = function () {
    location.replace(next());
  };
  var redirect = function () {
    location.replace("/auth.html?next=" + encodeURIComponent(location.pathname));
  };
  var unxor = function (h) {
    var o = "", i = 0;
    for (; i < h.length; i += 2) {
      o += String.fromCharCode(parseInt(h.substr(i, 2), 16) ^ KEY.charCodeAt((i / 2) % KEY.length));
    }
    return o;
  };
  var b64u = function (u) {
    var s = "";
    for (var i = 0; i < u.length; i++) s += String.fromCharCode(u[i]);
    return btoa(s).replace(/\+/g, "-").replace(/\//g, "_").replace(/=+$/, "");
  };
  var b64d = function (s) {
    s = s.replace(/-/g, "+").replace(/_/g, "/");
    while (s.length % 4) s += "=";
    var b = atob(s), u = new Uint8Array(b.length);
    for (var i = 0; i < b.length; i++) u[i] = b.charCodeAt(i);
    return u;
  };
  var hexb = function (h) {
    var u = new Uint8Array(h.length / 2);
    for (var i = 0; i < u.length; i++) u[i] = parseInt(h.substr(i * 2, 2), 16);
    return u;
  };
  var cteq = function (a, b) {
    if (a.length !== b.length) return false;
    var d = 0;
    for (var i = 0; i < a.length; i++) d |= a[i] ^ b[i];
    return d === 0;
  };
  var hmac = function (msg) {
    return subtle.importKey("raw", enc.encode(unxor(CFG.k)), { name: "HMAC", hash: "SHA-256" }, false, ["sign"]).then(function (k) {
      return subtle.sign("HMAC", k, msg).then(function (s) {
        return new Uint8Array(s);
      });
    });
  };
  var pbkdf2 = function (pw) {
    return subtle.importKey("raw", enc.encode(pw), "PBKDF2", false, ["deriveBits"]).then(function (k) {
      return subtle.deriveBits({ name: "PBKDF2", salt: enc.encode(unxor(CFG.s)), iterations: CFG.i, hash: "SHA-256" }, k, 256).then(function (b) {
        return new Uint8Array(b);
      });
    });
  };
  var payload = function () {
    var r = new Uint8Array(4);
    crypto.getRandomValues(r);
    var h = "";
    for (var i = 0; i < r.length; i++) h += (r[i] < 16 ? "0" : "") + r[i].toString(16);
    return b64u(enc.encode(JSON.stringify({ exp: Date.now() + CFG.t * 1000, rnd: h })));
  };
  var sign = function () {
    var p = payload();
    return hmac(enc.encode(p)).then(function (s) {
      return "v1." + p + "." + b64u(s);
    });
  };
  var verify = function (tok) {
    var parts = tok.split(".");
    if (parts.length !== 3 || parts[0] !== "v1") return Promise.resolve(false);
    return hmac(enc.encode(parts[1])).then(function (s) {
      if (!cteq(b64d(parts[2]), s)) return false;
      var exp = 0;
      try {
        exp = JSON.parse(dec.decode(b64d(parts[1]))).exp || 0;
      } catch (e) {}
      return exp > Date.now();
    });
  };
  var get = function (u, tok) {
    var hd = {};
    hd[CFG.h] = tok;
    return fetch(u, { headers: hd }).then(function (r) {
      if (!r.ok) throw new Error("fetch");
      return r.text();
    });
  };
  var fail = function (el) {
    el.hidden = false;
    el.textContent = "内容加载失败";
  };
  var loading = function (el) {
    el.hidden = false;
    el.textContent = "加载中…";
  };
  var list = function (el, tok) {
    loading(el);
    get("/deep/data/index.json", tok).then(function (t) {
      var items = JSON.parse(t).items || [];
      el.innerHTML = "";
      if (!items.length) {
        var p = document.createElement("p");
        p.className = "deep-empty";
        p.textContent = "暂无文章";
        el.appendChild(p);
      } else {
        items.forEach(function (it) {
          var a = document.createElement("a");
          a.href = it.url;
          a.textContent = it.title;
          var tm = document.createElement("time");
          tm.textContent = it.date;
          var row = document.createElement("article");
          row.className = "post-list-item";
          row.appendChild(a);
          row.appendChild(tm);
          el.appendChild(row);
        });
      }
      el.hidden = false;
    }, function () {
      fail(el);
    });
  };
  var gate = function () {
    document.addEventListener("DOMContentLoaded", function () {
      var tok = localStorage.getItem(STORE);
      var m = location.pathname.match(/^\/deep\/articles\/([^/]+)\/?$/);
      if (m) {
        var post = document.getElementById("deep-post");
        if (!post) return;
        loading(post);
        get("/deep/data/posts/" + encodeURIComponent(m[1]) + ".html", tok).then(function (h) {
          post.innerHTML = h;
          post.hidden = false;
        }, function () {
          fail(post);
        });
        return;
      }
      var home = document.getElementById("deep-home");
      if (home) {
        loading(home);
        get("/deep/data/home.html", tok).then(function (h) {
          home.innerHTML = h;
          home.hidden = false;
        }, function () {
          fail(home);
        });
      }
      var listEl = document.getElementById("deep-list");
      if (listEl) list(listEl, tok);
    });
  };
  var auth = function () {
    document.addEventListener("DOMContentLoaded", function () {
      var form = document.getElementById("deep-auth-form");
      if (!form) return;
      var input = document.getElementById("deep-auth-input");
      var err = document.getElementById("deep-auth-error");
      var tok = localStorage.getItem(STORE);
      if (tok) {
        verify(tok).then(function (ok) {
          if (ok) go();
        });
        return;
      }
      form.addEventListener("submit", function (e) {
        e.preventDefault();
        var v = (input.value || "").trim();
        if (!v) return;
        pbkdf2(v).then(function (h) {
          if (cteq(h, hexb(unxor(CFG.p)))) {
            sign().then(function (t) {
              localStorage.setItem(STORE, t);
              go();
            });
          } else {
            err.hidden = false;
            input.select();
          }
        }, function () {
          err.textContent = "当前环境不支持安全认证";
          err.hidden = false;
        });
      });
    });
  };
  if (!subtle) {
    if (AUTH) {
      auth();
    } else {
      redirect();
    }
    return;
  }
  if (AUTH) {
    auth();
    return;
  }
  var tok = localStorage.getItem(STORE);
  if (!tok) {
    redirect();
    return;
  }
  verify(tok).then(function (ok) {
    if (ok) {
      gate();
    } else {
      localStorage.removeItem(STORE);
      redirect();
    }
  }, function () {
    redirect();
  });
})();
