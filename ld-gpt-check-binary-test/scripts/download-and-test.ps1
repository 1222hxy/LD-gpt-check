param(
    [switch]$RunTest,
    [string]$InstallDir = ".ld-gpt-check-bin",
    [string]$Repo = "1222hxy/LD-gpt-check",
    [string]$Model = "",
    [ValidateSet("low", "medium", "high", "xhigh")]
    [string]$Effort = "medium",
    [string]$Timeout = "30m"
)

$ErrorActionPreference = "Stop"

function Get-LdgptArch {
    $arch = $env:PROCESSOR_ARCHITEW6432
    if (-not $arch) {
        $arch = $env:PROCESSOR_ARCHITECTURE
    }
    if (-not $arch -and $PSVersionTable.OS) {
        $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    }

    switch -Regex ($arch) {
        "AMD64|X64" { return "amd64" }
        "ARM64" { return "arm64" }
        default {
            throw "Unsupported Windows architecture: $arch"
        }
    }
}

$arch = Get-LdgptArch
$asset = "ld-gpt-check_windows_${arch}.exe"
$url = "https://github.com/$Repo/releases/latest/download/$asset"

New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
$binPath = Join-Path $InstallDir "ld-gpt-check.exe"

Write-Host "Downloading $asset"
Write-Host "Source: $url"
Invoke-WebRequest -Uri $url -OutFile $binPath

Write-Host ""
Write-Host "Binary: $binPath"
& $binPath version

Write-Host ""
Write-Host "Checking built-in suites"
& $binPath run --no-remote-questions --list-suites

if (-not $RunTest) {
    Write-Host ""
    Write-Host "Downloaded and smoke-checked. Re-run with -RunTest to execute one benchmark pass."
    exit 0
}

$codex = Get-Command codex -ErrorAction SilentlyContinue
$codexCmd = Get-Command codex.cmd -ErrorAction SilentlyContinue
if (-not $codex -and -not $codexCmd) {
    throw "Codex CLI was not found in PATH. Install and log in to Codex before running the benchmark."
}

$runArgs = @("run", "--no-remote-questions", "-n", "1", "--timeout", $Timeout, "-r", $Effort)
if ($Model) {
    $runArgs += @("-m", $Model)
}

Write-Host ""
Write-Host "Running one LD-gpt-check benchmark pass"
& $binPath @runArgs
