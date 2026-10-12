# v1.1.0 发行材料

本版媒体组件由固定源码自行构建，替换此前资料不齐备的通用 Gyan 二进制。
启用的唯一外部音频编码库为 LAME 3.100，FFmpeg / FFprobe 版本为 8.1.1。
应用原创内容仍为 MIT；FFmpeg 程序为 GPLv3 或后续版本，第三方版权分别保留。

## 发行文件

| 文件 | 内容 |
|---|---|
| qqmusic-convert-v1.1.0-windows-amd64.zip | EXE、中英文说明、截图、许可证、来源与校验文件 |
| qqmusic-convert-v1.1.0-source.zip | 本版本应用源码、脚本、补丁、锁定清单与文档 |
| qqmusic-convert-v1.1.0-corresponding-source.zip | 应用源码及媒体组件源码、运行库来源、补丁、构建步骤和版本记录 |
| RELEASE_MANIFEST.json | 版本、提交、许可范围、构建输入、实际二进制与资产哈希 |
| SHA256SUMS.txt | 发行资产的 SHA-256 |

对应源码与二进制在同一版本的 GitHub Release 中提供。
源码范围见 [SOURCE_AVAILABILITY.md](SOURCE_AVAILABILITY.md)，重建步骤见
[MEDIA_BUILD.md](MEDIA_BUILD.md)，许可原文见 `licenses/`。

## 构建来源

- FFmpeg：固定提交 239f2c733de417201d7ad3b3b8b0d9b63285b2b1。
- LAME：3.100 原始发布档案，未修改源码。
- FFmpeg 构建补丁：使用 GNU Make 原生写入归档响应文件，避开 Windows 命令行长度限制。
- 编译器：LLVM-MinGW 20261006，Clang 23.1.3、GNU Make 4.4.1。
- LLVM、MinGW-w64 和工具链构建来源：原始档案与完整修订值均随包记录。
- 动态依赖：只允许 Windows 自带系统 DLL，不携带其他第三方运行 DLL。

全部档案、工具链和补丁哈希见 `third_party/media-sources.json`。
实际组件哈希与构建结果见发行清单和配套构建记录。

## 发布核对

发布前必须核对源代码提交、程序功能、格式覆盖、许可证原文、PE 依赖、资产内容和哈希。
既有音频、认证数据、私人报告、开发工具二进制与个人路径不进入发行包。
不是只附一份上游源码来替代实际静态组件的对应源码材料。

本次不声称二进制逐字节可复现，也不把整个 EXE 统一称为“仅 MIT”。
程序未进行商业代码签名；以实际 Release 来源和校验值核对下载文件。
