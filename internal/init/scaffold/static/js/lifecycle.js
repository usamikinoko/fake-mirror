(function () {
  var initializers = [];
  var ready = false;

  function run() {
    for (var i = 0; i < initializers.length; i++) {
      try {
        initializers[i]();
      } catch (error) {
        console.error("[rainhush] page initializer failed", error);
      }
    }
  }

  window.registerPageInit = function (init) {
    if (typeof init !== "function") throw new TypeError("page initializer must be a function");
    initializers.push(init);
    if (ready) run();
  };
  window.runPageInitializers = run;

  document.addEventListener("DOMContentLoaded", function () {
    ready = true;
    run();
  });
  window.addEventListener("pagechange", run);
})();
