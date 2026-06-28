/** @type {import('tailwindcss').Config} */
export default {
  content: ["./index.html", "./src/**/*.{js,ts,jsx,tsx}"],
  theme: {
    extend: {
      colors: {
        ink: "#080c0f",
        mint: "#68f0a5",
        sky: "#79d7ff",
        amber: "#f0c46c",
        red: "#ff6b7a",
        violet: "#a997ff",
      },
      boxShadow: {
        glow: "0 0 22px rgba(37, 99, 235, 0.26)",
      },
      fontFamily: {
        sans: [
          "Noto Sans SC",
          "Inter",
          "ui-sans-serif",
          "system-ui",
          "-apple-system",
          "BlinkMacSystemFont",
          "Segoe UI",
          "PingFang SC",
          "Hiragino Sans GB",
          "Microsoft YaHei",
          "sans-serif",
        ],
        mono: [
          "Noto Sans SC",
          "SFMono-Regular",
          "Cascadia Code",
          "Roboto Mono",
          "Consolas",
          "Liberation Mono",
          "monospace",
        ],
      },
    },
  },
  plugins: [],
};
