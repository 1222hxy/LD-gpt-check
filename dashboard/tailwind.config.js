export default {
  content: ["./index.html", "./src/**/*.{js,jsx}"],
  theme: {
    extend: {
      fontFamily: {
        sans: [
          "Inter",
          "Noto Sans SC",
          "ui-sans-serif",
          "system-ui",
          "sans-serif",
        ],
      },
      colors: {
        ink: "#111827",
        moss: "#166534",
        sea: "#0f766e",
        rust: "#b45309",
        berry: "#be123c",
      },
      boxShadow: {
        soft: "0 16px 50px rgba(15, 23, 42, 0.08)",
      },
    },
  },
  plugins: [],
};
