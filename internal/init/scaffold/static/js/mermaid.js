(function () {
  var initialized = false;
  var loading = false;

  function run() {
    if (!document.querySelector("pre.mermaid")) return;
    if (typeof mermaid === "undefined") {
      if (loading) return;
      loading = true;
      var script = document.createElement("script");
      script.src = "https://cdn.jsdelivr.net/npm/mermaid@10/dist/mermaid.min.js";
      script.async = true;
      script.onload = function () { loading = false; run(); };
      script.onerror = function () { loading = false; script.remove(); };
      document.head.appendChild(script);
      return;
    }
    if (!initialized) {
      mermaid.initialize({
        startOnLoad: false,
        theme: "default",
        securityLevel: "antiscript",
      });
      initialized = true;
    }
    mermaid.run({ querySelector: "pre.mermaid" });
  }

  window.initMermaidDiagrams = run;
  if (window.registerPageInit) window.registerPageInit(run);
  else document.addEventListener("DOMContentLoaded", run);
})();
