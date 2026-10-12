# 许可证范围

原创 Go 应用、构建脚本、文档和原创 Logo 使用根目录的 MIT 许可证。
第三方作品保留各自版权和许可；项目 MIT 不会把内置 FFmpeg 重新许可为 MIT。

| 内容 | 许可与范围 |
|---|---|
| 原创应用与文档 | MIT |
| QMC2 / TEA 移植与相关夹具 | 保留 QM Unlock 原始 MIT 声明 |
| Go 与 golang.org/x/sys | BSD |
| rsrc 资源编译工具 | MIT，仅用于构建 |
| 本次内置 FFmpeg / FFprobe | GPLv3 或后续版本 |
| 静态链接的 LAME 3.100 | GNU Library GPL v2 或后续版本；组合 FFmpeg 按 GPLv3 或后续版本分发 |
| LLVM 编译器运行库 | Apache 2.0 with LLVM Exceptions，保留随附声明 |
| MinGW 运行库与头文件 | 保留原始版权和许可文本，见 licenses/ |
| FFmpeg 构建补丁 | 随 FFmpeg 按其 GPL 条款提供 |

本次媒体组件由固定源码构建，不再使用依赖资料未齐备的通用 Gyan 二进制。
FFmpeg 对应源码、LAME 源码、构建补丁、脚本、工具链版本和运行库来源与二进制一起提供。
准确版本、配置、哈希和来源见 NOTICE.md、third_party/media-sources.json 与发行清单。

媒体工具经解压后作为独立进程运行。这是实现说明，不替代对分发组合的法律判断。
Windows 自带 DLL 和普通构建工具的范围按适用许可证区分；运行库版权不因此被删除。
保留全部许可与声明，不把整个 EXE 描述为“仅 MIT”。

GPLv3 第 6 节的网络提供方式要求按版本提供可访问的对应源码。
发行页同时提供二进制与对应源码；没有另行承诺三年书面源码提供方式。
本项目许可不授予任何音乐录音、商标或第三方账号资源的权利。
