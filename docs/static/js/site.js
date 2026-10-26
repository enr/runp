// Copy-to-clipboard for the hero command box and every code block.
(function () {
  "use strict";

  function flash(btn, label) {
    var prev = btn.textContent;
    btn.textContent = label;
    btn.classList.add("copied");
    setTimeout(function () {
      btn.textContent = prev;
      btn.classList.remove("copied");
    }, 1400);
  }

  function copy(text, btn) {
    if (navigator.clipboard && navigator.clipboard.writeText) {
      navigator.clipboard.writeText(text).then(
        function () { flash(btn, "copied ✓"); },
        function () { flash(btn, "error"); }
      );
    } else {
      var ta = document.createElement("textarea");
      ta.value = text;
      ta.style.position = "fixed";
      ta.style.opacity = "0";
      document.body.appendChild(ta);
      ta.select();
      try { document.execCommand("copy"); flash(btn, "copied ✓"); }
      catch (e) { flash(btn, "error"); }
      document.body.removeChild(ta);
    }
  }

  // Hero command box: button is already in the markup.
  document.querySelectorAll(".cmd[data-copy]").forEach(function (box) {
    var btn = box.querySelector(".cmd-copy");
    var text = box.querySelector(".cmd-text");
    if (btn && text) {
      btn.addEventListener("click", function () {
        copy(text.textContent.trim(), btn);
      });
    }
  });

  // Docs code blocks: inject a copy button into each block.
  document.querySelectorAll(".content pre, .content .highlight").forEach(function (block) {
    // Avoid double-injecting when a <pre> is nested inside a .highlight.
    if (block.tagName === "PRE" && block.closest(".highlight")) return;
    var code = block.querySelector("code") || block;
    var btn = document.createElement("button");
    btn.type = "button";
    btn.className = "pre-copy";
    btn.textContent = "copy";
    btn.setAttribute("aria-label", "Copy code");
    btn.addEventListener("click", function () {
      copy(code.textContent.replace(/\n$/, ""), btn);
    });
    block.appendChild(btn);
  });
})();
