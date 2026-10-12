# Version-specific source availability

Application source repository: https://github.com/navy6688/qqmusic-convert
Release: v1.1.0.

The release provides application sources and the matching
`qqmusic-convert-v1.1.0-corresponding-source.zip` next to the Windows binary.
The source bundle includes:

- The original Go application, build scripts, public synthetic test vectors,
  original MIT LICENSE, NOTICE, and third-party license texts.
- Original FFmpeg source at 239f2c733de417201d7ad3b3b8b0d9b63285b2b1.
- Original LAME 3.100 source, the only enabled external audio encoder library.
- The FFmpeg Windows archive-response-file patch and the actual build scripts.
- Exact LLVM 23.1.3 and MinGW-w64 sources, plus the pinned LLVM-MinGW build scripts.
- Version, configuration, source/input hashes, and media binary hashes.

Archives are pinned by third_party/media-sources.json. Build instructions are in
docs/MEDIA_BUILD.md. The pinned compiler binary can be downloaded free from its
recorded official release URL and verified by its recorded SHA-256. General
build tools are not installed as end-user runtime dependencies.

The executables import only Windows system DLLs. No Windows DLL is redistributed
inside the release. Runtime notices remain included; the exact compiler/runtime
sources are supplied in addition to the FFmpeg and LAME application sources.

This release no longer embeds the previous general-purpose Gyan binary whose
full external dependency build materials were not available. The source bundle
is matched to the media profile actually built for this version, not that old
binary. The original source archives remain intact; the applied build patch is
provided separately.

Version-specific binary and source access is supplied together using the
applicable GPLv3 section 6 network distribution option. No separate written
source-offer commitment is asserted. Retain the source assets while distributing
the matching binary, and keep notices with redistributed copies.

No real audio, account data, login tokens, song keys, or personal computer paths
are provided as public source fixtures. This inventory records the materials
provided and does not replace legal assessment of any other distribution.
