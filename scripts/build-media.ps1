param(
    [string]$ToolchainRoot = '',
    [string]$Archives = '',
    [ValidateRange(1,64)][int]$Jobs = 4
)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$build = Join-Path $root 'build\release-media'
$lock = Get-Content -LiteralPath (Join-Path $root 'third_party\media-sources.json') -Raw | ConvertFrom-Json
if (-not $Archives) { $Archives = Join-Path $build 'archives' }
New-Item -ItemType Directory -Path $Archives -Force | Out-Null
New-Item -ItemType Directory -Path $build -Force | Out-Null

function Get-PinnedArchive($Entry) {
    $archive = Join-Path $Archives $Entry.filename
    if (-not (Test-Path -LiteralPath $archive)) {
        Write-Host ('Downloading: ' + $Entry.filename)
        & curl.exe --location --fail --silent --show-error --retry 3 --retry-all-errors --max-time 240 --output $archive $Entry.url
        if ($LASTEXITCODE -ne 0) { throw ('Download failed: ' + $Entry.url) }
    }
    $digest = (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($digest -ne $Entry.sha256) { throw ('Source hash mismatch: ' + $archive + '; expected ' + $Entry.sha256 + ', got ' + $digest) }
    return $archive
}

foreach ($entry in $lock.sources) { $null = Get-PinnedArchive $entry }
$compilerArchive = Join-Path $Archives $lock.toolchain.filename
$existingCompilerArchive = Join-Path $root 'build\devtools\llvm-mingw-ucrt.zip'
if (-not (Test-Path -LiteralPath $compilerArchive) -and (Test-Path -LiteralPath $existingCompilerArchive)) {
    $digest = (Get-FileHash -LiteralPath $existingCompilerArchive -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($digest -eq $lock.toolchain.sha256) { Copy-Item -LiteralPath $existingCompilerArchive -Destination $compilerArchive }
}
$compilerArchive = Get-PinnedArchive $lock.toolchain
if (-not $ToolchainRoot) {
    $compilerDirectory = Join-Path $build 'compiler'
    $ToolchainRoot = Join-Path $compilerDirectory $lock.toolchain.folder
    if (-not (Test-Path -LiteralPath (Join-Path $ToolchainRoot 'bin\clang.exe'))) {
        Expand-Archive -LiteralPath $compilerArchive -DestinationPath $compilerDirectory
    }
}
$ToolchainRoot = (Resolve-Path -LiteralPath $ToolchainRoot).Path
$sourceDirectory = Join-Path $build 'sources'
New-Item -ItemType Directory -Path $sourceDirectory -Force | Out-Null
foreach ($entry in $lock.sources | Where-Object { $_.name -in @('ffmpeg','lame') }) {
    & tar.exe -xf (Join-Path $Archives $entry.filename) -C $sourceDirectory
    if ($LASTEXITCODE -ne 0) { throw ('Source extraction failed: ' + $entry.filename) }
}
foreach ($entry in $lock.patches) {
    $digest = (Get-FileHash -LiteralPath (Join-Path $root $entry.path) -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($digest -ne $entry.sha256) { throw ('Build patch hash mismatch: ' + $entry.path) }
}
$git = (Get-Command git -ErrorAction Stop).Source
$gitRoot = Split-Path -Parent (Split-Path -Parent $git)
$bash = Join-Path $gitRoot 'bin\bash.exe'
if (-not (Test-Path -LiteralPath $bash)) { throw 'Git for Windows with Bash is required.' }
& $bash (Join-Path $PSScriptRoot 'build-media.sh') $sourceDirectory $ToolchainRoot $build $Jobs
if ($LASTEXITCODE -ne 0) { throw 'Pinned FFmpeg/LAME build failed.' }
$binaries = @('ffmpeg.exe','ffprobe.exe') | ForEach-Object {
    $path = Join-Path $build ('bin\' + $_)
    $firstLine = (& $path -version | Select-Object -First 1)
    if ($LASTEXITCODE -ne 0 -or $firstLine -notmatch 'version 8\.1\.1') { throw ('Unexpected built media version: ' + $firstLine) }
    [ordered]@{ name=$_; bytes=(Get-Item -LiteralPath $path).Length; sha256=(Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant(); version=$firstLine }
}
$receipt = [ordered]@{
    schema=1; profile=$lock.profile
    source_lock_sha256=(Get-FileHash -LiteralPath (Join-Path $root 'third_party\media-sources.json') -Algorithm SHA256).Hash.ToLowerInvariant()
    build_script_sha256=(Get-FileHash -LiteralPath (Join-Path $PSScriptRoot 'build-media.sh') -Algorithm SHA256).Hash.ToLowerInvariant()
    toolchain_archive_sha256=$lock.toolchain.sha256
    sources=$lock.sources; patches=$lock.patches; binaries=@($binaries)
}
$receipt | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath (Join-Path $build 'build-receipt.json') -Encoding utf8
Write-Output ('Built pinned media tools: ' + (Join-Path $build 'bin'))
