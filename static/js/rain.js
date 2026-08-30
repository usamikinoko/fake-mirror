(function () {
  var canvas = document.getElementById("rain-canvas");
  if (!canvas) return;

  var ctx = canvas.getContext("2d");
  var drops = [];
  var cw = 0;
  var ch = 0;
  var frameId = 0;
  var color = "";
  var reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  var saveData = navigator.connection && navigator.connection.saveData;
  var WIND = 0;
  var LEN_MIN = 8;
  var LEN_RANGE = 18;
  var SPD_MIN = 8;
  var SPD_RANGE = 12;
  var OPA_MIN = 0.10;
  var OPA_RANGE = 0.16;

  function getDropCount() {
    if (window.innerWidth < 768) return 0;
    if (window.innerWidth < 960) return 96;
    return 140;
  }

  function shouldAnimate() {
    return !reducedMotion && !saveData && !document.hidden && window.innerWidth >= 768 && getDropCount() > 0;
  }

  function resize() {
    var r = canvas.getBoundingClientRect();
    var w = Math.floor(r.width) || window.innerWidth;
    var h = Math.floor(r.height) || window.innerHeight;
    if (w !== cw || h !== ch) {
      canvas.width = w;
      canvas.height = h;
      cw = w;
      ch = h;
      return true;
    }
    return false;
  }

  function setVisibility() {
    var show = window.innerWidth >= 768;
    canvas.style.display = show ? "block" : "none";
  }

  function createDrop(x) {
    return {
      x: x !== undefined ? x : Math.random() * cw,
      y: Math.random() * ch - ch,
      length: LEN_MIN + Math.random() * LEN_RANGE,
      speed: SPD_MIN + Math.random() * SPD_RANGE,
      opacity: OPA_MIN + Math.random() * OPA_RANGE,
    };
  }

  function resetDrop(d) {
    d.x = Math.random() * cw;
    d.y = -d.length * 2 - Math.random() * ch * 0.3;
    d.length = LEN_MIN + Math.random() * LEN_RANGE;
    d.speed = SPD_MIN + Math.random() * SPD_RANGE;
    d.opacity = OPA_MIN + Math.random() * OPA_RANGE;
  }

  function getColor() {
    var style = getComputedStyle(document.documentElement);
    var raw = style.getPropertyValue("--rain-color").trim();
    if (raw) return raw;
    var t = document.documentElement.getAttribute("data-theme");
    return t === "light" ? "40,55,90" : "210,225,245";
  }

  function draw() {
    if (!color) color = getColor();
    ctx.clearRect(0, 0, cw, ch);

    for (var i = 0; i < drops.length; i++) {
      var d = drops[i];
      var dx = d.speed * WIND;

      ctx.beginPath();
      ctx.moveTo(d.x, d.y);
      ctx.lineTo(d.x + dx * 0.6, d.y + d.length);
      ctx.strokeStyle = "rgba(" + color + "," + d.opacity + ")";
      ctx.lineWidth = 0.7 + d.length / 30;
      ctx.lineCap = "round";
      ctx.stroke();

      d.x += dx;
      d.y += d.speed;

      if (d.y > ch + d.length || d.x < -20 || d.x > cw + 20) {
        resetDrop(d);
      }
    }
  }

  function rebuildDrops() {
    drops = [];
    var count = getDropCount();
    for (var i = 0; i < count; i++) {
      drops.push(createDrop());
    }
  }

  function loop() {
    frameId = 0;
    if (!shouldAnimate()) {
      ctx.clearRect(0, 0, cw, ch);
      return;
    }
    if (resize()) {
      rebuildDrops();
    }
    draw();
    schedule();
  }

  function schedule() {
    if (!frameId) {
      frameId = requestAnimationFrame(loop);
    }
  }

  function init() {
    setVisibility();
    resize();
    if (getDropCount() > 0) {
      rebuildDrops();
    }
  }

  function sync() {
    setVisibility();
    if (shouldAnimate()) {
      init();
      schedule();
      return;
    }

    if (frameId) {
      cancelAnimationFrame(frameId);
      frameId = 0;
    }
    ctx.clearRect(0, 0, cw, ch);
  }

  window.addEventListener("resize", sync);
  document.addEventListener("visibilitychange", sync);
  window.addEventListener("themechange", function () { color = getColor(); });
  sync();
})();
