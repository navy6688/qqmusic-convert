# 媒体组件源码构建 / Media build

本版使用 FFmpeg 8.1.1 的固定提交与 LAME 3.100 原始源码，自行生成 Windows x64
FFmpeg / FFprobe。第三方源文件和编译工具链的哈希见 `third_party/media-sources.json`。
唯一启用的外部编码库是 LAME；其余解码和封装来自 FFmpeg 自身源码。

## 在 Windows 上构建

需要 Go 1.26+、Git for Windows（包含 Bash）和可用的下载网络。
默认脚本会下载固定源码和 LLVM-MinGW 20261006，核对每个 SHA-256 后构建。
这些依赖仅用于构建；最终用户无需安装它们。

```powershell
& ".\scripts\build.ps1"
```

只构建媒体工具：

```powershell
& ".\scripts\build-media.ps1" -Jobs 8
```

若已下载发行页的对应源码包，可复用其中的档案目录：

```powershell
& ".\scripts\build-media.ps1" -Archives "..\archives" -Jobs 8
& ".\scripts\build.ps1" `
  -FFmpeg ".\build\release-media\bin\ffmpeg.exe" `
  -FFprobe ".\build\release-media\bin\ffprobe.exe"
```

若网络需要代理，在当前终端配置标准 `https_proxy` / `http_proxy` 环境变量。
脚本不修改全局 Git、系统网络设置或凭据。已有档案必须匹配固定哈希，不匹配会停止。

## 构建步骤与补丁

1. 校验源码、工具链和补丁的 SHA-256。
2. 编译 LAME 静态编码库，不安装前端程序。
3. 在生成目录中复制 FFmpeg 源码，应用 `patches/ffmpeg-native-make.patch`。
4. 用本机 GNU Make 构建两个静态媒体程序，关闭外部自动探测和网络协议。
5. 记录版本、配置、二进制哈希和构建输入，输出 `build/release-media/build-receipt.json`。

补丁只修改归档响应文件的生成方法，使用 GNU Make 的 `file` 函数，避免 Windows
原生命令行长度限制。它不修改音频算法。该补丁按 FFmpeg 的 GPL 条款提供。

原生 MinGW Make 调用 Git shell 时使用 Windows 短路径。Git 安装路径若含空格且
没有可用的短路径，脚本会明确停止；可将 Git 安装在不含空格的路径后重试。
普通音乐路径没有因此被限制。

## 对应源码与运行库

发行源码包保留 FFmpeg、LAME、LLVM 项目、MinGW-w64 及 LLVM-MinGW 构建脚本的
原始档案，同时提供应用源码、补丁、构建脚本和版本记录。未对原始档案重新打包。
Windows UCRT 和系统 DLL 由操作系统提供；PE 依赖检查不允许额外第三方 DLL。
LLVM / MinGW 运行库的原始版权声明保留在 `licenses/`。

不承诺逐字节可复现。可核对的范围是固定输入源码、工具链档案、脚本、配置、
功能验证与本次实际二进制的校验值。

## English

Build on Windows with Go 1.26+ and Git for Windows including Bash. The default
application build downloads and verifies pinned archives, builds FFmpeg/LAME,
then embeds the two media executables. LLVM-MinGW 20261006 is the pinned compiler
and Make distribution; all source/toolchain hashes are in the lock file.

The corresponding-source bundle contains an `application/` directory and an
`archives/` directory. From `application/`, run the archive-reuse commands above.
The build-only FFmpeg patch uses GNU Make response-file generation. Original
source archives, the patch, compiler/runtime source origins, notices, and version
records are provided. Windows system DLLs are supplied by the operating system.
No bit-for-bit reproducibility claim is made.
