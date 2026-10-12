param(
    [string]$FFmpeg = '',
    [string]$FFprobe = '',
    [switch]$Race
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$oldPath = $env:Path
$oldCGO = $env:CGO_ENABLED
$oldOS = $env:GOOS
$oldArch = $env:GOARCH
$oldCC = $env:CC
$env:Path = [Environment]::ExpandEnvironmentVariables(
    [Environment]::GetEnvironmentVariable('Path', 'Machine') + ';' +
    [Environment]::GetEnvironmentVariable('Path', 'User'))
if (-not (Get-Command go -ErrorAction SilentlyContinue)) { throw 'Go 1.26 or newer is required.' }
if (-not $FFmpeg) {
    & (Join-Path $PSScriptRoot 'build-media.ps1')
    $FFmpeg = Join-Path $root 'build\release-media\bin\ffmpeg.exe'
}
$FFmpeg = (Resolve-Path -LiteralPath $FFmpeg).Path
if (-not $FFprobe) { $FFprobe = Join-Path (Split-Path -Parent $FFmpeg) 'ffprobe.exe' }
$FFprobe = (Resolve-Path -LiteralPath $FFprobe).Path
Push-Location -LiteralPath $root
try {
    $env:GOOS = 'windows'
    $env:GOARCH = 'amd64'
    $env:CGO_ENABLED = '0'
    go mod tidy
    if ($LASTEXITCODE -ne 0) { throw 'go mod tidy failed.' }
    go run ./cmd/package-media -ffmpeg $FFmpeg -ffprobe $FFprobe -output (Join-Path $root 'internal\media\assets')
    if ($LASTEXITCODE -ne 0) { throw 'Embedded media packaging failed.' }
    go run ./cmd/generate-logo -output 'internal/desktop/assets'
    if ($LASTEXITCODE -ne 0) { throw 'Logo generation failed.' }
    go run github.com/akavel/rsrc@v0.10.2 -arch amd64 -ico 'internal/desktop/assets/logo.ico' -manifest 'internal/desktop/assets/app.manifest' -o 'cmd/qqmusic-convert/rsrc_windows_amd64.syso'
    if ($LASTEXITCODE -ne 0) { throw 'Executable icon/manifest embedding failed.' }
    $unformatted = @(gofmt -l cmd internal)
    if ($LASTEXITCODE -ne 0 -or $unformatted.Count -gt 0) { throw ('Run gofmt first: ' + ($unformatted -join ', ')) }
    go vet ./...
    if ($LASTEXITCODE -ne 0) { throw 'go vet failed.' }
    go test ./...
    if ($LASTEXITCODE -ne 0) { throw 'Go tests failed.' }
    if ($Race) {
        if (-not $env:CC) {
            $compiler = Get-Command gcc,clang -ErrorAction SilentlyContinue | Select-Object -First 1
            if ($compiler) { $env:CC = $compiler.Source }
        }
        if (-not $env:CC) { throw 'Race tests need a native Windows GCC or LLVM/MinGW compiler; set CC first.' }
        $env:CGO_ENABLED = '1'
        go test -race ./...
        if ($LASTEXITCODE -ne 0) { throw 'Race detector tests failed.' }
        $env:CGO_ENABLED = '0'
    }
    $dist = Join-Path $root 'dist'
    New-Item -ItemType Directory -Path $dist -Force | Out-Null
    $binary = Join-Path $dist 'qqmusic-convert.exe'
    go build -trimpath -ldflags '-H windowsgui -s -w' -o $binary ./cmd/qqmusic-convert
    if ($LASTEXITCODE -ne 0) { throw 'Release EXE build failed.' }
    Copy-Item -LiteralPath (Join-Path $root 'README.md') -Destination (Join-Path $dist '使用说明.md')
    $imageDirectory = Join-Path $dist 'docs\images'
    New-Item -ItemType Directory -Path $imageDirectory -Force | Out-Null
    Copy-Item -LiteralPath (Join-Path $root 'docs\images\main-window.png') -Destination (Join-Path $imageDirectory 'main-window.png')
    $digest = (Get-FileHash -LiteralPath $binary -Algorithm SHA256).Hash.ToLowerInvariant()
    ($digest + '  qqmusic-convert.exe') | Set-Content -LiteralPath (Join-Path $dist 'SHA256.txt') -Encoding ascii
    Write-Output ('Built: ' + $binary)
    Write-Output ('Only this EXE is required at runtime; no Python, Go or external FFmpeg installation.')
}
finally {
    Pop-Location
    $env:Path = $oldPath
    $env:CGO_ENABLED = $oldCGO
    $env:GOOS = $oldOS
    $env:GOARCH = $oldArch
    $env:CC = $oldCC
}
