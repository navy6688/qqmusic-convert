# 项目许可证说明

[English](LICENSING.en.md)

## 原创内容：MIT

项目原创 Go 代码、文档和原创 Logo 采用根目录 `LICENSE` 中的 MIT 许可证。
版权署名使用 `qqmusic-convert contributors`，各贡献者保留其实际拥有的权利。
该署名不将第三方版权归入本项目，也不替贡献者确认其无权授权的内容。

MIT 允许使用、修改、复制、分发和商业使用，要求保留版权和许可声明，并包含无担保条款。
英文 `LICENSE` 是许可正文；本文件是阅读说明，没有添加“禁止商用”或其他额外限制。

音乐内容本身不由该软件许可证授权。工具面向用户有权访问的本地音频，
这项用途说明不应被写成与 MIT 正文冲突的软件许可条款。

## 第三方内容：分别保留许可

| 内容 | 许可 | 位置 |
|---|---|---|
| QM Unlock 派生的 QMC2 / TEA 与算法向量 | MIT，保留原作者声明 | `licenses/QMUnlock-MIT.txt` |
| Go 运行时和标准库 | BSD | `licenses/Go-BSD.txt` |
| golang.org/x/sys v0.49.0 | BSD | `licenses/xsys-BSD.txt` |
| rsrc v0.10.2 构建工具 | MIT | `licenses/rsrc-MIT.txt` |
| 当前内嵌 FFmpeg / FFprobe | 发布方标注 GPLv3；其链接库也需各自审查 | `licenses/FFmpeg-GPL.txt` |

`NOTICE.md` 记录来源、版本、版权及组件范围。根目录 MIT 许可证不会将 FFmpeg 或其他
第三方重新许可为 MIT。也不能只因保留了一份 GPL 文本，就宣布整个发行包已满足全部分发条件。

FFmpeg 在本项目中作为独立进程运行，但压缩后的二进制内嵌于 EXE。
本说明记录这种结构，不据此作出“所有组合必定是聚合”或“整个应用自动变成 GPL”的结论。

## 当前公开发行边界

- 原创源码和 MIT 许可文本已整理；同步源码时保留所有第三方声明。
- 当前 EXE 的发行材料已整理为本地预览包，尚不应标记为公开二进制发行就绪。
- 已取得发布方指明提交的 FFmpeg 源码，不等于已取得整个静态构建的完整对应源码。
- 外部依赖完整源码、补丁、构建脚本、工具链记录和依赖版权清单仍需核实。

具体待补项见 [EXE 发行材料](DISTRIBUTION.md)。确认这些项之前，不发布未落实内容的
源码提供承诺，也不使用“全部许可合规”作为说明。

## 官方资料

- MIT 许可正文：https://opensource.org/license/mit
- FFmpeg 许可说明：https://ffmpeg.org/legal.html
- GPLv3 正文：https://www.gnu.org/licenses/gpl-3.0.html

具体发行方式以适用许可证和实际软件组合为准；有争议的组合应接受专业许可审查。
