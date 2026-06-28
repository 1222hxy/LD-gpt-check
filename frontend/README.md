# LD-gpt-check Frontend

TailwindCSS + Vite static homepage for LD-gpt-check. This directory is frontend-only.

Backend API code, OAuth secrets, D1 schema, and Worker deployment config live in `../worker/`.

## Boundary Rules

- Frontend deploys to Cloudflare Pages.
- Backend deploys from `../worker` to Cloudflare Workers.
- Do not add D1 bindings, OAuth client secrets, `TOKEN_SECRET`, or API route handlers in this directory.
- Only `VITE_` public variables are allowed here, and they must be safe to expose in browser code.
- If the frontend needs to call the backend, use a public API base URL such as `VITE_PUBLIC_API_BASE_URL`.

Copy the optional frontend env example when needed:

```bash
cp .env.example .env
```

## Logo

The gradient wordmark lives at:

```text
public/logo-wordmark.svg
```

It is a transparent-background SVG and is copied to the site root during `npm run build`.

## Local development

```bash
npm ci
npm run dev
```

## Build

```bash
npm run build
```

The production output is generated in `dist/`.

## Cloudflare Pages

Use these settings:

- Root directory: `frontend`
- Build command: `npm run build`
- Build output directory: `dist`
- Node version: `22`

You can also deploy manually after building:

```bash
npm run deploy
```

This deploy command publishes only static files from `dist/`. It does not deploy the API Worker.

See the detailed guide: [`../docs/cloudflare-pages-deploy.md`](../docs/cloudflare-pages-deploy.md).

## Screenshots

Local screenshot tooling is documented here:

```text
../docs/frontend-screenshot-tool.md
```
