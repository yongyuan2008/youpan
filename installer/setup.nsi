Unicode true
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "x64.nsh"

Name "有盼"
OutFile "DaysCountdown-Setup-x64.exe"
InstallDir "$LOCALAPPDATA\Programs\DaysCountdown"
RequestExecutionLevel user
SetCompressor /SOLID lzma
ShowInstDetails nevershow
ShowUninstDetails nevershow
VIProductVersion "2.2.0.0"
VIAddVersionKey /LANG=2052 "ProductName" "有盼"
VIAddVersionKey /LANG=2052 "FileDescription" "有盼 · Windows 安装程序"
VIAddVersionKey /LANG=2052 "FileVersion" "2.2.0"
VIAddVersionKey /LANG=2052 "LegalCopyright" ""
!define APPKEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\DaysCountdown.Desktop"
!define MUI_ICON "Calendar.ico"
!define MUI_UNICON "Calendar.ico"
!define MUI_WELCOMEFINISHPAGE_BITMAP "welcome.bmp"
!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "安装有盼"
!define MUI_WELCOMEPAGE_TEXT "把值得期待的日子和每天要做的事，放在一起。$\r$\n$\r$\n倒计时、今日待办、日历、笔记，支持每天和工作日提醒。$\r$\n$\r$\n程序将安装到当前用户，并保留原有倒计时数据。"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_RUN "$INSTDIR\DaysCountdown.exe"
!define MUI_FINISHPAGE_RUN_TEXT "打开有盼"
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "SimpChinese"

!macro CheckRunning
  retry:
    FindWindow $0 "DaysCountdownMainWindow"
    StrCmp $0 0 done
    IfSilent cancel
    MessageBox MB_RETRYCANCEL|MB_ICONINFORMATION "请先退出有盼，再点击「重试」。$\r$\n2.0 版关闭窗口后仍会在托盘运行，请在托盘菜单选择「退出有盼」。" IDRETRY retry
  cancel:
    SetErrorLevel 2
    Abort
  done:
!macroend

Function .onInit
  SetShellVarContext current
  ${IfNot} ${RunningX64}
    IfSilent +2
    MessageBox MB_OK|MB_ICONSTOP "此安装包需要 64 位 Windows。"
    SetErrorLevel 1
    Abort
  ${EndIf}
  SetRegView 64
  !insertmacro CheckRunning
FunctionEnd

Function un.onInit
  SetShellVarContext current
  SetRegView 64
  !insertmacro CheckRunning
FunctionEnd

Section
  SetOutPath "$INSTDIR"
  SetOverwrite on
  ClearErrors
  File /oname=DaysCountdown.new "DaysCountdown.exe"
  IfErrors install_failed
  System::Call 'kernel32::MoveFileExW(w "$INSTDIR\DaysCountdown.new", w "$INSTDIR\DaysCountdown.exe", i 9) i.r0'
  StrCmp $0 0 install_failed
  WriteUninstaller "$INSTDIR\Uninstall.exe"
  ReadRegStr $0 HKCU "${APPKEY}" "DisplayName"
  ${If} $0 == "日子倒计时"
    Delete "$DESKTOP\日子倒计时.lnk"
    Delete "$SMPROGRAMS\日子倒计时.lnk"
  ${EndIf}
  CreateShortcut "$DESKTOP\有盼.lnk" "$INSTDIR\DaysCountdown.exe"
  CreateShortcut "$SMPROGRAMS\有盼.lnk" "$INSTDIR\DaysCountdown.exe"
  WriteRegStr HKCU "${APPKEY}" "DisplayName" "有盼"
  WriteRegStr HKCU "${APPKEY}" "DisplayVersion" "2.2.0"
  WriteRegStr HKCU "${APPKEY}" "DisplayIcon" "$INSTDIR\DaysCountdown.exe"
  WriteRegStr HKCU "${APPKEY}" "InstallLocation" "$INSTDIR"
  WriteRegStr HKCU "${APPKEY}" "UninstallString" '$\"$INSTDIR\Uninstall.exe$\"'
  WriteRegStr HKCU "${APPKEY}" "QuietUninstallString" '$\"$INSTDIR\Uninstall.exe$\" /S'
  WriteRegDWORD HKCU "${APPKEY}" "NoModify" 1
  WriteRegDWORD HKCU "${APPKEY}" "NoRepair" 1
  Goto finished
  install_failed:
    Delete "$INSTDIR\DaysCountdown.new"
    IfSilent +2
    MessageBox MB_ICONSTOP|MB_OK "无法写入程序。请关闭有盼，并确认安装文件夹可以写入。"
    SetErrorLevel 3
    Abort
  finished:
SectionEnd

Section "Uninstall"
  Delete "$DESKTOP\有盼.lnk"
  Delete "$SMPROGRAMS\有盼.lnk"
  Delete "$INSTDIR\DaysCountdown.exe"
  Delete "$INSTDIR\Uninstall.exe"
  RMDir "$INSTDIR"
  DeleteRegKey HKCU "${APPKEY}"
SectionEnd
