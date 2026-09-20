(function () {
  "use strict";

  var matter = document.getElementById("matter");
  if (!matter) return;

  var pages = Array.prototype.slice.call(matter.children);
  var total = pages.length;
  if (total === 0) return;

  var body = document.body;

  var home = body.getAttribute("data-home") || "";
  var brand = body.getAttribute("data-short") || body.getAttribute("data-book") || "Book";
  var otherLabel = body.getAttribute("data-other-label") || "";
  var otherUrl = body.getAttribute("data-other-url") || "";

  var bar = document.createElement("div");
  bar.id = "bookbar";

  var parts = [];
  parts.push(
    '<div class="brand"><img src="' + home + 'snowflake.svg" alt="Snow">' +
      "<span>" + brand + "</span></div>"
  );
  parts.push('<div class="spacer"></div>');
  if (otherLabel && otherUrl) {
    parts.push('<a class="swap" href="' + otherUrl + '">' + otherLabel + "</a>");
  }
  parts.push(
    '<button id="prev" type="button" aria-label="Prev page">&#8249;</button>'
  );
  parts.push('<span class="count" id="count"></span>');
  parts.push(
    '<button id="next" type="button" aria-label="Next page">&#8250;</button>'
  );
  bar.innerHTML = parts.join("");
  body.appendChild(bar);

  var prevBtn = document.getElementById("prev");
  var nextBtn = document.getElementById("next");
  var counter = document.getElementById("count");

  function pageWidth() {
    return matter.clientWidth;
  }

  function currentIndex() {
    if (pageWidth() === 0) return 0;
    return Math.round(matter.scrollLeft / pageWidth());
  }

  function go(index) {
    index = Math.max(0, Math.min(total - 1, index));
    matter.scrollTo({ left: index * pageWidth(), behavior: "smooth" });
    update();
  }

  function update() {
    var i = currentIndex();
    counter.textContent = (i + 1) + " / " + total;
    prevBtn.disabled = i <= 0;
    nextBtn.disabled = i >= total - 1;
  }

  prevBtn.addEventListener("click", function () {
    go(currentIndex() - 1);
  });
  nextBtn.addEventListener("click", function () {
    go(currentIndex() + 1);
  });

  var scrollTimer = null;
  matter.addEventListener("scroll", function () {
    if (scrollTimer) return;
    scrollTimer = setTimeout(function () {
      scrollTimer = null;
      update();
    }, 60);
  });

  document.addEventListener("keydown", function (event) {
    var tag = event.target;
    if (
      tag &&
      (tag.tagName === "INPUT" ||
        tag.tagName === "TEXTAREA" ||
        tag.isContentEditable)
    ) {
      return;
    }
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      go(currentIndex() - 1);
    } else if (event.key === "ArrowRight") {
      event.preventDefault();
      go(currentIndex() + 1);
    } else if (event.key === "Home") {
      event.preventDefault();
      go(0);
    } else if (event.key === "End") {
      event.preventDefault();
      go(total - 1);
    }
  });

  var tocLinks = matter.querySelectorAll("a[href^='#']");
  Array.prototype.forEach.call(tocLinks, function (link) {
    link.addEventListener("click", function (event) {
      var id = link.getAttribute("href").slice(1);
      if (!id) return;
      var target = document.getElementById(id);
      if (!target) return;
      event.preventDefault();
      for (var i = 0; i < pages.length; i++) {
        if (pages[i].contains(target)) {
          go(i);
          return;
        }
      }
    });
  });

  var resizeTimer = null;
  window.addEventListener("resize", function () {
    if (resizeTimer) return;
    resizeTimer = setTimeout(function () {
      resizeTimer = null;
      matter.scrollLeft = currentIndex() * pageWidth();
      update();
    }, 150);
  });

  update();
})();