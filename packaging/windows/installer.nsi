; M365Kit Windows Installer (NSIS)
; Build with: makensis installer.nsi
; Requires NSIS 3.x with EnVar plugin

!define APP_NAME "M365Kit"
!define APP_VERSION "0.4.0"
!define APP_PUBLISHER "KLYTICS LLC"
!define APP_URL "https://github.com/monykiss/m365kit"
!define EXE_NAME "kit.exe"
!define INSTALL_DIR "$PROGRAMFILES64\M365Kit"

Name "${APP_NAME} ${APP_VERSION}"
OutFile "M365Kit-${APP_VERSION}-Setup.exe"
InstallDir "${INSTALL_DIR}"
RequestExecutionLevel admin

; UI
!include "MUI2.nsh"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_LANGUAGE "English"

Section "M365Kit (required)" SecMain
  SectionIn RO
  SetOutPath "$INSTDIR"

  ; Copy binary
  File "${EXE_NAME}"

  ; Add to system PATH
  EnVar::SetHKLM
  EnVar::AddValue "PATH" "$INSTDIR"

  ; Write uninstall registry keys
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\M365Kit" \
    "DisplayName" "${APP_NAME}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\M365Kit" \
    "UninstallString" "$\"$INSTDIR\uninstall.exe$\""
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\M365Kit" \
    "Publisher" "${APP_PUBLISHER}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\M365Kit" \
    "DisplayVersion" "${APP_VERSION}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\M365Kit" \
    "URLInfoAbout" "${APP_URL}"
  WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\M365Kit" \
    "DisplayIcon" "$INSTDIR\${EXE_NAME}"
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\M365Kit" \
    "NoModify" 1
  WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\M365Kit" \
    "NoRepair" 1

  ; Create uninstaller
  WriteUninstaller "$INSTDIR\uninstall.exe"

  ; Create Start Menu shortcut
  CreateDirectory "$SMPROGRAMS\${APP_NAME}"
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\M365Kit Terminal.lnk" "cmd.exe" '/k "$INSTDIR\${EXE_NAME}" --help'
  CreateShortcut "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk" "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
  ; Remove files
  Delete "$INSTDIR\${EXE_NAME}"
  Delete "$INSTDIR\uninstall.exe"
  RMDir "$INSTDIR"

  ; Remove from PATH
  EnVar::SetHKLM
  EnVar::DeleteValue "PATH" "$INSTDIR"

  ; Remove registry keys
  DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\M365Kit"

  ; Remove Start Menu
  Delete "$SMPROGRAMS\${APP_NAME}\M365Kit Terminal.lnk"
  Delete "$SMPROGRAMS\${APP_NAME}\Uninstall.lnk"
  RMDir "$SMPROGRAMS\${APP_NAME}"
SectionEnd
