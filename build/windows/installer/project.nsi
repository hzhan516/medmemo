Unicode true

!include "wails_tools.nsh"

VIProductVersion "${INFO_PRODUCTVERSION}.0"
VIFileVersion    "${INFO_PRODUCTVERSION}.0"

VIAddVersionKey "CompanyName"     "${INFO_COMPANYNAME}"
VIAddVersionKey "FileDescription" "${INFO_PRODUCTNAME} Installer"
VIAddVersionKey "ProductVersion"  "${INFO_PRODUCTVERSION}"
VIAddVersionKey "FileVersion"     "${INFO_PRODUCTVERSION}"
VIAddVersionKey "LegalCopyright"  "${INFO_COPYRIGHT}"
VIAddVersionKey "ProductName"     "${INFO_PRODUCTNAME}"

ManifestDPIAware true

!include "MUI.nsh"
!include "LogicLib.nsh"

!define MUI_ICON "..\icon.ico"
!define MUI_UNICON "..\icon.ico"
!define MUI_FINISHPAGE_NOAUTOCLOSE
!define MUI_ABORTWARNING

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

!insertmacro MUI_UNPAGE_INSTFILES

!insertmacro MUI_LANGUAGE "English"

Name "${INFO_PRODUCTNAME}"
OutFile "..\..\bin\${INFO_PROJECTNAME}-${ARCH}-installer.exe"
InstallDir "$PROGRAMFILES64\${INFO_COMPANYNAME}\${INFO_PRODUCTNAME}"
ShowInstDetails show

Function .onInit
   !insertmacro wails.checkArchitecture
FunctionEnd

Section
    !insertmacro wails.setShellContext

    !insertmacro wails.webview2runtime

    SetOutPath $INSTDIR

    !insertmacro wails.files

    SetOutPath "$INSTDIR\resources\rules"
    File /r "..\..\..\resources\rules\*.*"
    SetOutPath "$INSTDIR\resources\models"
    File /r "..\..\..\resources\models\*.*"
    SetOutPath "$INSTDIR\resources\lib\windows"
    File "..\..\..\resources\lib\windows\*.dll"
    SetOutPath $INSTDIR

    CreateShortcut "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"
    CreateShortCut "$DESKTOP\${INFO_PRODUCTNAME}.lnk" "$INSTDIR\${PRODUCT_EXECUTABLE}"

    !insertmacro wails.associateFiles
    !insertmacro wails.associateCustomProtocols

    ; 记录安装路径到注册表，供自动更新器识别原安装目录
    UserInfo::GetAccountType
    Pop $0
    ${If} $0 == "Admin"
        WriteRegStr HKLM "Software\MedMemo" "InstallPath" $INSTDIR
    ${Else}
        WriteRegStr HKCU "Software\MedMemo" "InstallPath" $INSTDIR
    ${EndIf}

    !insertmacro wails.writeUninstaller

    ; 安装完成后启动新版本
    Exec '"$INSTDIR\${PRODUCT_EXECUTABLE}"'
SectionEnd

Section "uninstall"
    !insertmacro wails.setShellContext

    RMDir /r "$AppData\${PRODUCT_EXECUTABLE}"

    RMDir /r $INSTDIR

    Delete "$SMPROGRAMS\${INFO_PRODUCTNAME}.lnk"
    Delete "$DESKTOP\${INFO_PRODUCTNAME}.lnk"

    !insertmacro wails.unassociateFiles
    !insertmacro wails.unassociateCustomProtocols

    ; 清理安装路径注册表
    UserInfo::GetAccountType
    Pop $0
    ${If} $0 == "Admin"
        DeleteRegKey HKLM "Software\MedMemo"
    ${Else}
        DeleteRegKey HKCU "Software\MedMemo"
    ${EndIf}

    !insertmacro wails.deleteUninstaller
SectionEnd
