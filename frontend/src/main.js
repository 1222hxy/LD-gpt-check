import "@fontsource/noto-sans-sc/chinese-simplified-400.css";
import "@fontsource/noto-sans-sc/chinese-simplified-600.css";
import { createJavaScriptRegexEngine } from "shiki/engine/javascript";
import { createHighlighterCore } from "shiki/core";
import shellscript from "shiki/langs/shellscript.mjs";
import githubLight from "shiki/themes/github-light.mjs";
import "./styles.css";

const navToggle = document.querySelector(".nav-toggle");
const nav = document.querySelector(".site-nav");
const year = document.querySelector("#year");

if (year) {
  year.textContent = new Date().getFullYear();
}

if (navToggle && nav) {
  navToggle.addEventListener("click", () => {
    const isOpen = nav.classList.toggle("is-open");
    navToggle.setAttribute("aria-expanded", String(isOpen));
  });

  nav.querySelectorAll("a").forEach((link) => {
    link.addEventListener("click", () => {
      nav.classList.remove("is-open");
      navToggle.setAttribute("aria-expanded", "false");
    });
  });
}

async function renderShikiBlocks() {
  const blocks = document.querySelectorAll("[data-shiki-code]");
  if (!blocks.length) {
    return;
  }
  const highlighter = await createHighlighterCore({
    themes: [githubLight],
    langs: [shellscript],
    engine: createJavaScriptRegexEngine(),
  });
  await Promise.all(
    [...blocks].map(async (block) => {
      const code = block.textContent.replace(/^\n+|\s+$/g, "");
      const lang = block.getAttribute("data-lang") || "shellscript";
      block.innerHTML = highlighter.codeToHtml(code, {
        lang: lang === "bash" ? "shellscript" : lang,
        theme: "github-light",
      });
    })
  );
}

renderShikiBlocks().catch((error) => {
  console.error("failed to render code block", error);
});
