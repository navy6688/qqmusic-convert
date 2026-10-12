# QQ Music Convert

English | [简体中文](README.md)

![Application logo](internal/desktop/assets/logo.svg)

A local audio converter for Windows, built with Go. Its native desktop interface converts
QQ Music musicex V1 files and common unencrypted audio formats to MP3, then verifies the output.

Use it only with local files you are entitled to access. The project does not download music,
log into accounts, export credentials, or bypass paid access. It is not an official Tencent
or QQ Music product.

## Features

- Batch conversion with selectable source and output folders.
- The executable's folder is the default source; its `MP3` subfolder is the default output.
- Per-file status, conversion stages, overall progress, and success / skip / failure counts.
- Cancellation, error details, output-folder access, and JSON conversion reports.
- Original files remain unchanged. Existing MP3 outputs are skipped, never overwritten.
- Native window resizing and maximization, with support for consecutive musicex footer layers.
- One executable with bundled media tools. No separate Python, Go, FFmpeg, or WebView2 installation at runtime.

## Interface preview

![QQ Music Convert interface showing folder selection, file status, and conversion progress](docs/images/main-window.png)

The screenshot shows the actual interface after converting three synthetic audio files, using demo paths and filenames.
Select source and output folders at the top, review each file in the list, and track overall progress and results at the bottom.
The desktop interface is currently in Simplified Chinese.

## Requirements

| Item | Requirement |
|---|---|
| Operating system | Windows 10 / 11, x64 |
| Encrypted audio | QQ Music desktop must be running and signed in, with access to the requested resource |
| Unencrypted audio | A QQ Music session is not required |
| Storage | About 23 MB for the EXE and 31 MB for the initial media cache, plus temporary audio and output space |

The executable is not commercially code-signed. Check its source and SHA-256 if Windows shows
an unknown-publisher prompt; do not disable security software. QQ Music updates may change
session or file internals, so compatibility with every client version is not guaranteed.

## Quick start

When releases are available, obtain the Windows EXE and checksum from this repository's
**Releases** page. Alternatively, build it from source using the steps below.

1. Place `qqmusic-convert.exe` in your music folder and open it.
2. Confirm the source and output folders, or choose different folders in the interface.
3. Open QQ Music and sign in before converting encrypted files.
4. Click **开始转换** (Start conversion) and wait for conversion and verification.
5. Click **打开保存目录** (Open output folder). Select a failed item and click
   **查看详情** (View details) to read its error.

The interface currently uses Simplified Chinese. An English README does not change the UI language.
Opening the app only scans files; conversion starts when you click the button. Only direct files
in the selected folder are processed. Subfolders are not scanned recursively.
After editing a folder path manually, refresh the list. Cancellation cleans up the current
job's temporary audio and preserves completed outputs.

## Supported formats

| Input | Formats |
|---|---|
| Encrypted audio | `.mflac`, `.mgg`, and `.mmp4` with a musicex V1 footer |
| Unencrypted audio | `.flac`, `.wav`, `.ogg`, `.m4a`, `.aac`, `.opus`, `.wma`, and `.mp3` |

Non-MP3 inputs are encoded at 320 kbps. MP3 inputs are verified and copied without another
lossy encoding pass. Multiple source formats with the same name receive format suffixes on
non-MP3 outputs, such as `Song (mflac).mp3`.

Legacy QMC formats, other providers' encrypted formats, and unknown future formats are not
supported. MP3 is lossy; the original files are always retained.

## Verification and privacy

- The original file's SHA-256 is checked before and after conversion. A changed file is not published.
- For FLAC with usable STREAMINFO values, all decoded PCM bytes are checked against the stored
  MD5 and sample count. Decoder warnings are accepted only when both checks match.
  This check supports 8-, 16-, 24-, and 32-bit FLAC.
- FLAC without usable checksums and other formats must pass strict full decoding.
- Every output MP3 is identified and fully decoded before publication. Existing outputs are not replaced.
- A matching key is requested separately for each encrypted resource. Validated session data is
  reused only in memory for the current batch.
- Only read access to `QQMusic.exe` is used. The tool does not write process memory, inject code, or elevate privileges.
- Required resource identifiers and authentication data go only to the fixed QQ Music HTTPS endpoint.
  Audio files are not uploaded.
- Login tokens and song keys are not written to configuration, logs, reports, or temporary files.

A failed job is recorded and processing continues. Reports contain paths, statuses, errors,
and verification data, but no keys. Remove personal paths and account information before
filing an issue, and do not attach real audio. Skipping an existing output does not verify it.

