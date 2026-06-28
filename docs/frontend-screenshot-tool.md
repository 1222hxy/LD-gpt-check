# Frontend screenshot tool

This project uses Playwright only for local visual checks of the `frontend/` site.

## What was installed

Project files:

```text
frontend/scripts/screenshot.js
```

NPM dependency:

```text
playwright
```

Local browser cache:

```text
frontend/.cache/ms-playwright/
```

Local browser runtime libraries:

```text
frontend/.cache/browser-libs/
```

Screenshot output:

```text
frontend/screenshots/
```

The `.cache/` and `screenshots/` directories are ignored in `frontend/.gitignore`.

## Install browser

```bash
cd frontend
npm run shot:install
```

In this environment Chromium also needed local NSS/NSPR shared libraries. They were downloaded with:

```bash
cd frontend
mkdir -p .cache/browser-libs/debs .cache/browser-libs/root
cd .cache/browser-libs/debs
apt-get download libnspr4 libnss3
cd ../../..
find .cache/browser-libs/debs -name '*.deb' -print -exec dpkg-deb -x {} .cache/browser-libs/root \;
```

No browser files are stored in the default Playwright cache.

## Take screenshots

Start the dev server:

```bash
cd frontend
npm run dev
```

Then run:

```bash
npm run shot
```

Output:

```text
frontend/screenshots/desktop.png
frontend/screenshots/mobile.png
```

Use a different URL:

```bash
URL=https://example.com npm run shot
```

## Delete the screenshot tool

To remove all screenshot tooling:

```bash
cd frontend
npm uninstall playwright
rm -rf .cache screenshots scripts/screenshot.js
```

Then remove these scripts from `frontend/package.json`:

```json
"shot:install": "PLAYWRIGHT_BROWSERS_PATH=.cache/ms-playwright playwright install chromium",
"shot": "PLAYWRIGHT_BROWSERS_PATH=.cache/ms-playwright LD_LIBRARY_PATH=.cache/browser-libs/root/usr/lib/x86_64-linux-gnu:$LD_LIBRARY_PATH node scripts/screenshot.js"
```

Optionally remove these ignore lines from `frontend/.gitignore`:

```text
.cache/
screenshots/
```
