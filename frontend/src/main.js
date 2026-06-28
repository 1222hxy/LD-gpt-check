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
        '"SFMono-Regular", "Cascadia Code", Consolas, "Liberation Mono", Menlo, Monaco, monospace',
      fontSize: window.matchMedia("(max-width: 640px)").matches ? 11.5 : 13,
      lineHeight: 1.42,
      rows: 15,
      scrollback: 0,
      theme: {
        background: "#ffffff",
        foreground: "#162033",
        cursor: "#2563eb",
        black: "#162033",
        blue: "#2563eb",
        brightBlue: "#0284c7",
        brightCyan: "#0891b2",
        brightGreen: "#047857",
        brightMagenta: "#7c3aed",
        brightRed: "#e11d48",
        brightYellow: "#b45309",
        cyan: "#0891b2",
        green: "#047857",
        magenta: "#7c3aed",
        red: "#e11d48",
        white: "#ffffff",
        yellow: "#b45309",
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
      blue: "\x1b[38;2;37;99;235m",
      cyan: "\x1b[38;2;8;145;178m",
      green: "\x1b[38;2;4;120;87m",
      red: "\x1b[38;2;225;29;72m",
      yellow: "\x1b[38;2;180;83;9m",
      violet: "\x1b[38;2;124;58;237m",
      gray: "\x1b[38;2;100;116;139m",
    };

    const command = `${c.blue}~/bench${c.reset} ${c.gray}$${c.reset} ${c.bold}ld-gpt-check run -m gpt-5.5 -r xhigh -n 5 --upload${c.reset}`;
    const outputEvents = [
      { delay: 120, line: "" },
      {
        delay: 240,
        line: `${c.cyan}start${c.reset} model=${c.bold}gpt-5.5${c.reset} reasoning=${c.violet}xhigh${c.reset} tests=${c.yellow}5${c.reset}`,
      },
      { delay: 650, line: `${c.gray}[00:00]${c.reset} case 1/5 candy_21  running codex exec --json` },
      { delay: 1100, line: `${c.gray}[01:31]${c.reset} case 1/5  ${c.green}PASS${c.reset}  answer=21  time=91.5s  tps=53.3` },
      { delay: 1250, line: `${c.gray}[03:15]${c.reset} case 2/5  ${c.red}FAIL${c.reset}  answer=27  time=104.2s tps=50.0` },
      { delay: 900, line: `${c.gray}[08:04]${c.reset} case 5/5  ${c.green}PASS${c.reset}  answer=21  time=88.7s  tps=51.0` },
      { delay: 160, line: "" },
      {
        delay: 260,
        line: `${c.yellow}Run${c.reset}  ${c.yellow}In Tok${c.reset}  ${c.yellow}Out Tok${c.reset}  ${c.yellow}Reason Tok${c.reset}  ${c.yellow}Time(s)${c.reset}   ${c.yellow}TPS${c.reset}  ${c.yellow}OK${c.reset}`,
      },
      { delay: 90, line: `1     10163     4873        4660     91.5  53.3  ${c.green}✓${c.reset}` },
      { delay: 90, line: `2     10163     5210        4901    104.2  50.0  ${c.red}×${c.reset}` },
      { delay: 90, line: `5     10163     4522        4310     88.7  51.0  ${c.green}✓${c.reset}` },
      {
        delay: 520,
        line: `${c.green}summary${c.reset} correct=4/5 accuracy=${c.bold}80.0%${c.reset} avg_time=96.8s avg_tps=51.4`,
      },
      { delay: 240, line: `${c.cyan}upload${c.reset} saved summary, token metrics and case previews only` },
    ];
    const outputLines = outputEvents.map((event) => event.line);

    const sleep = (ms) => new Promise((resolve) => window.setTimeout(resolve, ms));
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;

    async function writeAnimated() {
      if (reducedMotion) {
        terminal.writeln(command);
        outputLines.forEach((line) => terminal.writeln(line));
        return;
      }

      for (const chunk of command.match(/(\x1b\[[0-9;]*m|.)/g) || []) {
        terminal.write(chunk);
        if (!chunk.startsWith("\x1b")) {
          await sleep(chunk === " " ? 8 : 13);
        }
      }
      terminal.writeln("");

      for (const { delay, line } of outputEvents) {
        await sleep(delay);
        terminal.writeln(line);
      }
    }

    writeAnimated().catch((error) => {
      console.error("failed to animate hero terminal", error);
      terminal.clear();
      terminal.writeln(command);
      outputLines.forEach((line) => terminal.writeln(line));
    });

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
