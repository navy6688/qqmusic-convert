# Copyright, attribution, and license notices

The original qqmusic-convert Go application, build scripts, documentation, and
original logo are licensed under the MIT License. Copyright (c) 2026
qqmusic-convert contributors. Third-party works retain their own terms.
The project MIT license does not relicense the bundled FFmpeg executables.

## QMC2 / TEA implementation

Ported and adapted from QM Unlock, pinned reference commit
c41a3e9affe67e01048d64fdd160a0002c913a30:
https://github.com/mary20050520/qmunlock
The original MIT copyright and license are retained in licenses/QMUnlock-MIT.txt.
Affected code includes internal/audio/qmc.go and derived algorithm test vectors.

## Go runtime and golang.org/x/sys

Built using Go 1.26.2; copyright (c) 2009 The Go Authors.
golang.org/x/sys is pinned to v0.49.0. Their BSD notices are retained in
licenses/Go-BSD.txt and licenses/xsys-BSD.txt.
https://go.dev/
https://pkg.go.dev/golang.org/x/sys

## Windows resource compiler

github.com/akavel/rsrc v0.10.2 is a build tool, not a runtime dependency.
Copyright (c) 2013-2017 The rsrc Authors. Its MIT notice is retained in
licenses/rsrc-MIT.txt.
https://github.com/akavel/rsrc

## FFmpeg / FFprobe

This release builds FFmpeg 8.1.1 from upstream commit
239f2c733de417201d7ad3b3b8b0d9b63285b2b1.
Copyright (c) 2000-2026 the FFmpeg developers; FFprobe identifies 2007-2026.
Configuration enables GPL and version 3. These executables are GPL version 3 or
later; the original GPL text is retained in licenses/FFmpeg-GPL.txt.
https://github.com/FFmpeg/FFmpeg/commit/239f2c733de417201d7ad3b3b8b0d9b63285b2b1
https://ffmpeg.org/legal.html

The only enabled external audio library is the statically linked LAME encoder.
Native decoders/demuxers are built from FFmpeg's own source. Network protocols,
external autodetection, and FFplay are disabled. FFmpeg tools are invoked as
separate processes; their compressed binaries are included in the application.
This description does not determine every distributed combination's legal scope.

The build applies patches/ffmpeg-native-make.patch, supplied under FFmpeg's
GPL terms, to generate archive response files with GNU Make on Windows. Original
sources, this patch, and the build scripts accompany the binary release.

## LAME

LAME 3.100, original upstream archive. Copyright notices remain in its source,
including Mark Taylor, Takehiro Tominaga, Robert Hegemann, Gabriel Bouvigne,
Alexander Leidinger, and other LAME contributors.
Its library is under the GNU Library General Public License version 2 or later.
The original text is retained in licenses/LAME-LGPL-2.0.txt. The combined FFmpeg
build is distributed under GPL version 3 or later with the corresponding sources.
https://lame.sourceforge.io/

## Compiler runtime and Windows C runtime support

LLVM-MinGW 20261006 provides Clang 23.1.3 and MinGW-w64 at commit
 a3d93999ef0521681d45e445c39964a7af0f593f.
LLVM compiler runtime notices are retained in licenses/LLVM-runtime.txt
(Apache License 2.0 with LLVM Exceptions and the additional included notices).
MinGW notices are retained in licenses/COPYING.MinGW-w64.txt and
licenses/COPYING.MinGW-w64-runtime.txt.
Exact upstream LLVM, MinGW, and toolchain build sources are supplied with this
release. Windows system DLLs remain supplied by the operating system.
https://github.com/mstorsjo/llvm-mingw/releases/tag/20261006

## Source availability

Pinned archives and hashes are recorded in third_party/media-sources.json.
The version-specific corresponding-source archive is supplied alongside the
Windows executable on the release page. It includes original sources, the
Windows build patch, build/install instructions, and version records.
See docs/SOURCE_AVAILABILITY.md, docs/MEDIA_BUILD.md, and docs/DISTRIBUTION.md.
No third-party endorsement or rights to music recordings are granted here.
