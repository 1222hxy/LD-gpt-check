# LD-gpt-check Frontend

TailwindCSS + Vite static homepage for LD-gpt-check. This directory contains only static frontend source.

Backend API code, OAuth secrets, D1 schema, and Worker deployment config live in `../worker/`.

## Boundary Rules

- The product frontend and backend deploy together to the same Cloudflare Worker.
- Do not deploy this frontend to Cloudflare Pages for production.
- `npm run build` writes static assets to `frontend/dist/`; `worker/wrangler.toml` serves that directory through Worker Static Assets.
- Deploy from `../worker` with `wrangler deploy`, or run `npm run deploy` here to build then deploy the Worker.
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

## Deploy

Production deployment is a single Cloudflare Worker with static assets:

```bash
cd frontend
npm run build
cd ../worker
../frontend/node_modules/.bin/wrangler deploy
```

From this directory, the shortcut is:

```bash
npm run deploy
```

This builds `frontend/dist/` and deploys the Worker that serves both the frontend and backend.

See the detailed guide: [`../docs/cloudflare-worker-deploy.md`](../docs/cloudflare-worker-deploy.md).

## Screenshots

Local screenshot tooling is documented here:

```text
../docs/frontend-screenshot-tool.md
```