## Build from source

Build on Windows with Go 1.26+ and Git for Windows including Bash.
The default build downloads hash-pinned sources and the LLVM-MinGW toolchain, then builds
FFmpeg / FFprobe and LAME. These are build dependencies, not extra runtime requirements.
The first source build needs substantial downloads and compilation time.

From the project root:

```powershell
& ".\scripts\build.ps1"
```

To deliberately use another media build, provide its paths and retain its matching source/license materials. These paths are examples:

```powershell
& ".\scripts\build.ps1" `
  -FFmpeg ".\tools\ffmpeg\bin\ffmpeg.exe" `
  -FFprobe ".\tools\ffmpeg\bin\ffprobe.exe"
```

The script fetches Go modules, prepares embedded media assets, generates the logo, embeds the
icon and manifest, runs formatting checks, `go vet`, and tests, then builds a Windows GUI EXE
with `CGO_ENABLED=0`. The `dist/` folder contains the executable, Chinese instructions, and
`SHA256.txt`. An initial build needs network access for modules and the pinned resource compiler.

Media archives, binary icons, and resource objects are generated and do not belong in Git.
Do not rely on `go build` alone in a clean checkout without preparing these resources.
If you replace the media tools, update their recorded versions, sources, and licenses as well.

## Tests and diagnostics

After the build script has prepared resources:

```powershell
go test ./... -count=1
go vet ./...
```

Race detection additionally requires a native Windows GCC or LLVM/MinGW compiler.
Set `CC`, then use `build.ps1 -Race`. The compiler is not a runtime dependency.

The headless diagnostic command emits JSON and exits with status 1 if any conversion fails:

```powershell
& ".\dist\qqmusic-convert.exe" -convert ".\music" -output ".\mp3"
```

Normal use requires only opening the EXE. Regression tests use synthetic data without real
accounts or songs. Local checks against real audio are not included as public fixtures.

## Project layout

```text
cmd/qqmusic-convert/   Application entry point
cmd/package-media/    Media asset packaging
cmd/generate-logo/    Logo, PNG, and ICO generation
internal/audio/       musicex, QMC2, and TEA
internal/qqmusic/     Read-only session access and resource-key requests
internal/media/       Tool cache and media processes
internal/convert/     Planning, conversion, verification, and publication
internal/desktop/     Native Win32 interface
docs/                 Publishing and maintenance documentation
third_party/          Pinned media source, toolchain, and patch hashes
patches/              FFmpeg Windows build patch
licenses/             Original third-party license texts
scripts/              Build scripts
```

The generated `build/`, `dist/`, and media archives are excluded by `.gitignore`.
See the [GitHub publishing guide](docs/GITHUB.md) for the source upload workflow and release checks
(the guide is currently in Chinese).

## Changes

**1.1 · 2026-10-11**

- Fixed incorrect key selection when cached audio has an appended footer for another resource.
  Consecutive footers are validated, the innermost original resource is selected, and all
  footer bytes are excluded from the encrypted audio range.
- Retained PCM checksum, sample-count, and full MP3 decoding checks.
- Added the application logo, native maximization, and window resizing.

## Attribution and licensing

The original Go application code, documentation, and original logo use the [MIT License](LICENSE).
Third-party work retains its own terms; see [NOTICE.md](NOTICE.md) and [licensing scope](docs/LICENSING.en.md).
The project MIT license does not relicense bundled FFmpeg or other third-party work.

The media tools are built from pinned sources, with LAME as the only enabled external
audio encoder library. The release provides the Windows program, application sources, and
media Corresponding Source including the build patch, scripts, and version records.
See [distribution materials](docs/DISTRIBUTION.md) and [media build instructions](docs/MEDIA_BUILD.md).

| Component | Attribution and license |
|---|---|
| QMC2 / TEA algorithms | QM Unlock, pinned commit `c41a3e9affe67e01048d64fdd160a0002c913a30`, MIT |
| Go and golang.org/x/sys | BSD; original texts are retained in `licenses/` |
| Bundled FFmpeg / FFprobe | FFmpeg 8.1.1 built from pinned sources, GPLv3 or later |
| LAME 3.100 | LGPLv2 or later, linked statically in FFmpeg |

Third-party notices are also embedded in the EXE and can be exported using the help button.
Public distribution of an EXE containing FFmpeg must meet the applicable requirements for
corresponding source, build materials, and notices. A license file or upstream source URL alone
is not proof that all distribution obligations have been satisfied. Consult FFmpeg's official
licensing guidance and the original component licenses.
