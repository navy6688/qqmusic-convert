#!/usr/bin/env bash
set -Eeuo pipefail

if [[ $# -lt 3 || $# -gt 4 ]]; then
    printf 'Usage: %s SOURCE_ROOT LLVM_MINGW_ROOT OUTPUT_ROOT [JOBS]\n' "$0" >&2
    exit 2
fi
repository_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
source_root="$(cygpath -au "$1")"
toolchain_root="$(cygpath -au "$2")"
output_root="$(cygpath -au "$3")"
jobs="${4:-4}"
if [[ ! "$jobs" =~ ^[1-9][0-9]*$ ]]; then
    printf 'Invalid build job count: %s\n' "$jobs" >&2
    exit 2
fi
ffmpeg_source="$source_root/FFmpeg-239f2c733d"
lame_source="$source_root/lame-3.100"
for required in "$ffmpeg_source/configure" "$lame_source/configure" "$toolchain_root/bin/x86_64-w64-mingw32-clang.exe" "$toolchain_root/bin/mingw32-make.exe"; do
    if [[ ! -f "$required" ]]; then
        printf 'Required build file is missing: %s\n' "$required" >&2
        exit 1
    fi
done
export PATH="$toolchain_root/bin:/usr/bin:/bin:$PATH"
# Source archives have no .git; prevent FFmpeg from reading the application repository version.
GIT_CEILING_DIRECTORIES="$(cygpath -m "$output_root")"
export GIT_CEILING_DIRECTORIES
compiler='x86_64-w64-mingw32-clang'
builder='mingw32-make'
build_shell="$(cygpath -m -s "$(command -v sh)")"
if [[ "$build_shell" == *' '* ]]; then
    printf 'Git shell needs a Windows short path without spaces: %s\n' "$build_shell" >&2
    exit 1
fi
mkdir -p "$output_root/work/lame" "$output_root/work/ffmpeg" "$output_root/prefix/include/lame" "$output_root/prefix/lib" "$output_root/bin"
trap 'printf "Media build failed at line %s; inspect the retained build directories.\n" "$LINENO" >&2' ERR

cd "$output_root/work/lame"
lame_relative="$(realpath --relative-to="$PWD" "$lame_source")"
CC="$compiler" AR=x86_64-w64-mingw32-ar RANLIB=x86_64-w64-mingw32-ranlib \
    bash "$lame_relative/configure" --host=x86_64-w64-mingw32 --prefix=/qqmusic-convert-media \
    --disable-shared --enable-static --disable-frontend --disable-decoder --disable-nasm CFLAGS=-O2
"$builder" -j "$jobs" SHELL="$build_shell" MAKE="$builder"
cp libmp3lame/.libs/libmp3lame.a "$output_root/prefix/lib/"
cp "$lame_source/include/lame.h" "$output_root/prefix/include/lame/"

# Native MinGW Make requires relative source paths; configure a work copy in place.
cp -a "$ffmpeg_source/." "$output_root/work/ffmpeg/"
cd "$output_root/work/ffmpeg"
patch -p1 < "$repository_root/patches/ffmpeg-native-make.patch"
bash ./configure \
    --prefix=/qqmusic-convert-media --target-os=mingw32 --arch=x86_64 --enable-cross-compile \
    --cc="$compiler" --ar=x86_64-w64-mingw32-ar --ranlib=x86_64-w64-mingw32-ranlib \
    --nm=x86_64-w64-mingw32-nm --strip=x86_64-w64-mingw32-strip \
    --disable-autodetect --disable-debug --disable-doc --disable-network --disable-x86asm \
    --disable-everything --enable-static --disable-shared --enable-gpl --enable-version3 \
    --enable-ffmpeg --enable-ffprobe --disable-ffplay --enable-avdevice \
    --disable-pthreads --enable-w32threads --enable-decoders --enable-demuxers --enable-parsers \
    --enable-libmp3lame --enable-encoder=libmp3lame,flac,pcm_s8,pcm_u8,pcm_s16le,pcm_s24le,pcm_s32le,pcm_s16be,pcm_s24be,pcm_s32be,pcm_f32le,pcm_f64le \
    --enable-muxer=mp3,flac,wav,ogg,null,pcm_s8,pcm_s16le,pcm_s24le,pcm_s32le \
    --enable-protocol=file,pipe --enable-indev=lavfi --enable-filter=sine,anullsrc,aresample,aformat,anull \
    --extra-cflags=-I../../prefix/include --extra-ldflags='-L../../prefix/lib -static'
"$builder" -j "$jobs" SHELL="$build_shell" MAKE="$builder"
cp ffmpeg.exe ffprobe.exe "$output_root/bin/"
"$output_root/bin/ffmpeg.exe" -version
"$output_root/bin/ffprobe.exe" -version
