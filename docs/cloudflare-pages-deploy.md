# Deploy the LD-gpt-check homepage to Cloudflare Pages

This guide is kept only for historical/static-preview reference. Production does not use Cloudflare Pages.

Production deployment is a single Cloudflare Worker:

```bash
cd frontend
npm run build
cd ../worker
../frontend/node_modules/.bin/wrangler deploy
```

The Worker serves `frontend/dist/` through Worker Static Assets and also handles `/account`, `/device`, `/auth/*`, `/api/*`, `/logout`, and `/health`. For production, follow [`cloudflare-worker-deploy.md`](./cloudflare-worker-deploy.md).

The old Pages flow below should not be used for production.

Official references:

- Cloudflare Pages build configuration: https://developers.cloudflare.com/pages/configuration/build-configuration/
- Cloudflare Pages Vite guide: https://developers.cloudflare.com/pages/framework-guides/deploy-a-vite3-project/
- Wrangler Pages commands: https://developers.cloudflare.com/workers/wrangler/commands/#pages

## Project layout

```text
frontend/
  index.html
  package.json
  package-lock.json
  src/
    main.js
    styles.css
  tailwind.config.js
  postcss.config.js
  vite.config.js
  wrangler.toml
```

The build output is `frontend/dist/`. It is generated locally and ignored by Git.

## Local prerequisites

Use Node 22 or newer. The project includes `frontend/.nvmrc`:

```bash
cd frontend
nvm use
```

Install dependencies:

```bash
npm ci
```

For local development:

```bash
npm run dev
```

For a production build:

```bash
npm run build
```

Verify the production build locally:

```bash
npm run preview
```

## Recommended deployment: Cloudflare Pages with Git

This is the best default path. Cloudflare builds every push and gives you preview deployments for branches and pull requests.

1. Push the repository to GitHub or GitLab.
2. Open Cloudflare Dashboard.
3. Go to `Workers & Pages`.
4. Choose `Create application`.
5. Choose `Pages`.
6. Choose `Connect to Git`.
7. Select the repository.
8. Configure build settings:

```text
Project name: ld-gpt-check
Production branch: main
Root directory: frontend
Framework preset: Vite
Build command: npm run build
Build output directory: dist
```

Set the Node version to 22. In Cloudflare Pages, add an environment variable if needed:

```text
NODE_VERSION=22
```

Then deploy. Cloudflare will run the build from `frontend/` and publish `frontend/dist/`.

## Manual deployment with Wrangler

Use this when you want to deploy from your machine without connecting Git.

From `frontend/`:

```bash
npm ci
npm run cf:login
npm run build
npm run deploy
```

The `deploy` script runs:

```bash
npm run build && wrangler pages deploy dist --project-name ld-gpt-check
```

If the Cloudflare project does not exist yet, Wrangler will prompt you to create it or associate the deployment with a Pages project.

## Preview deployments

Cloudflare Pages automatically creates preview deployments for non-production branches when using Git integration.

For manual deployments, use a branch name:

```bash
npx wrangler pages deploy dist --project-name ld-gpt-check --branch preview-homepage
```

## Custom domain

After the first successful deployment:

1. Open the Pages project in Cloudflare.
2. Go to `Custom domains`.
3. Add the domain or subdomain, for example:

```text
ld-gpt-check.example.com
```

4. Follow Cloudflare's DNS prompt.

If the domain is already on Cloudflare DNS, this is usually automatic.

## Environment variables

The current homepage is a static site and does not require runtime secrets.

Use environment variables only for build-time public values. Do not put secrets into Vite variables because frontend variables are bundled into client-side assets.

If a future version needs a public API base URL, use a `VITE_` variable:

```text
VITE_API_BASE_URL=https://api.example.com
```

Read it in frontend code with:

```js
import.meta.env.VITE_API_BASE_URL
```

## Cache and rebuild notes

If Cloudflare appears to serve an old build:

1. Confirm the latest commit was deployed.
2. Check the build log for `npm run build`.
3. Confirm the output directory is exactly `dist`.
4. Trigger `Retry deployment`.

The built asset filenames are content-hashed by Vite, so normal CSS/JS updates should invalidate cleanly.

## Common problems

### Build command runs from the wrong directory

Symptom:

```text
npm ERR! enoent Could not read package.json
```

Fix:

```text
Root directory: frontend
Build command: npm run build
Build output directory: dist
```

### Tailwind styles are missing

Run locally:

```bash
cd frontend
npm run build
```

If the local build works, check that Cloudflare is using the committed `package-lock.json`, `tailwind.config.js`, `postcss.config.js`, and `src/styles.css`.

### Wrong Node version

Set:

```text
NODE_VERSION=22
```

Then redeploy.

### Wrangler is not logged in

Run:

```bash
cd frontend
npm run cf:login
npm run cf:whoami
```

Then deploy again:

```bash
npm run deploy
```
