# GitHub 同步与发布指南

本指南区分两个动作：**同步项目源码**和**发布可下载的 EXE**。
源码放在 Git 仓库；可执行文件、校验文件和发行材料放在 Releases。

## 1. 首次同步前准备

- 确认仓库名称，例如 `qqmusic-convert`，以及 Public / Private 可见性。
- 本项目原创内容已采用根目录 MIT `LICENSE`；保留 `NOTICE.md` 与第三方许可。公开 EXE 前先检查 `docs/DISTRIBUTION.md` 中的缺项。
- 保留 `go.mod`、`go.sum`、源代码、构建脚本和第三方许可。
- 不上传真实音频、账号配置、进程内存、登录令牌、密钥或包含个人路径的转换报告。
- 不上传 `build/`、`dist/`、临时编译器、内嵌媒体压缩包和生成的图标资源对象。
- 公开截图应使用合成音频和示例路径，不直接使用本地排查截图。

可以先使用私有仓库存放源码，等许可证和发行材料明确后，再决定是否公开。
`.gitignore` 能排除匹配的未跟踪文件，但不会移除已经提交的文件。
如果曾将真实凭据提交进历史，单纯删除文件不能清除泄露，需要撤销凭据并处理历史。

## 2. 在 GitHub 创建空仓库

1. 登录 GitHub，进入新建仓库页面。
2. 输入仓库名称，选择 Public 或 Private。
3. 首次从本地导入时，不勾选初始化 README、`.gitignore` 或 LICENSE，保持远程仓库为空。
4. 创建后复制仓库的 HTTPS 地址。

下面的命令假设远程是空仓库。如果远程已有提交，请先获取并检查远程历史，
不要使用强制推送来覆盖它。

## 3. 在项目根目录提交源码

在项目文件夹打开 PowerShell。下列步骤只需要执行一次：

```powershell
git init -b main

git add .gitignore LICENSE NOTICE.md README.md README.en.md go.mod go.sum cmd internal scripts licenses docs

git status --short
git diff --cached --stat
git diff --cached
```

检查暂存文件只包含源码和文档，没有音频、密钥、账号信息、大型二进制或私人报告，
再创建提交：

```powershell
git commit -m "Initial commit"
```

如果 Git 提示缺少提交身份，可只为当前仓库设置：

```powershell
git config user.name "YOUR_COMMIT_NAME"
git config user.email "YOUR_COMMIT_EMAIL"
```

将示例身份替换成你自己的提交名称和邮箱。若不想公开个人邮箱，可以使用 GitHub
账号设置中提供的 noreply 邮箱。这里没有设置全局 Git 配置。

## 4. 关联远程并推送

替换下面的 GitHub 用户名、仓库名和地址：

```powershell
git remote add origin https://github.com/YOUR_GITHUB_USERNAME/qqmusic-convert.git
git remote -v
git push -u origin main
```

HTTPS 认证可以使用 Git Credential Manager 的浏览器登录。也可以选择 SSH 远程地址，
并配置自己的 SSH 密钥；SSH 是另一种连接方式，不是 HTTPS 密码。
不要把令牌写进远程 URL、README 或脚本。GitHub 账号密码不能作为 Git 的 HTTPS 推送密码。

## 5. 后续更新

先查看状态。有其他电脑或协作者推送时，先处理自己的工作，再获取并检查远程变更。

```powershell
git status

git add README.md README.en.md docs
git diff --cached
git commit -m "Update documentation"
git push
```

源码变更时按需要把相应文件加入暂存区；不要把此示例的路径列表当作每次都固定的清单。
不要为解决冲突而默认使用 `git push --force`。

## 6. README 维护

- `README.md`：默认中文，面向项目主要使用者。
- `README.en.md`：英文，覆盖相同的功能、限制、构建流程和许可说明。
- 两份文档顶部提供语言切换，更新功能或限制时同步修改。
- README 介绍项目当前行为，不记录个人电脑目录、真实歌曲批次或临时排查过程。
- 英文文档不代表界面已有英文翻译；当前 GUI 仍为简体中文。

## 7. 发布 EXE

上传源码不代表必须立即公开 EXE。准备发布时：

1. 在干净的源代码检出中运行构建脚本和相关验证。
2. 记录 Go 工具链、依赖、FFmpeg 版本、配置、来源、哈希及第三方许可证。
3. 确认项目级许可证和第三方二进制的分发条件已经处理。
4. 用 Git 标签标记实际要发布的提交，例如 `v1.1.0`。
5. 在 GitHub 创建对应 Release，上传 EXE、`SHA256.txt`、双语说明和所需发行材料。

示例标签命令只在确认提交已完成验证后执行：

```powershell
git tag -a v1.1.0 -m "Release v1.1.0"
git push origin v1.1.0
```

当前 EXE 约 92 MB，版本库不应承担二进制发行和存档职责。将其放进 Releases，
避免后续每次构建都让 Git 历史继续增大。不要提交源目录中的媒体压缩资源来绕过这一点。

### FFmpeg 发行材料

当前构建内置的是启用 GPLv3 的 FFmpeg / FFprobe。公开分发前，应按适用许可准备
对应版本的源码、所用第三方库源码、构建脚本与配置，以及许可和版权声明。
仅提供上游网站首页或许可文本，不代表已经履行全部对应源码要求。

应依据具体二进制、其构建选项和许可证判断分发要求，不把当前 FFmpeg 构建标为 LGPL，
也不未经评估就把整个应用声明为某个开源许可证。本指南不代替对具体发行包的许可审查。

## 官方参考

- GitHub：将本地代码添加到 GitHub
  https://docs.github.com/en/migrations/importing-source-code/using-the-command-line-to-import-source-code/adding-locally-hosted-code-to-github
- GitHub：大文件与发行资产
  https://docs.github.com/en/repositories/working-with-files/managing-large-files/about-large-files-on-github
- GitHub：缓存 GitHub 凭据
  https://docs.github.com/en/get-started/git-basics/caching-your-github-credentials-in-git
- GitHub：仓库许可证
  https://docs.github.com/en/repositories/managing-your-repositorys-settings-and-features/customizing-your-repository/licensing-a-repository
- FFmpeg：许可说明
  https://ffmpeg.org/legal.html
