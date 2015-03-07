;--------------------------------
;Include Modern UI

  !include "MUI2.nsh"

;--------------------------------
;General

  ;Name and file
  Name "Mist"
  OutFile "mist-setup.exe"

  ;Default installation folder
  InstallDir "$PROGRAMFILES\Mist"

  ;Get installation folder from registry if available
  InstallDirRegKey HKCU "Software\Mist" ""

  ;Request application privileges for Windows Vista
  RequestExecutionLevel admin

  SetCompressor /SOLID lzma ; had the strongest compression rate for Mist

;--------------------------------
;Variables

;--------------------------------
;Interface Settings


  !define MUI_ICON "logo.ico"
  !define MUI_HEADERIMAGE
  !define MUI_HEADERIMAGE_BITMAP "ethereum.bmp"
  !define MUI_HEADERIMAGE_RIGHT
  !define MUI_ABORTWARNING

;--------------------------------
;Pages

  ;!insertmacro MUI_PAGE_LICENSE "tmp/LICENCE"
  ;!insertmacro MUI_PAGE_COMPONENTS
  !insertmacro MUI_PAGE_DIRECTORY

  ;Start Menu Folder Page Configuration
  !define MUI_STARTMENUPAGE_REGISTRY_ROOT "HKCU"
  !define MUI_STARTMENUPAGE_REGISTRY_KEY "Software\Mist"
  !define MUI_STARTMENUPAGE_REGISTRY_VALUENAME "Start Menu Folder"

  ;!insertmacro MUI_PAGE_STARTMENU Application $StartMenuFolder

  !insertmacro MUI_PAGE_INSTFILES

  !insertmacro MUI_UNPAGE_CONFIRM
  !insertmacro MUI_UNPAGE_INSTFILES

;--------------------------------
;Languages

  !insertmacro MUI_LANGUAGE "English"

;--------------------------------
;Installer Sections

Section

  SetOutPath "$INSTDIR"

  file /r $%GOPATH%\pkg\ethereum\*.*
  file logo.ico

  ;Store installation folder
  WriteRegStr HKCU "Software\Mist" "" $INSTDIR

  ;Create uninstaller
  WriteUninstaller "$INSTDIR\Uninstall.exe"


  CreateShortCut "$DESKTOP\Mist.lnk" "$INSTDIR\mist.exe" "" "$INSTDIR\logo.ico" 0

  ;create start-menu items
  CreateDirectory "$SMPROGRAMS\Mist"
  CreateShortCut "$SMPROGRAMS\Mist\Uninstall.lnk" "$INSTDIR\Uninstall.exe" "" "$INSTDIR\Uninstall.exe" 0
  CreateShortCut "$SMPROGRAMS\Mist\Mist.lnk" "$INSTDIR\mist.exe" "" "$INSTDIR\logo.ico" 0

SectionEnd

;--------------------------------
;Descriptions

  ;Assign language strings to sections
  ;!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
  ;  !insertmacro MUI_DESCRIPTION_TEXT ${SecDummy} $(DESC_SecDummy)
  ;!insertmacro MUI_FUNCTION_DESCRIPTION_END

;--------------------------------
;Uninstaller Section

Section "Uninstall"

  ;ADD YOUR OWN FILES HERE...
  RMDir /r "$INSTDIR\*.*"

  RMDir "$INSTDIR"

  Delete "$DESKTOP\Mist.lnk"
  Delete "$SMPROGRAMS\Mist\*.*"
  RmDir  "$SMPROGRAMS\Mist"

  DeleteRegKey /ifempty HKCU "Software\Mist"

SectionEnd
