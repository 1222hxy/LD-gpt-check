# Repository Guidelines

## Project Structure & Module Organization

This repository currently contains no application source, tests, or build assets in the working tree. When adding code, keep the layout predictable:

- `src/` for production source code.
- `tests/` for automated tests that mirror `src/` structure.
- `docs/` for contributor or architecture notes.
- `assets/` for static files such as images, fixtures, or sample data.

Avoid placing generated files, dependency folders, or local environment output at the repository root.

## Build, Test, and Development Commands

No build or test commands are defined yet. Add project-specific commands in the relevant manifest when introducing a toolchain, such as `package.json`, `pyproject.toml`, `Makefile`, or similar.

Recommended examples:

- `npm test` or `pytest` to run the full test suite.
- `npm run lint` or `ruff check .` to run static checks.
- `npm run dev` or `make dev` to start a local development server.

Document any new commands in this file as soon as they become part of the normal workflow.

## Coding Style & Naming Conventions

Follow the formatter and linter configured for the language added to the repository. If none exists yet, add one before the codebase grows. Prefer clear, descriptive names over abbreviations.

Use consistent naming within each language ecosystem: `snake_case` for Python modules and functions, `camelCase` for JavaScript variables and functions, and `PascalCase` for classes, React components, or exported types.

## Testing Guidelines

Place tests under `tests/` unless the selected framework has a stronger convention. Name test files after the unit or feature they cover, for example `tests/test_parser.py` or `src/parser.test.ts`.

Every behavioral change should include a focused test. For bug fixes, add a regression test that fails before the fix and passes afterward.

## Commit & Pull Request Guidelines

This working tree does not expose Git history, so no repository-specific commit convention can be inferred. Use concise, imperative commit messages such as `Add parser tests` or `Fix config validation`.

Pull requests should include a short summary, the reason for the change, tests run, and any user-visible impact. Include screenshots for UI changes and link related issues when available.

## Security & Configuration Tips

Do not commit secrets, API keys, local credentials, or machine-specific configuration. Store local settings in ignored environment files such as `.env.local`, and provide documented examples like `.env.example` when configuration is required.
