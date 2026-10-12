# Copyright, attribution, and license notices

The original qqmusic-convert application code, documentation, and original logo
are licensed under the MIT License. Copyright (c) 2026 qqmusic-convert contributors.
The root LICENSE contains the license text. Third-party works retain their own
copyrights and licenses; this notice does not relicense them.

## QMC2 / TEA implementation

Ported and adapted from QM Unlock:
https://github.com/mary20050520/qmunlock
Pinned reference commit: c41a3e9affe67e01048d64fdd160a0002c913a30.
The original MIT copyright and license notice is retained in licenses/QMUnlock-MIT.txt.
Affected code includes internal/audio/qmc.go and derived algorithm test vectors.

## Go runtime and standard library

The current executable was built using Go 1.26.2. Copyright (c) 2009 The Go Authors.
The BSD license is retained in licenses/Go-BSD.txt.
https://go.dev/

## golang.org/x/sys

Version v0.49.0; BSD license retained in licenses/xsys-BSD.txt.
https://pkg.go.dev/golang.org/x/sys

## Windows resource compiler (build tool)

github.com/akavel/rsrc v0.10.2. Copyright (c) 2013-2017 The rsrc Authors.
MIT notice retained in licenses/rsrc-MIT.txt. This is a build-time tool, not an
additional end-user runtime dependency.
https://github.com/akavel/rsrc

## Bundled FFmpeg / FFprobe

Current binaries: Gyan.dev 8.1.1-essentials_build-www.gyan.dev, Windows x64 static.
The publisher states GPL v3. Configuration includes --enable-gpl and --enable-version3.
Copyright (c) 2000-2026 the FFmpeg developers for FFmpeg; FFprobe identifies its
copyright range as 2007-2026. GPL text is retained in licenses/FFmpeg-GPL.txt.

Publisher-declared FFmpeg source commit:
239f2c733de417201d7ad3b3b8b0d9b63285b2b1.
https://github.com/FFmpeg/FFmpeg/commit/239f2c733de417201d7ad3b3b8b0d9b63285b2b1
Publisher and official guidance:
https://www.gyan.dev/ffmpeg/builds/
https://ffmpeg.org/legal.html

FFmpeg tools are invoked as separate processes, and their compressed binaries
are included in the executable. This description does not determine the legal
classification of every distributed combination. The project MIT license does
not replace FFmpeg's GPL or the licenses of its linked dependencies.

The current publisher's static build enables many external libraries. Their
recorded versions and unresolved source/build/license items are listed in the
release-materials inventory. A complete public binary distribution requires
meeting the applicable Corresponding Source and notice conditions; neither this
NOTICE nor the downloaded upstream FFmpeg archive alone establishes completeness.

See docs/LICENSING.md and docs/DISTRIBUTION.md for scope and release status.
No third-party endorsement or rights to music recordings are granted here.
