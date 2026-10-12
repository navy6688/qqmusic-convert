# QQ Music Convert v1.1.0

首次正式 Windows x64 发行。

## 功能

- 原生 Go 图形界面，可选择源目录和保存目录，显示逐个文件状态及总进度。
- 支持 musicex V1 的 MFLAC / MGG / MMP4 和常见普通音频，输出 320 kbps MP3。
- 修复连续多层 musicex 尾部选择了错误资源密钥的问题。
- 支持停止、查看详情、保存结果报告、窗口最大化与缩放。
- 保留原文件，跳过已有目标；输出经识别与完整解码后才发布。

## 本次发行准备

- 从固定源码构建 FFmpeg 8.1.1 / LAME 3.100，替换通用 Gyan 静态组件。
- 默认源码构建脚本自动验证源码和工具链哈希。
- 提供匹配的对应源码包、Windows 构建补丁、版本记录和完整许可文本。
- 运行时只依赖 Windows 系统 DLL，无需安装 Go、Python 或外部 FFmpeg。
- 中英文 README 包含真实界面截图，示例使用合成音频与演示路径。

Windows 10 / 11 x64。加密音频需要 QQ 音乐桌面客户端已登录且有资源访问权限。
GUI 当前为简体中文。EXE 未进行商业代码签名。

## English

First Windows x64 release. Native Go UI with folder selection, per-file status,
progress, cancellation, details, and reports. Fixes resource-key selection for
consecutive musicex footers and supports native window resizing/maximization.
Media tools are built from pinned FFmpeg/LAME sources. Matching source materials,
build patch/instructions, notices, manifest, and checksums accompany the binary.
Original audio and existing outputs are preserved. The GUI is Simplified Chinese.
