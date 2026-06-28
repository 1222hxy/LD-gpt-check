---
name: ld-gpt-check-binary-test
description: Download the latest LD-gpt-check release binary for the current operating system, verify it starts, and run a one-pass local benchmark with the built-in question suite. Use when a user asks an agent to install, download, smoke-test, or "帮我测" LD-gpt-check from a released binary instead of building from source.
---

# LD-gpt-check Binary Test

## Overview

Use this skill to help a user try LD-gpt-check with the published GitHub release binary. Use the bundled Bash script on macOS/Linux/Git Bash/WSL, and the bundled PowerShell script on native Windows.

## Workflow

1. Confirm the user has Codex CLI installed and logged in before running the real benchmark. If not, stop after download and tell the user to install/login to Codex first.
2. Download the latest release binary that matches the current OS and CPU.
3. Run a non-network smoke check:
   `ld-gpt-check version`
   `ld-gpt-check run --no-remote-questions --list-suites`
4. When the user asked to test, run exactly one local benchmark pass and do not upload:
   `ld-gpt-check run --no-remote-questions -n 1 --timeout 30m`
5. Report the binary path, whether the smoke checks passed, and the one-pass result summary.

## macOS/Linux/Git Bash

From the repository root containing this skill:

```bash
bash ld-gpt-check-binary-test/scripts/download-and-test.sh --run-test
```

Useful options:

- `--install-dir DIR`: put the downloaded binary somewhere else.
- `--model MODEL`: pass an explicit model to `ld-gpt-check run`.
- `--effort low|medium|high|xhigh`: set reasoning effort, default `medium`.
- `--timeout 10m`: set the per-test timeout, default `30m`.
- Omit `--run-test` to download and smoke-check only.

## Windows PowerShell

From the repository root containing this skill:

```powershell
powershell -ExecutionPolicy Bypass -File .\ld-gpt-check-binary-test\scripts\download-and-test.ps1 -RunTest
```

Useful options:

- `-InstallDir DIR`: put the downloaded binary somewhere else.
- `-Model MODEL`: pass an explicit model to `ld-gpt-check run`.
- `-Effort low|medium|high|xhigh`: set reasoning effort, default `medium`.
- `-Timeout 10m`: set the per-test timeout, default `30m`.
- Omit `-RunTest` to download and smoke-check only.

The PowerShell script automatically chooses `ld-gpt-check_windows_amd64.exe` or `ld-gpt-check_windows_arm64.exe`.

## Reporting

Keep the final user-facing report short:

- Binary path
- Downloaded asset name
- Smoke-check status
- One-pass test result, including pass/fail, model, token counts, elapsed time, and TPS when available
- Any blocker, especially missing Codex CLI login, network failure, or platform mismatch
