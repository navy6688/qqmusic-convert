# v1.1.0 release notes — preparation

Status: local preview; public binary distribution is pending complete third-party
source/build/license materials. No GitHub Release or Git tag has been created.

## Changes

- Fixed incorrect key selection when consecutive musicex footers refer to
  different cached/alias resources; use the innermost original resource.
- Added the music/conversion-arrow logo to the executable and native interface.
- Enabled window maximization and resizing.
- Preserved source checksums, FLAC PCM integrity checks, strict MP3 decoding,
  cancellation cleanup, and no-overwrite publication.
- Added bilingual project documentation and MIT licensing for original work.

## Runtime

Windows 10/11 x64. A single executable with bundled media tools. Python, Go,
external FFmpeg installation and WebView2 are not end-user runtime requirements.
Encrypted audio requires a running, signed-in QQ Music desktop client with access
to the resource. Existing MP3 outputs are skipped, and original files remain intact.

## Licensing and source

Original application work is MIT; third-party works retain their own terms.
The current bundled FFmpeg build is GPLv3. Consult LICENSE, NOTICE.md,
THIRD_PARTY_NOTICES.txt, SOURCE_AVAILABILITY.md and docs/DISTRIBUTION.md.
Do not label the complete preview executable "MIT only" or publicly distribute
this preview as a completed corresponding-source release.

## Artifacts

Actual filenames, versions, sizes, hashes and release blockers are recorded in
RELEASE_MANIFEST.json and SHA256SUMS.txt. Verify the checksums before using files.
The partial FFmpeg upstream source archive is explicitly labelled partial.
