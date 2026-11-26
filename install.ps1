#!/usr/bin/env pwsh
# install.ps1 — installs the nxs CLI on Windows.
#
# The Windows counterpart of install.sh: same release, same archive naming, same
# checksum file. Everything happens under the user's profile, so no administrator
# rights are needed.
#
#   irm https://raw.githubusercontent.com/nexspence/nxs/main/install.ps1 | iex
#
# or with an explicit version and install directory:
#
#   .\install.ps1 -Version v0.5.0 -InstallDir "$env:LOCALAPPDATA\Programs\nxs"

[CmdletBinding()]
param(
    # Pin a release tag (e.g. v0.5.0). Defaults to the latest release.
    [string] $Version,
    # Where the binary is placed. Defaults to %LOCALAPPDATA%\Programs\nxs.
    [string] $InstallDir
)

$ErrorActionPreference = 'Stop'

$Repo = 'nexspence/nxs'
$Binary = 'nxs.exe'

# ── architecture ──────────────────────────────────────────────────────────────
# The release archives are built for amd64 and arm64 only.
$arch = switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { 'amd64' }
    'ARM64' { 'arm64' }
    'x86'   { 'amd64' }   # 32-bit Windows on a 64-bit CPU runs the x64 binary
    default { $null }
}
if (-not $arch) {
    throw "Unsupported architecture: $env:PROCESSOR_ARCHITECTURE (expected AMD64 or ARM64)"
}

# ── version ───────────────────────────────────────────────────────────────────
if (-not $Version) {
    # Windows PowerShell 5.1 defaults to TLS 1.0, which the GitHub API refuses.
    if ($PSVersionTable.PSVersion.Major -lt 6) {
        [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    }
    $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $release.tag_name
}
if ($Version -notmatch '^v\d+\.\d+\.\d+') {
    throw "Unrecognised version '$Version' (expected a tag like v0.5.0)"
}

# ── paths ─────────────────────────────────────────────────────────────────────
if (-not $InstallDir) {
    $InstallDir = Join-Path $env:LOCALAPPDATA 'Programs\nxs'
}
$work = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
New-Item -ItemType Directory -Path $work -Force | Out-Null
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null

try {
    # ── download ───────────────────────────────────────────────────────────────
    # The archive name mirrors install.sh: nxs_<version>_<os>_<arch>.zip
    $versionNoV = $Version.TrimStart('v')
    $archive = "nxs_${versionNoV}_windows_${arch}.zip"
    $url = "https://github.com/$Repo/releases/download/$Version/$archive"

    Write-Host "Installing nxs $Version ($arch)..."
    $archivePath = Join-Path $work $archive
    Invoke-WebRequest -Uri $url -OutFile $archivePath -UseBasicParsing

    # ── verify the checksum ────────────────────────────────────────────────────
    # goreleaser publishes checksums.txt next to the archives; install.sh does
    # not check it, this script does.
    $checksumsPath = Join-Path $work 'checksums.txt'
    try {
        Invoke-WebRequest -Uri "https://github.com/$Repo/releases/download/$Version/checksums.txt" `
            -OutFile $checksumsPath -UseBasicParsing
        $expected = (Get-Content $checksumsPath |
            Where-Object { $_ -match "\s+$([regex]::Escape($archive))\s*$" } |
            Select-Object -First 1 |
            ForEach-Object { ($_ -split '\s+')[0] })

        if ($expected) {
            $actual = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLower()
            if ($actual -ne $expected.ToLower()) {
                throw "Checksum mismatch for $archive`n  expected $expected`n  actual   $actual"
            }
            Write-Host "Checksum verified."
        }
    }
    catch {
        if ($_.Exception.Message -like 'Checksum mismatch*') { throw }
        Write-Warning "Could not verify the checksum: $($_.Exception.Message)"
    }

    # ── extract ────────────────────────────────────────────────────────────────
    $extractDir = Join-Path $work 'extract'
    Expand-Archive -Path $archivePath -DestinationPath $extractDir -Force

    $binary = Get-ChildItem -Path $extractDir -Filter $Binary -Recurse | Select-Object -First 1
    if (-not $binary) {
        throw "$Binary not found inside $archive"
    }
    Copy-Item -Path $binary.FullName -Destination (Join-Path $InstallDir $Binary) -Force

    # ── PATH ───────────────────────────────────────────────────────────────────
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if ($userPath -notlike "*$InstallDir*") {
        $newPath = if ([string]::IsNullOrEmpty($userPath)) { $InstallDir } else { "$userPath;$InstallDir" }
        [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
        Write-Host "Added $InstallDir to your user PATH (restart the shell to pick it up)."
    }

    Write-Host "Done. Run: nxs --help"
}
finally {
    Remove-Item -Path $work -Recurse -Force -ErrorAction SilentlyContinue
}
