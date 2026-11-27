# 有盼 · Youpan

有盼是一款 Mac 和 Windows 桌面应用，把值得期待的日子、每天的工作安排和随手记录放在一起。

## 功能

- **倒计时**：五张倒计时卡片，默认是退休、寒假、暑假、五一和十一；名称和目标日期可以按自己的需要修改。
- **今天与未来七天**：快速添加、搜索、星标、按天完成待办，支持每天、周一至周五和每周重复。
- **日历**：双击日期即可新建待办并自动填写当天；单击查看该日的安排，也可以点「新建待办」。
- **笔记**：保存、搜索文字记录，并生成有关联的今日待办。
- **外观**：直接拖动窗口边角调整大小；窗口透明度可在 50%—100% 之间调节，设置保存在本机。
- **运行方式**：窗口可置顶；关闭窗口后留在 Mac 菜单栏或 Windows 托盘，完全退出后停止提醒。

## 使用

倒计时卡片用于修改日期；统一的设置入口用于修改五项倒计时名称、窗口透明度。名称修改不会更改已保存的目标日期或待办。

提醒需要应用保持运行。工作日规则表示周一至周五，不包含中国法定节假日和周末调休规则。数据保存在本机，不需要注册应用账号。

详细操作见 [使用说明](使用说明.md)。正式安装文件见仓库的 [Releases](https://github.com/yongyuan2008/youpan/releases)；源码下载包不包含个人倒计时、待办或笔记数据。

## 从源码构建

### macOS

需要 macOS、Xcode Command Line Tools，以及可编译 SwiftUI 的 macOS SDK。

```sh
./mac/build.sh
```

脚本先在隔离目录运行核心测试，再生成包含 Apple Silicon 与 Intel 架构的 `mac/build/有盼.app`，使用本地签名。最低运行系统为 macOS 13。

### Windows x64

需要 Go 1.21 或更新版本和 NSIS 3。仓库包含 Windows 图标资源对象；如果机器上的 LLVM resource tools 可用，也可以从 `installer/resources.rc` 再生成。

```sh
./build-windows.sh
```

也可以在 Windows PowerShell 中先构建应用，再打包：

```powershell
cd windows
go test ./...
$env:GOOS = "windows"
$env:GOARCH = "amd64"
$env:CGO_ENABLED = "0"
go build -trimpath -ldflags="-H=windowsgui -s -w" -o ../installer/DaysCountdown.exe .
cd ../installer
makensis setup.nsi
```

Windows 安装程序安装到当前用户目录，不要求管理员权限。旧版本数据目录继续使用。

## 数据与升级

- macOS：`~/Library/Application Support/DaysCountdown/`
- Windows：`%LOCALAPPDATA%\DaysCountdown\`

`data.json` 保存倒计时日期和置顶状态，`planner.json` 保存任务、笔记与提醒记录，`appearance.json` 保存名称和窗口外观。备份前先完全退出有盼，然后复制整个数据目录。

## 验证范围

源码包含倒计时、重复任务、笔记关联、提醒去重和存储保护的核心测试。已提供 Mac 通用架构与 Windows x64 构建。Windows 托盘、安装运行和不同显示缩放比例仍需 Windows 实机验收；本地签名不等于商业开发者签名或系统公证。
