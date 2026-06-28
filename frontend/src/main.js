import "@fontsource/noto-sans-sc/chinese-simplified-400.css";
import "@fontsource/noto-sans-sc/chinese-simplified-600.css";
import { FitAddon } from "@xterm/addon-fit";
import { Terminal } from "@xterm/xterm";
import "@xterm/xterm/css/xterm.css";
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

function renderHeroTerminal() {
  const mount = document.querySelector("#hero-terminal");
  if (!mount) {
    return;
  }

  try {
    const terminal = new Terminal({
      allowTransparency: true,
      convertEol: true,
      cursorBlink: true,
      cursorStyle: "bar",
      disableStdin: true,
      fontFamily:
        '"Cascadia Code", "SFMono-Regular", "Noto Sans SC", Consolas, "Liberation Mono", monospace',
      fontSize: window.matchMedia("(max-width: 640px)").matches ? 11 : 12,
      lineHeight: 1.35,
      rows: 15,
      scrollback: 0,
      theme: {
        background: "#08111f",
        foreground: "#d7e3ff",
        cursor: "#68f0a5",
        black: "#101827",
        blue: "#79d7ff",
        brightBlue: "#9de8ff",
        brightCyan: "#7df9ff",
        brightGreen: "#8ff7b3",
        brightMagenta: "#c8b6ff",
        brightRed: "#ff8ca0",
        brightYellow: "#ffe08a",
        cyan: "#5eead4",
        green: "#68f0a5",
        magenta: "#a997ff",
        red: "#ff6b7a",
        white: "#f8fbff",
        yellow: "#f0c46c",
      },
    });
    const fitAddon = new FitAddon();
    terminal.loadAddon(fitAddon);
    terminal.open(mount);

    const fit = () => {
      try {
        fitAddon.fit();
      } catch {
        // The hero terminal is decorative; layout should not break if fitting fails.
      }
    };

    fit();
    requestAnimationFrame(fit);

    const c = {
      reset: "\x1b[0m",
      bold: "\x1b[1m",
      dim: "\x1b[2m",
      blue: "\x1b[38;2;121;215;255m",
      cyan: "\x1b[38;2;94;234;212m",
      green: "\x1b[38;2;104;240;165m",
      red: "\x1b[38;2;255;107;122m",
      yellow: "\x1b[38;2;240;196;108m",
      violet: "\x1b[38;2;169;151;255m",
      gray: "\x1b[38;2;116;133;166m",
    };

    [
      `${c.blue}~/bench${c.reset} ${c.gray}$${c.reset} ${c.bold}ld-gpt-check run -m gpt-5.5 -r xhigh -n 5 --upload${c.reset}`,
      "",
      `${c.cyan}start${c.reset} model=${c.bold}gpt-5.5${c.reset} reasoning=${c.violet}xhigh${c.reset} tests=${c.yellow}5${c.reset}`,
      `${c.gray}case 1/5${c.reset} candy_21  ${c.green}PASS${c.reset}  answer=21  time=18.7s  tps=39.4`,
      `${c.gray}case 2/5${c.reset} candy_21  ${c.red}FAIL${c.reset}  answer=27  time=21.3s  tps=38.1`,
      `${c.gray}case 5/5${c.reset} candy_21  ${c.green}PASS${c.reset}  answer=21  time=17.9s  tps=38.5`,
      "",
      `${c.yellow}Run${c.reset}  ${c.yellow}In Tok${c.reset}  ${c.yellow}Out Tok${c.reset}  ${c.yellow}Reason Tok${c.reset}  ${c.yellow}Time(s)${c.reset}   ${c.yellow}TPS${c.reset}  ${c.yellow}OK${c.reset}`,
      `${c.dim}---  ------  -------  ----------  -------  ----  --${c.reset}`,
      `1       412      736        2100     18.7  39.4  ${c.green}✓${c.reset}`,
      `2       412      812        2800     21.3  38.1  ${c.red}×${c.reset}`,
      `5       412      690        1900     17.9  38.5  ${c.green}✓${c.reset}`,
      "",
      `${c.green}summary${c.reset} correct=4/5 accuracy=${c.bold}80.0%${c.reset} avg_time=19.4s avg_tps=38.9`,
      `${c.cyan}upload${c.reset} saved summary, token metrics and case previews only`,
    ].forEach((line) => terminal.writeln(line));

    const resizeObserver = new ResizeObserver(fit);
    resizeObserver.observe(mount);
  } catch (error) {
    mount.textContent = [
      "$ ld-gpt-check run -m gpt-5.5 -r xhigh -n 5 --upload",
      "summary correct=4/5 accuracy=80.0% avg_time=19.4s avg_tps=38.9",
    ].join("\n");
    console.error("failed to render hero terminal", error);
  }
}

renderHeroTerminal();
