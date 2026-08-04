document.querySelectorAll(".image-layout-carousel").forEach((root) => {
  const slides = Array.from(root.querySelectorAll(".slide"));
  if (slides.length === 0) return;
  const thumbs = root.dataset.thumbnails === "true";
  let current = 0;

  const pills = document.createElement("div");
  pills.className = "carousel-pills";
  const thumbStrip = document.createElement("div");
  thumbStrip.className = "carousel-thumbnails";

  slides.forEach((_, i) => {
    if (thumbs) {
      const img = document.createElement("img");
      img.src = slides[i].querySelector("img").src;
      img.alt = `Thumbnail ${i + 1}`;
      img.loading = "lazy";
      img.decoding = "async";
      img.addEventListener("click", () => go(i));
      thumbStrip.appendChild(img);
    } else {
      const b = document.createElement("button");
      b.type = "button";
      b.setAttribute("aria-label", `第 ${i + 1} 张`);
      b.addEventListener("click", () => go(i));
      pills.appendChild(b);
    }
  });

  if (thumbs) root.appendChild(thumbStrip); else root.appendChild(pills);

  function go(i) {
    slides[current].classList.remove("active");
    if (thumbs) thumbStrip.children[current]?.classList.remove("active");
    else pills.children[current]?.classList.remove("active");
    current = (i + slides.length) % slides.length;
    slides[current].classList.add("active");
    if (thumbs) thumbStrip.children[current]?.classList.add("active");
    else pills.children[current]?.classList.add("active");
    if (thumbs) thumbStrip.children[current]?.scrollIntoView({ behavior: "smooth", block: "nearest", inline: "center" });
  }

  root.querySelector(".nav-button.prev").addEventListener("click", () => go(current - 1));
  root.querySelector(".nav-button.next").addEventListener("click", () => go(current + 1));
  root.tabIndex = 0;
  root.addEventListener("keydown", (e) => {
    if (e.key === "ArrowLeft") go(current - 1);
    if (e.key === "ArrowRight") go(current + 1);
  });

  if (thumbs) thumbStrip.children[0]?.classList.add("active");
  else pills.children[0]?.classList.add("active");
});
