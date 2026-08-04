document.querySelectorAll(".my-box").forEach((el) => {
  el.addEventListener("click", () => {
    el.style.opacity = el.style.opacity === "0.6" ? "1" : "0.6";
  });
});
