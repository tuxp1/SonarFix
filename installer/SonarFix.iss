#include "Version.iss"
[Setup]
AppId=SonarFix
AppName=SonarFix
AppVerName=SonarFix v{#AppVersion}
UninstallDisplayName=SonarFix v{#AppVersion}
UpdateUninstallLogAppName=yes
AppVersion={#AppVersion}
AppPublisher=SonarFix Project
AppComments=Independent Windows audio utility. Not affiliated with SteelSeries.
DefaultDirName={localappdata}\Programs\SonarFix
DefaultGroupName=SonarFix
PrivilegesRequired=lowest
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64compatible
MinVersion=10.0
; Same background as the SonarFix window; the native dark title bar matches it.
WizardStyle=modern dark hidebevels
WizardBackColor=#10151F
WizardSizePercent=120,120
DisableDirPage=yes
DisableProgramGroupPage=yes
DisableReadyPage=yes
DisableWelcomePage=no
UsePreviousTasks=no
CloseApplications=yes
RestartApplications=no
AppMutex={code:SettingsMutex}
UninstallDisplayIcon={app}\SonarFix.Settings.exe
SetupIconFile=..\SonarFix-source\winres\icon.ico
; The script hides the large side images and draws a centered logo instead.
WizardImageFile=images\wizard-large-*.png
WizardSmallImageFile=images\wizard-small-*.png
OutputDir=..\dist
OutputBaseFilename=SonarFix-Setup-{#AppVersion}
Compression=lzma2
SolidCompression=no
VersionInfoVersion={#NumericVersion}
VersionInfoTextVersion={#AppVersion}
VersionInfoProductVersion={#NumericVersion}
VersionInfoProductTextVersion={#AppVersion}
VersionInfoDescription=SonarFix Installer
VersionInfoCompany=SonarFix Project
SetupLogging=yes
ShowLanguageDialog=no
LanguageDetectionMethod=uilanguage

[Languages]
Name: "en"; MessagesFile: "compiler:Default.isl"
Name: "ar"; MessagesFile: "Arabic-LTR.isl"

[Files]
Source: "images\wizard-small-*.png"; Flags: dontcopy
Source: "..\dist\SonarFix.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\dist\SonarFix.Settings.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\README.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSE"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\THIRD_PARTY_NOTICES.md"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\LICENSES\*"; DestDir: "{app}\LICENSES"; Flags: ignoreversion recursesubdirs createallsubdirs

[InstallDelete]
; The legacy worker was SonarFix.Worker.exe. Task Manager lists Run-key
; entries by file name, so it is now SonarFix.exe; remove the old copy on upgrade.
Type: files; Name: "{app}\SonarFix.Worker.exe"
; The update source is now fixed inside SonarFix.Settings.exe.
Type: files; Name: "{app}\updates.json"

[Icons]
Name: "{group}\SonarFix"; Filename: "{app}\SonarFix.Settings.exe"

[Registry]
Root: HKCU; Subkey: "Software\SonarFix"; ValueType: string; ValueName: "InstallDir"; ValueData: "{app}"; Flags: uninsdeletevalue

[Run]
Filename: "{app}\SonarFix.Settings.exe"; Description: "{code:OpenSettings}"; Flags: postinstall nowait skipifsilent

[Code]
const
  // Colors of the SonarFix window (TColor is $BBGGRR).
  ColorBack = $1F1510;      // #10151F
  ColorCard = $30221A;      // #1A2230
  ColorField = $2D2017;     // #17202D
  ColorMuted = $CDB8AA;     // #AAB8CD
  ColorText = $FAF2ED;      // #EDF2FA
  ColorTeal = $717B26;      // #267B71
  ColorTealLight = $C8D671; // #71D6C8
  ColorTealDark = $585C24;  // #245C58
  ColorNeutral = $493325;   // #253349
  ColorTrack = $6A5346;     // #46536A
  ColorEdge = $9F8573;      // #73859F
  ColorDanger = $43368C;    // #8C3643
  OptEcho = 0; OptDisable = 1; OptRedirect = 2; OptStartup = 3; OptOpen = 4;
  ModeUpdate = 0; ModeModify = 1; ModeUninstall = 2;

var
  ConfigPage, MaintenancePage: TWizardPage;
  PathEdit: TNewEdit;
  BrowseButton: TBitmapButton;
  LocationLabel, OutputLabel, ExistingLabel: TNewStaticText;
  OutputCombo: TNewComboBox;
  OptOn: array[0..4] of Boolean;
  OptButton: array[0..4] of TBitmapButton;
  OptTitle: array[0..4] of TNewStaticText;
  OptDesc: array[0..1] of TNewStaticText;
  ModeButton: array[0..2] of TBitmapButton;
  ModeTitle, ModeDesc: array[0..2] of TNewStaticText;
  Mode: Integer;
  BackOverlay, NextOverlay, CancelOverlay, LanguageButton: TBitmapButton;
  WelcomeLogo, FinishedLogo: TBitmapImage;
  PrevInstalled, LegacyInstalled: Boolean;
  PrevVersion: String;
  UninstallingUI: Boolean;
  OutputIDs: TStringList;
  InfoFile: String;
  QuietClose: Boolean;
  UILanguage: String;
  BaseBack, BaseNext, BaseCancel, BasePageNameLeft: Integer;
  ConfigBottom: Integer;
  UITestDone, UINavigated: Boolean;

function OpenWaitProcess(Access: LongWord; Inherit: Boolean; PID: LongWord): THandle; external 'OpenProcess@kernel32.dll stdcall';
function WaitForProcess(H: THandle; Milliseconds: LongWord): LongWord; external 'WaitForSingleObject@kernel32.dll stdcall';
function CloseWaitHandle(H: THandle): Boolean; external 'CloseHandle@kernel32.dll stdcall';
function PostWizardMessage(Wnd: HWND; Msg: LongWord; WParam: NativeUInt; LParam: NativeInt): Boolean; external 'PostMessageW@user32.dll stdcall';

function HasArabic(Value: String): Boolean;
var I: Integer;
begin
  Result:=False;
  for I:=1 to Length(Value) do if (Ord(Value[I])>=$0600) and (Ord(Value[I])<=$06FF) then begin Result:=True;Exit;end;
end;

function Txt(English, Arabic: String): String;
begin
  // A Latin-only caption (such as "Switch to English") stays a plain LTR run;
  // an RTL embedding would reverse its word order.
  if (UILanguage = 'ar') and HasArabic(Arabic) then begin
    Result:=Arabic;
    StringChangeEx(Result,'SonarFix',#$200E+'SonarFix'+#$200E,True);
    StringChangeEx(Result,'Discord',#$200E+'Discord'+#$200E,True);
    StringChangeEx(Result,'Windows',#$200E+'Windows'+#$200E,True);
    // Keep the English pair in its own order inside Arabic text.
    StringChangeEx(Result,'Game && Chat',#$200E+'Game && Chat'+#$200E,True);
    StringChangeEx(Result,'Game & Chat',#$200E+'Game & Chat'+#$200E,True);
    StringChangeEx(Result,'Game',#$200E+'Game'+#$200E,True);
    StringChangeEx(Result,'Chat',#$200E+'Chat'+#$200E,True);
    // Alignment alone leaves an LTR paragraph with scrambled mixed text.
    Result:=#$202B+Result+#$202C;
  end else if UILanguage = 'ar' then Result:=Arabic
  else Result:=English;
end;

#include "Translations.iss"

function IsArabic: Boolean;
begin Result := UILanguage = 'ar'; end;

function Validating: Boolean;
begin Result := ExpandConstant('{param:VALIDATEONLY|0}') = '1'; end;

function IMax(A, B: Integer): Integer;
begin if A > B then Result := A else Result := B; end;

// Mirrors a horizontal position inside its container for Arabic.
function MX(Left, Width, Container: Integer): Integer;
begin if IsArabic then Result := Container-Left-Width else Result := Left; end;

function Isolated(Value: String): String;
begin Result := #$200E+Value+#$200E; end;

function ShowPrompt(Text: String; Typ: TMsgBoxType; Buttons: Integer): Integer;
var F: TSetupForm; L: TNewStaticText; A,B: TNewButton; W: Integer;
begin
  F:=CreateCustomForm(460,170,True,True);
  try
    F.Caption:='SonarFix';
    L:=TNewStaticText.Create(F);L.Parent:=F;L.AutoSize:=False;L.WordWrap:=True;L.ShowAccelChar:=False;
    L.SetBounds(ScaleX(20),ScaleY(18),F.ClientWidth-ScaleX(40),ScaleY(92));L.Caption:=Text;
    if IsArabic then L.Alignment:=taRightJustify;
    A:=TNewButton.Create(F);A.Parent:=F;A.Default:=True;
    B:=TNewButton.Create(F);B.Parent:=F;B.Cancel:=True;
    if Buttons=MB_YESNO then begin A.Caption:=Txt('Yes','نعم');A.ModalResult:=mrYes;B.Caption:=Txt('No','لا');B.ModalResult:=mrNo;end
    else begin A.Caption:=Txt('OK','حسنًا');A.ModalResult:=mrOk;A.Cancel:=True;B.Visible:=False;end;
    W:=F.CalculateButtonWidth([A.Caption,B.Caption]);
    A.SetBounds(F.ClientWidth-ScaleX(20)-W,F.ClientHeight-ScaleY(48),W,ScaleY(30));
    B.SetBounds(A.Left-ScaleX(8)-W,A.Top,W,A.Height);
    if IsArabic then begin A.Left:=ScaleX(20);B.Left:=A.Left+W+ScaleX(8);end;
    F.ActiveControl:=A;F.FlipAndCenterIfNeeded(False,nil,False);
    Result:=F.ShowModal;
    if Result=mrYes then Result:=IDYES else if Result=mrNo then Result:=IDNO;
  finally F.Free; end;
end;

function OpenSettings(Param: String): String;
begin Result := Tr('open_sonarfix_settings'); end;

function SettingsMutex(Param: String): String;
begin
  // Read-only layout validation does not install or need to close the UI.
  if Validating then Result:=''
  else Result:='Local\SonarFix_Settings';
end;

function InstalledPath: String;
begin
  if not RegQueryStringValue(HKCU, 'Software\SonarFix', 'InstallDir', Result) then
    Result := ExpandConstant('{localappdata}\Programs\SonarFix');
end;

// ---------------------------------------------------------------------------
// Drawing. Styled native controls ignore script colors, so buttons, switches,
// checkboxes and radio buttons are accessible TBitmapButtons painted here.
// TBitmapButton keeps a 2 pixel focus margin around its bitmap.

function NewSurface(B: TBitmapButton): TBitmap;
begin
  Result:=TBitmap.Create;Result.Width:=B.Width-4;Result.Height:=B.Height-4;
  Result.Canvas.Brush.Color:=ColorBack;Result.Canvas.Pen.Color:=ColorBack;
  Result.Canvas.Rectangle(0,0,Result.Width,Result.Height);
end;

procedure PaintButton(B: TBitmapButton; Fill, Ink: TColor; Bold: Boolean);
var M: TBitmap;
begin
  M:=NewSurface(B);
  try
    M.Canvas.Brush.Color:=Fill;M.Canvas.Pen.Color:=Fill;
    M.Canvas.RoundRect(0,0,M.Width,M.Height,ScaleX(12),ScaleY(12));
    M.Canvas.Font.Name:=WizardForm.NextButton.Font.Name;M.Canvas.Font.Height:=WizardForm.NextButton.Font.Height;
    M.Canvas.Font.Color:=Ink;
    if Bold then M.Canvas.Font.Style:=[fsBold] else M.Canvas.Font.Style:=[];
    M.Canvas.Brush.Style:=bsClear;
    M.Canvas.TextOut((M.Width-M.Canvas.TextWidth(B.Caption)) div 2,(M.Height-M.Canvas.TextHeight(B.Caption)) div 2,B.Caption);
    B.Bitmap:=M;
  finally M.Free; end;
end;

procedure PaintSwitch(B: TBitmapButton; On: Boolean);
var M: TBitmap; H, T, Y: Integer;
begin
  M:=NewSurface(B);
  try
    H:=M.Height;T:=H-ScaleY(8);Y:=(H-T) div 2;
    if On then begin M.Canvas.Brush.Color:=ColorTeal;M.Canvas.Pen.Color:=ColorTealLight;end
    else begin M.Canvas.Brush.Color:=ColorTrack;M.Canvas.Pen.Color:=ColorEdge;end;
    M.Canvas.RoundRect(0,0,M.Width,H,H,H);
    M.Canvas.Brush.Color:=clWhite;M.Canvas.Pen.Color:=clWhite;
    // The thumb moves right when on, in both languages, like the app.
    if On then M.Canvas.Ellipse(M.Width-ScaleX(4)-T,Y,M.Width-ScaleX(4),Y+T)
    else M.Canvas.Ellipse(ScaleX(4),Y,ScaleX(4)+T,Y+T);
    B.Bitmap:=M;
  finally M.Free; end;
end;

procedure PaintCheck(B: TBitmapButton; On: Boolean);
var M: TBitmap; S: Integer;
begin
  M:=NewSurface(B);
  try
    S:=M.Height;M.Canvas.Pen.Width:=IMax(1,ScaleX(1));
    if On then begin M.Canvas.Brush.Color:=ColorTeal;M.Canvas.Pen.Color:=ColorTeal;end
    else begin M.Canvas.Brush.Color:=ColorBack;M.Canvas.Pen.Color:=ColorEdge;end;
    M.Canvas.RoundRect(1,1,S-1,S-1,ScaleX(7),ScaleY(7));
    if On then begin
      M.Canvas.Pen.Color:=clWhite;M.Canvas.Pen.Width:=IMax(2,ScaleX(2));
      M.Canvas.MoveTo(S*26 div 100,S*52 div 100);M.Canvas.LineTo(S*43 div 100,S*69 div 100);M.Canvas.LineTo(S*75 div 100,S*31 div 100);
    end;
    B.Bitmap:=M;
  finally M.Free; end;
end;

procedure PaintRadio(B: TBitmapButton; On: Boolean);
var M: TBitmap; S, D: Integer;
begin
  M:=NewSurface(B);
  try
    S:=M.Height;M.Canvas.Pen.Width:=IMax(1,ScaleX(1));M.Canvas.Brush.Color:=ColorBack;
    if On then M.Canvas.Pen.Color:=ColorTealLight else M.Canvas.Pen.Color:=ColorEdge;
    M.Canvas.Ellipse(1,1,S-1,S-1);
    if On then begin
      D:=S div 4;M.Canvas.Brush.Color:=ColorTealLight;M.Canvas.Pen.Color:=ColorTealLight;
      M.Canvas.Ellipse(D,D,S-D,S-D);
    end;
    B.Bitmap:=M;
  finally M.Free; end;
end;

// The same overlapping A / ع glyph as the language button in SonarFix.
procedure PaintLanguageButton(B: TBitmapButton);
var M: TBitmap; S: Integer;
begin
  M:=NewSurface(B);
  try
    S:=ScaleY(21);
    M.Canvas.Font.Name:='Segoe UI';M.Canvas.Font.Height:=-ScaleY(13);M.Canvas.Font.Color:=ColorText;
    M.Canvas.Pen.Color:=ColorTealLight;M.Canvas.Brush.Color:=ColorTealDark;
    M.Canvas.RoundRect(0,0,S,S,ScaleX(8),ScaleY(8));
    M.Canvas.Brush.Style:=bsClear;
    M.Canvas.TextOut((S-M.Canvas.TextWidth('A')) div 2,(S-M.Canvas.TextHeight('A')) div 2,'A');
    M.Canvas.Brush.Style:=bsSolid;M.Canvas.Brush.Color:=ColorCard;
    M.Canvas.RoundRect(M.Width-S,M.Height-S,M.Width,M.Height,ScaleX(8),ScaleY(8));
    M.Canvas.Brush.Style:=bsClear;
    M.Canvas.TextOut(M.Width-S+(S-M.Canvas.TextWidth('ع')) div 2,M.Height-S+(S-M.Canvas.TextHeight('ع')) div 2,'ع');
    B.Bitmap:=M;
  finally M.Free; end;
  B.Caption:=Tr('switch_language');B.Hint:=B.Caption;B.ShowHint:=True;
end;

function CreateLanguageButton(Parent: TWinControl; ClientWidth: Integer; Click: TNotifyEvent): TBitmapButton;
begin
  Result:=TBitmapButton.Create(Parent);Result.Parent:=Parent;
  Result.SetBounds(0,ScaleY(10),ScaleX(34)+4,ScaleY(28)+4);
  Result.Cursor:=crHand;Result.OnClick:=Click;Result.TabStop:=True;
end;

procedure PlaceLanguageButton(B: TBitmapButton; ClientWidth: Integer);
begin
  B.Left:=MX(ClientWidth-ScaleX(14)-B.Width,B.Width,ClientWidth);
  PaintLanguageButton(B);B.BringToFront;
end;

function LoadLogo(Parent: TWinControl; Size: Integer): TBitmapImage;
var Sizes: array[0..8] of Integer; I, Pick: Integer; FileName: String;
begin
  Sizes[0]:=58;Sizes[1]:=71;Sizes[2]:=77;Sizes[3]:=85;Sizes[4]:=97;Sizes[5]:=112;Sizes[6]:=124;Sizes[7]:=147;Sizes[8]:=159;
  Pick:=8;
  for I:=8 downto 0 do if Sizes[I]>=Size then Pick:=I;
  FileName:='wizard-small-'+IntToStr(Sizes[Pick])+'.png';
  ExtractTemporaryFile(FileName);
  Result:=TBitmapImage.Create(Parent);Result.Parent:=Parent;
  Result.BackColor:=ColorBack;Result.Stretch:=Sizes[Pick]<>Size;
  Result.PngImage.LoadFromFile(ExpandConstant('{tmp}\')+FileName);
  Result.SetBounds(0,0,Size,Size);
end;

// ---------------------------------------------------------------------------
// Text helpers.

procedure Muted(L: TNewStaticText);
begin
  // Excluding seFont lets the script color win over the dark style.
  L.StyleElements:=[seClient, seBorder];L.Font.Color:=ColorMuted;
end;

function NewText(Parent: TWinControl; Bold: Boolean; Click: TNotifyEvent): TNewStaticText;
begin
  Result:=TNewStaticText.Create(WizardForm);Result.Parent:=Parent;
  Result.AutoSize:=False;Result.WordWrap:=True;Result.ShowAccelChar:=False;
  if Bold then Result.Font.Style:=[fsBold];
  if Click<>nil then begin Result.OnClick:=Click;Result.Cursor:=crHand;end;
end;

procedure PlaceText(L: TNewStaticText; Left, Top, Width: Integer);
begin
  L.SetBounds(Left,Top,Width,ScaleY(20));L.AdjustHeight;
  if IsArabic then L.Alignment:=taRightJustify else L.Alignment:=taLeftJustify;
end;

procedure PlaceCentered(L: TNewStaticText; Top, Width: Integer);
begin
  L.SetBounds(ScaleX(28),Top,Width-ScaleX(56),ScaleY(20));L.AdjustHeight;L.Alignment:=taCenter;
end;

function StateText(On: Boolean): String;
begin if On then Result:=Tr('state_on') else Result:=Tr('state_off'); end;

// ---------------------------------------------------------------------------
// Layout.

procedure LayoutWelcome;
var W, H, Y, Total: Integer;
begin
  W:=WizardForm.WelcomePage.Width;H:=WizardForm.WelcomePage.Height;
  WizardForm.WizardBitmapImage.Visible:=False;
  PlaceCentered(WizardForm.WelcomeLabel1,0,W);PlaceCentered(WizardForm.WelcomeLabel2,0,W);PlaceCentered(ExistingLabel,0,W);
  Total:=WelcomeLogo.Height+ScaleY(18)+WizardForm.WelcomeLabel1.Height+ScaleY(8)+WizardForm.WelcomeLabel2.Height;
  if ExistingLabel.Visible then Total:=Total+ScaleY(20)+ExistingLabel.Height;
  Y:=(H-Total) div 2;
  WelcomeLogo.Left:=(W-WelcomeLogo.Width) div 2;WelcomeLogo.Top:=Y;Y:=Y+WelcomeLogo.Height+ScaleY(18);
  WizardForm.WelcomeLabel1.Top:=Y;Y:=Y+WizardForm.WelcomeLabel1.Height+ScaleY(8);
  WizardForm.WelcomeLabel2.Top:=Y;Y:=Y+WizardForm.WelcomeLabel2.Height+ScaleY(20);
  ExistingLabel.Top:=Y;
end;

procedure LayoutFinished;
var W, H, Y, Total, Box, Gap, TextW: Integer;
begin
  W:=WizardForm.FinishedPage.Width;H:=WizardForm.FinishedPage.Height;
  WizardForm.WizardBitmapImage2.Visible:=False;WizardForm.RunList.Visible:=False;
  PlaceCentered(WizardForm.FinishedHeadingLabel,0,W);PlaceCentered(WizardForm.FinishedLabel,0,W);
  Box:=OptButton[OptOpen].Height;Gap:=ScaleX(8);
  OptTitle[OptOpen].AutoSize:=True;OptTitle[OptOpen].WordWrap:=False;TextW:=OptTitle[OptOpen].Width;
  Total:=FinishedLogo.Height+ScaleY(18)+WizardForm.FinishedHeadingLabel.Height+ScaleY(8)+WizardForm.FinishedLabel.Height+ScaleY(24)+Box;
  Y:=(H-Total) div 2;
  FinishedLogo.Left:=(W-FinishedLogo.Width) div 2;FinishedLogo.Top:=Y;Y:=Y+FinishedLogo.Height+ScaleY(18);
  WizardForm.FinishedHeadingLabel.Top:=Y;Y:=Y+WizardForm.FinishedHeadingLabel.Height+ScaleY(8);
  WizardForm.FinishedLabel.Top:=Y;Y:=Y+WizardForm.FinishedLabel.Height+ScaleY(24);
  OptButton[OptOpen].Top:=Y;
  OptButton[OptOpen].Left:=MX((W-Box-Gap-TextW) div 2,Box,W);
  OptTitle[OptOpen].Left:=MX((W-Box-Gap-TextW) div 2+Box+Gap,TextW,W);
  OptTitle[OptOpen].Top:=Y+(Box-OptTitle[OptOpen].Height) div 2;
end;

procedure LayoutMaintenance;
var W, Y, I, Box, Gap, TextW: Integer;
begin
  W:=MaintenancePage.SurfaceWidth;Box:=ModeButton[0].Height;Gap:=ScaleX(12);TextW:=W-Box-Gap;Y:=ScaleY(4);
  for I:=0 to 2 do begin
    ModeButton[I].Left:=MX(0,Box,W);ModeButton[I].Top:=Y;
    PlaceText(ModeTitle[I],MX(Box+Gap,TextW,W),Y+(Box-ScaleY(20)) div 2,TextW);
    PlaceText(ModeDesc[I],MX(Box+Gap,TextW,W),ModeTitle[I].Top+ModeTitle[I].Height+ScaleY(2),TextW);
    Y:=ModeDesc[I].Top+ModeDesc[I].Height+ScaleY(18);
  end;
end;

procedure PlaceCheckRow(I, Y, W: Integer);
var Box, Gap: Integer;
begin
  Box:=OptButton[I].Height;Gap:=ScaleX(10);
  OptButton[I].Left:=MX(0,Box,W);OptButton[I].Top:=Y;
  PlaceText(OptTitle[I],MX(Box+Gap,W-Box-Gap,W),Y,W-Box-Gap);
  OptTitle[I].Top:=Y+(Box-OptTitle[I].Height) div 2;
end;

procedure LayoutConfig;
var W, Y, I, SwW, SwH, Gap, TextW, BW, FieldH: Integer; ShowOutput: Boolean;
begin
  W:=ConfigPage.SurfaceWidth;Gap:=ScaleX(14);
  SwW:=OptButton[OptEcho].Width;SwH:=OptButton[OptEcho].Height;TextW:=W-SwW-Gap;
  FieldH:=PathEdit.Height;BW:=ScaleX(86)+4;
  PlaceText(LocationLabel,MX(0,W,W),0,W);Y:=LocationLabel.Height+ScaleY(5);
  if BrowseButton.Visible then begin
    PathEdit.SetBounds(MX(0,W-BW-ScaleX(6),W),Y,W-BW-ScaleX(6),FieldH);
    BrowseButton.SetBounds(MX(W-BW,BW,W),Y-2,BW,FieldH+4);
  end else PathEdit.SetBounds(0,Y,W,FieldH);
  Y:=Y+FieldH+ScaleY(18);
  for I:=OptEcho to OptDisable do begin
    OptButton[I].Left:=MX(W-SwW,SwW,W);OptButton[I].Top:=Y;
    PlaceText(OptTitle[I],MX(0,TextW,W),Y,TextW);OptTitle[I].Top:=Y+(SwH-OptTitle[I].Height) div 2;
    Y:=Y+IMax(SwH,OptTitle[I].Height)+ScaleY(1);
    PlaceText(OptDesc[I],MX(0,TextW,W),Y,TextW);Y:=Y+OptDesc[I].Height;
    if I=OptDisable then begin
      ShowOutput:=OptOn[OptDisable];
      OutputLabel.Visible:=ShowOutput;OutputCombo.Visible:=ShowOutput;
      OptButton[OptRedirect].Visible:=ShowOutput;OptTitle[OptRedirect].Visible:=ShowOutput;
      if ShowOutput then begin
        Y:=Y+ScaleY(10);
        PlaceText(OutputLabel,MX(0,TextW,W),Y,TextW);Y:=Y+OutputLabel.Height+ScaleY(4);
        OutputCombo.SetBounds(0,Y,W,FieldH);Y:=Y+OutputCombo.Height+ScaleY(8);
        PlaceCheckRow(OptRedirect,Y,W);Y:=Y+OptButton[OptRedirect].Height;
      end;
    end;
    Y:=Y+ScaleY(16);
  end;
  PlaceCheckRow(OptStartup,Y,W);
  ConfigBottom:=Y+OptButton[OptStartup].Height;
end;

procedure SetOption(I: Integer; Value: Boolean);
begin
  OptOn[I]:=Value;
  if I<=OptDisable then PaintSwitch(OptButton[I],Value) else PaintCheck(OptButton[I],Value);
  OptButton[I].Caption:=OptTitle[I].Caption+' : '+StateText(Value);
  if I=OptDisable then LayoutConfig;
end;

procedure OptionClicked(Sender: TObject);
var I: Integer;
begin
  for I:=0 to 4 do if (Sender=OptButton[I]) or (Sender=OptTitle[I]) then begin SetOption(I,not OptOn[I]);Exit;end;
  for I:=0 to 1 do if Sender=OptDesc[I] then begin SetOption(I,not OptOn[I]);Exit;end;
end;

procedure PlaceOverlay(O: TBitmapButton; N: TNewButton; Caption: String; Fill: TColor; Show: Boolean);
begin
  O.Visible:=Show and N.Visible;N.TabStop:=not O.Visible;
  if not O.Visible then Exit;
  O.Caption:=Caption;O.Enabled:=N.Enabled;
  O.SetBounds(N.Left-2,N.Top-2,N.Width+4,N.Height+4);
  if Fill=ColorNeutral then PaintButton(O,Fill,ColorText,False) else PaintButton(O,Fill,clWhite,True);
  O.BringToFront;
end;

procedure UpdateButtons;
var Page: Integer; NextCaption: String; Fill: TColor; Show: Boolean;
begin
  Page:=WizardForm.CurPageID;NextCaption:=Tr('next');Fill:=ColorTeal;
  if Page=ConfigPage.ID then NextCaption:=Tr('install')
  else if Page=MaintenancePage.ID then begin
    if Mode=ModeUpdate then NextCaption:=Tr('update_action')
    else if Mode=ModeUninstall then begin NextCaption:=Tr('uninstall_2');Fill:=ColorDanger;end;
  end else if Page=wpFinished then NextCaption:=Tr('finish');
  WizardForm.NextButton.Caption:=NextCaption;WizardForm.BackButton.Caption:=Tr('back');WizardForm.CancelButton.Caption:=Tr('cancel');
  Show:=(Page=wpWelcome) or (Page=MaintenancePage.ID) or (Page=ConfigPage.ID) or (Page=wpFinished);
  PlaceOverlay(NextOverlay,WizardForm.NextButton,NextCaption,Fill,Show);
  PlaceOverlay(BackOverlay,WizardForm.BackButton,Tr('back'),ColorNeutral,Show);
  PlaceOverlay(CancelOverlay,WizardForm.CancelButton,Tr('cancel'),ColorNeutral,Show);
end;

procedure SetMode(M: Integer);
var I: Integer;
begin
  Mode:=M;
  for I:=0 to 2 do begin PaintRadio(ModeButton[I],I=M);ModeButton[I].Caption:=ModeTitle[I].Caption+' : '+StateText(I=M);end;
  UpdateButtons;
end;

procedure ModeClicked(Sender: TObject);
var I: Integer;
begin
  for I:=0 to 2 do if (Sender=ModeButton[I]) or (Sender=ModeTitle[I]) or (Sender=ModeDesc[I]) then SetMode(I);
end;

procedure ApplyUninstallLanguage;
begin
  UninstallProgressForm.Caption:=Tr('uninstall_title');
  UninstallProgressForm.PageNameLabel.Caption:=Tr('uninstall_title');
  UninstallProgressForm.PageDescriptionLabel.Caption:=Tr('please_wait');
  UninstallProgressForm.CancelButton.Caption:=Tr('cancel');
  UninstallProgressForm.StatusLabel.Visible:=False;
  if IsArabic then begin
    UninstallProgressForm.PageNameLabel.Alignment:=taRightJustify;UninstallProgressForm.PageDescriptionLabel.Alignment:=taRightJustify;
  end else begin
    UninstallProgressForm.PageNameLabel.Alignment:=taLeftJustify;UninstallProgressForm.PageDescriptionLabel.Alignment:=taLeftJustify;
  end;
  PlaceLanguageButton(LanguageButton,UninstallProgressForm.ClientWidth);
end;

// All editable pages are owned by this script. Language changes only captions
// and layout in this process; selections and audio configuration are untouched.
procedure ApplyLanguage;
var I: Integer; Version: String;
begin
  Version:=Isolated('{#AppVersion}');
  WizardForm.Caption:=Tr('set_up_sonarfix');
  WizardForm.WelcomeLabel1.Caption:='SonarFix';
  WizardForm.WelcomeLabel2.Caption:=Tr('welcome_description');
  if PrevVersion<>'' then ExistingLabel.Caption:=FmtMessage(Tr('installed_found'),[Isolated(PrevVersion)])
  else ExistingLabel.Caption:=Tr('maintenance_title');
  MaintenancePage.Caption:=Tr('maintenance_title');MaintenancePage.Description:=Tr('maintenance_description');
  ModeTitle[ModeUpdate].Caption:=Tr('option_update');ModeDesc[ModeUpdate].Caption:=FmtMessage(Tr('option_update_description'),[Version]);
  ModeTitle[ModeModify].Caption:=Tr('option_modify');ModeDesc[ModeModify].Caption:=FmtMessage(Tr('option_modify_description'),[Version]);
  ModeTitle[ModeUninstall].Caption:=Tr('option_uninstall');ModeDesc[ModeUninstall].Caption:=Tr('option_uninstall_description');
  ConfigPage.Caption:=Tr('set_up_sonarfix');
  ConfigPage.Description:=Tr('config_description');
  if WizardForm.CurPageID=MaintenancePage.ID then begin
    WizardForm.PageNameLabel.Caption:=MaintenancePage.Caption;WizardForm.PageDescriptionLabel.Caption:=MaintenancePage.Description;
  end else if WizardForm.CurPageID=wpInstalling then begin
    WizardForm.PageNameLabel.Caption:=Tr('installing_sonarfix');WizardForm.PageDescriptionLabel.Caption:=Tr('please_wait');
    WizardForm.StatusLabel.Visible:=False;WizardForm.FilenameLabel.Visible:=False;
  end else begin
    WizardForm.PageNameLabel.Caption:=ConfigPage.Caption;WizardForm.PageDescriptionLabel.Caption:=ConfigPage.Description;
  end;
  LocationLabel.Caption:=Tr('install_location_2');BrowseButton.Caption:=Tr('browse_2');
  OptTitle[OptEcho].Caption:=Tr('prevent_duplicate_mic_audio');OptDesc[OptEcho].Caption:=Tr('prevents_duplicate_mic_audio_during_discord_screen_s');
  OptTitle[OptDisable].Caption:=Tr('disable_game_chat');OptDesc[OptDisable].Caption:=Tr('disables_sonar_game_and_chat_outputs_while_keeping_y');
  OutputLabel.Caption:=Tr('audio_output_device');
  OptTitle[OptRedirect].Caption:=Tr('set_as_windows_default_output');
  OptTitle[OptStartup].Caption:=Tr('start_with_windows');
  OptTitle[OptOpen].Caption:=Tr('open_sonarfix');
  WizardForm.FinishedHeadingLabel.Caption:=Tr('sonarfix_installed_successfully');
  WizardForm.FinishedLabel.Caption:=Tr('you_can_now_open_sonarfix_and_manage_your_selected_f');
  if WizardForm.RunList.Items.Count>0 then WizardForm.RunList.ItemCaption[0]:=Tr('open_sonarfix');
  if IsArabic then begin
    WizardForm.PageNameLabel.Alignment:=taRightJustify;WizardForm.PageDescriptionLabel.Alignment:=taRightJustify;
    WizardForm.PageNameLabel.Left:=WizardForm.ClientWidth-BasePageNameLeft-WizardForm.PageNameLabel.Width;
    WizardForm.BackButton.Left:=WizardForm.ClientWidth-BaseBack-WizardForm.BackButton.Width;
    WizardForm.NextButton.Left:=WizardForm.ClientWidth-BaseNext-WizardForm.NextButton.Width;
    WizardForm.CancelButton.Left:=WizardForm.ClientWidth-BaseCancel-WizardForm.CancelButton.Width;
  end else begin
    WizardForm.PageNameLabel.Alignment:=taLeftJustify;WizardForm.PageDescriptionLabel.Alignment:=taLeftJustify;
    WizardForm.PageNameLabel.Left:=BasePageNameLeft;
    WizardForm.BackButton.Left:=BaseBack;WizardForm.NextButton.Left:=BaseNext;WizardForm.CancelButton.Left:=BaseCancel;
  end;
  WizardForm.PageDescriptionLabel.Left:=WizardForm.PageNameLabel.Left;
  // Finish is the only button on the last page; keep it at the outer edge.
  if WizardForm.CurPageID=wpFinished then WizardForm.NextButton.Left:=WizardForm.CancelButton.Left;
  PaintButton(BrowseButton,ColorNeutral,ColorText,False);
  for I:=0 to 4 do SetOption(I,OptOn[I]);
  SetMode(Mode);
  LayoutWelcome;LayoutMaintenance;LayoutConfig;LayoutFinished;
  PlaceLanguageButton(LanguageButton,WizardForm.ClientWidth);
  UpdateButtons;
end;

function InitializeSetup: Boolean;
var SavedLanguage: String;
begin
  Result := True; UILanguage := ActiveLanguage;
  if (Pos('/LANG=',Uppercase(GetCmdTail))=0) and
    RegQueryStringValue(HKCU,'Software\SonarFix','UILanguage',SavedLanguage) then
    if (SavedLanguage='en') or (SavedLanguage='ar') then UILanguage:=SavedLanguage;
end;

procedure LanguageChanged(Sender: TObject);
begin
  if IsArabic then UILanguage:='en' else UILanguage:='ar';
  if not Validating then RegWriteStringValue(HKCU,'Software\SonarFix','UILanguage',UILanguage);
  if UninstallingUI then ApplyUninstallLanguage else ApplyLanguage;
end;

procedure CancelButtonClick(CurPageID: Integer; var Cancel, Confirm: Boolean);
begin
  Confirm := False;
  if QuietClose or Validating then Exit;
  Cancel := ShowPrompt(Tr('cancel_sonarfix_setup'),mbConfirmation,MB_YESNO)=IDYES;
end;

procedure BrowseClicked(Sender: TObject);
var Dir: String;
begin
  Dir := PathEdit.Text;
  if BrowseForFolder(Tr('choose_install_folder'), Dir, True) then PathEdit.Text := Dir;
end;

procedure RemoveExisting;
var Uninstaller: String; Code: Integer;
begin
  Uninstaller := AddBackslash(InstalledPath) + 'unins000.exe';
  if not FileExists(Uninstaller) then begin
    if not FileExists(ExpandConstant('{localappdata}\SonarFix\SonarFix.exe')) then begin
      ShowPrompt(Tr('no_installation_found'), mbInformation, MB_OK); Exit;
    end;
    if not Exec(ExpandConstant('{tmp}\SonarFix.exe'), '--prepare-uninstall', '', SW_HIDE, ewWaitUntilTerminated, Code) then Code := 1;
    if Code <> 0 then begin ShowPrompt(Tr('audio_restoration_failed_the_recovery_settings_have_'),mbError,MB_OK); Exit; end;
    if ShowPrompt(Tr('legacy_sonarfix_is_stopped_and_startup_is_removed_al'),mbConfirmation,MB_YESNO) = IDYES then begin
      DeleteFile(ExpandConstant('{localappdata}\SonarFix\SonarFix.exe'));
      DeleteFile(ExpandConstant('{localappdata}\SonarFix\settings.json'));
      DeleteFile(ExpandConstant('{localappdata}\SonarFix\status.json'));
      DeleteFile(ExpandConstant('{localappdata}\SonarFix\SonarFix.log'));
      DeleteFile(ExpandConstant('{localappdata}\SonarFix\SonarFix.log.1'));
      RemoveDir(ExpandConstant('{localappdata}\SonarFix'));
    end;
    ShowPrompt(Tr('legacy_sonarfix_removed'),mbInformation,MB_OK); QuietClose := True; WizardForm.Close; Exit;
  end;
  // The uninstaller hands over to the SonarFix confirmation window.
  if Exec(Uninstaller, '', '', SW_SHOWNORMAL, ewWaitUntilTerminated, Code) then begin
    if Code = 0 then begin QuietClose := True; WizardForm.Close; end;
  end;
end;

procedure OverlayClicked(Sender: TObject);
begin
  if Sender=BackOverlay then WizardForm.BackButton.OnClick(WizardForm.BackButton)
  else if Sender=CancelOverlay then WizardForm.CancelButton.OnClick(WizardForm.CancelButton)
  else WizardForm.NextButton.OnClick(WizardForm.NextButton);
end;

function CreateOverlay: TBitmapButton;
begin
  Result:=TBitmapButton.Create(WizardForm);Result.Parent:=WizardForm;
  Result.Cursor:=crHand;Result.OnClick:=@OverlayClicked;Result.Visible:=False;Result.TabStop:=True;
end;

function CreateOptionButton(Parent: TWinControl; Width, Height: Integer; Click: TNotifyEvent): TBitmapButton;
begin
  Result:=TBitmapButton.Create(WizardForm);Result.Parent:=Parent;
  Result.SetBounds(0,0,Width+4,Height+4);Result.Cursor:=crHand;Result.OnClick:=Click;Result.TabStop:=True;
end;

// Equal button widths that fit every caption in both languages.
procedure SizeNavigationButtons;
var W, Margin, GapNC, GapBN: Integer; Saved: String;
begin
  Margin:=WizardForm.ClientWidth-WizardForm.CancelButton.Left-WizardForm.CancelButton.Width;
  GapNC:=WizardForm.CancelButton.Left-WizardForm.NextButton.Left-WizardForm.NextButton.Width;
  GapBN:=IMax(ScaleX(8),WizardForm.NextButton.Left-WizardForm.BackButton.Left-WizardForm.BackButton.Width);
  Saved:=UILanguage;
  UILanguage:='en';W:=WizardForm.CalculateButtonWidth([Tr('next'),Tr('install'),Tr('update_action'),Tr('uninstall_2'),Tr('finish'),Tr('back'),Tr('cancel')]);
  UILanguage:='ar';W:=IMax(W,WizardForm.CalculateButtonWidth([Tr('next'),Tr('install'),Tr('update_action'),Tr('uninstall_2'),Tr('finish'),Tr('back'),Tr('cancel')]));
  UILanguage:=Saved;
  W:=IMax(W+ScaleX(12),WizardForm.NextButton.Width);
  WizardForm.CancelButton.Width:=W;WizardForm.NextButton.Width:=W;WizardForm.BackButton.Width:=W;
  WizardForm.CancelButton.Left:=WizardForm.ClientWidth-Margin-W;
  WizardForm.NextButton.Left:=WizardForm.CancelButton.Left-GapNC-W;
  WizardForm.BackButton.Left:=WizardForm.NextButton.Left-GapBN-W;
  BaseBack:=WizardForm.BackButton.Left;BaseNext:=WizardForm.NextButton.Left;BaseCancel:=WizardForm.CancelButton.Left;
end;

procedure InitializeWizard;
var Code, I, Count: Integer; ID, Name, Chosen, OptionValue: String;
begin
  OutputIDs := TStringList.Create;
  ExtractTemporaryFile('SonarFix.exe'); InfoFile := ExpandConstant('{tmp}\sonarfix-info.ini');
  if not Exec(ExpandConstant('{tmp}\SonarFix.exe'), '--setup-info "'+InfoFile+'"', '', SW_HIDE, ewWaitUntilTerminated, Code) then
    RaiseException(Txt('Unable to inspect audio devices.','تعذر فحص أجهزة الصوت.'));
  if Code <> 0 then RaiseException(Txt('Unable to read SonarFix settings. Open logs.','تعذر قراءة إعدادات SonarFix. افتح السجلات.'));

  PrevInstalled:=FileExists(AddBackslash(InstalledPath)+'unins000.exe');
  LegacyInstalled:=(not PrevInstalled) and FileExists(ExpandConstant('{localappdata}\SonarFix\SonarFix.exe'));
  if Validating and (ExpandConstant('{param:UIFRESH|0}')='1') then begin PrevInstalled:=False;LegacyInstalled:=False;end;
  if PrevInstalled then RegQueryStringValue(HKCU,'Software\Microsoft\Windows\CurrentVersion\Uninstall\SonarFix_is1','DisplayVersion',PrevVersion);
  PrevInstalled:=PrevInstalled or LegacyInstalled;
  Mode:=ModeUpdate;

  SizeNavigationButtons;
  BasePageNameLeft:=WizardForm.PageNameLabel.Left;
  WizardForm.WizardSmallBitmapImage.Visible:=False;
  WizardForm.PageNameLabel.Width:=WizardForm.ClientWidth-BasePageNameLeft-ScaleX(70);
  WizardForm.PageDescriptionLabel.Width:=WizardForm.PageNameLabel.Width;
  Muted(WizardForm.PageDescriptionLabel);

  // Welcome: centered logo, name, one-line description and installation state.
  WelcomeLogo:=LoadLogo(WizardForm.WelcomePage,ScaleX(64));
  WizardForm.WelcomeLabel1.Font.Size:=20;WizardForm.WelcomeLabel1.Font.Style:=[fsBold];
  WizardForm.WelcomeLabel1.AutoSize:=False;WizardForm.WelcomeLabel1.WordWrap:=True;
  WizardForm.WelcomeLabel2.AutoSize:=False;WizardForm.WelcomeLabel2.WordWrap:=True;Muted(WizardForm.WelcomeLabel2);
  ExistingLabel:=NewText(WizardForm.WelcomePage,False,nil);ExistingLabel.Visible:=PrevInstalled;
  ExistingLabel.StyleElements:=[seClient, seBorder];ExistingLabel.Font.Color:=ColorTealLight;

  // Previous installation: update, modify or uninstall.
  MaintenancePage:=CreateCustomPage(wpWelcome,'','');
  for I:=0 to 2 do begin
    ModeButton[I]:=CreateOptionButton(MaintenancePage.Surface,ScaleX(18),ScaleY(18),@ModeClicked);
    ModeTitle[I]:=NewText(MaintenancePage.Surface,True,@ModeClicked);
    ModeDesc[I]:=NewText(MaintenancePage.Surface,False,@ModeClicked);Muted(ModeDesc[I]);
  end;

  ConfigPage := CreateCustomPage(MaintenancePage.ID,'','');
  LocationLabel:=NewText(ConfigPage.Surface,False,nil);Muted(LocationLabel);
  PathEdit := TNewEdit.Create(WizardForm); PathEdit.Parent := ConfigPage.Surface;
  PathEdit.AutoSelect:=False;PathEdit.Text := WizardDirValue;
  // Same field look as the read-only installation folder in SonarFix.
  PathEdit.StyleElements:=[seBorder];PathEdit.Color:=ColorField;PathEdit.Font.Color:=ColorText;
  BrowseButton:=CreateOptionButton(ConfigPage.Surface,ScaleX(86),ScaleY(26),@BrowseClicked);
  // An existing installation keeps its folder; changing it would leave the old files behind.
  if PrevInstalled and not LegacyInstalled then begin PathEdit.ReadOnly:=True;BrowseButton.Visible:=False;end;

  for I:=OptEcho to OptDisable do begin
    OptButton[I]:=CreateOptionButton(ConfigPage.Surface,ScaleX(38),ScaleY(22),@OptionClicked);
    OptTitle[I]:=NewText(ConfigPage.Surface,True,@OptionClicked);
    OptDesc[I]:=NewText(ConfigPage.Surface,False,@OptionClicked);Muted(OptDesc[I]);
  end;
  OutputLabel:=NewText(ConfigPage.Surface,False,nil);Muted(OutputLabel);
  OutputCombo := TNewComboBox.Create(WizardForm); OutputCombo.Parent := ConfigPage.Surface; OutputCombo.Style := csDropDownList;
  for I:=OptRedirect to OptStartup do begin
    OptButton[I]:=CreateOptionButton(ConfigPage.Surface,ScaleX(18),ScaleY(18),@OptionClicked);
    OptTitle[I]:=NewText(ConfigPage.Surface,False,@OptionClicked);
  end;
  OptButton[OptOpen]:=CreateOptionButton(WizardForm.FinishedPage,ScaleX(18),ScaleY(18),@OptionClicked);
  OptTitle[OptOpen]:=NewText(WizardForm.FinishedPage,False,@OptionClicked);
  FinishedLogo:=LoadLogo(WizardForm.FinishedPage,ScaleX(64));
  WizardForm.FinishedHeadingLabel.AutoSize:=False;WizardForm.FinishedHeadingLabel.WordWrap:=True;
  WizardForm.FinishedLabel.AutoSize:=False;WizardForm.FinishedLabel.WordWrap:=True;Muted(WizardForm.FinishedLabel);

  // Current saved options; command-line values override them.
  OptOn[OptEcho] := GetIniString('Settings','Echo','false',InfoFile) = 'true';
  OptionValue := ExpandConstant('{param:ECHO|}'); if OptionValue <> '' then OptOn[OptEcho] := OptionValue = '1';
  OptOn[OptDisable] := GetIniString('Settings','Disable','false',InfoFile) = 'true';
  OptionValue := ExpandConstant('{param:DISABLE|}'); if OptionValue <> '' then OptOn[OptDisable] := OptionValue = '1';
  OptOn[OptRedirect] := GetIniString('Settings','Redirect','false',InfoFile) = 'true';
  OptionValue := ExpandConstant('{param:REDIRECT|}'); if OptionValue <> '' then OptOn[OptRedirect] := OptionValue = '1';
  OptOn[OptStartup] := GetIniString('Settings','Startup','false',InfoFile) = 'true';
  OptionValue := ExpandConstant('{param:STARTUP|}'); if OptionValue <> '' then OptOn[OptStartup] := OptionValue = '1';
  OptOn[OptOpen] := True;

  Count := StrToIntDef(GetIniString('Outputs','Count','0',InfoFile),0);
  Chosen := GetIniString('Settings','Output','',InfoFile); if Chosen = '' then Chosen := GetIniString('Outputs','Default','',InfoFile);
  OptionValue := ExpandConstant('{param:OUTPUT|}'); if OptionValue <> '' then Chosen := OptionValue;
  for I := 0 to Count-1 do begin
    ID := GetIniString('Output'+IntToStr(I),'ID','',InfoFile); Name := GetIniString('Output'+IntToStr(I),'Name','',InfoFile);
    OutputCombo.Items.Add(Name); OutputIDs.Add(ID); if ID = Chosen then OutputCombo.ItemIndex := I;
  end;
  if (Chosen<>'') and (OutputCombo.ItemIndex<0) then begin
    OutputCombo.Items.Add(Tr('unavailable_saved_output')+' — '+Chosen); OutputIDs.Add(Chosen); OutputCombo.ItemIndex:=OutputIDs.Count-1;
  end;

  BackOverlay:=CreateOverlay;NextOverlay:=CreateOverlay;CancelOverlay:=CreateOverlay;
  LanguageButton:=CreateLanguageButton(WizardForm,WizardForm.ClientWidth,@LanguageChanged);
  if not Validating then RegWriteStringValue(HKCU, 'Software\SonarFix', 'UILanguage', UILanguage);
  ApplyLanguage;

  // Worst case for fitting: the output section is open.
  OptOn[OptDisable]:=True;LayoutConfig;
  Log(Format('SonarFix config layout: surface height=%d; content bottom=%d', [ConfigPage.SurfaceHeight, ConfigBottom]));
  if Validating and (ConfigBottom > ConfigPage.SurfaceHeight) then RaiseException('SonarFix layout does not fit this display scale.');
  OptOn[OptDisable] := GetIniString('Settings','Disable','false',InfoFile) = 'true';
  OptionValue := ExpandConstant('{param:DISABLE|}'); if OptionValue <> '' then OptOn[OptDisable] := OptionValue = '1';
  SetOption(OptDisable,OptOn[OptDisable]);
end;

function ShouldSkipPage(PageID: Integer): Boolean;
begin
  Result := False;
  if PageID = MaintenancePage.ID then Result := not PrevInstalled
  else if PageID = ConfigPage.ID then Result := PrevInstalled and (Mode = ModeUpdate);
end;

procedure TestLanguageSwitches;
var Path, Device, ReportFile: String; Saved: array[0..4] of Boolean; I, J, Sw: Integer;
begin
  ReportFile:=ExpandConstant('{param:UITESTFILE|}');
  if (ReportFile='') or UITestDone or not Validating then Exit;
  UITestDone:=True;
  SaveStringToFile(ReportFile,'FAIL: test did not finish',False);
  PathEdit.Text:='C:\SonarFix UI test\العربية';Path:=PathEdit.Text;
  SetOption(OptEcho,False);SetOption(OptDisable,True);SetOption(OptRedirect,True);SetOption(OptStartup,False);
  if OutputCombo.Items.Count=0 then begin OutputCombo.Items.Add('Headphones (Arctis Nova Pro Wireless)');OutputIDs.Add('test-only-output');end;
  OutputCombo.ItemIndex:=OutputCombo.Items.Count-1;Device:=OutputIDs[OutputCombo.ItemIndex];
  for J:=0 to 4 do Saved[J]:=OptOn[J];
  UILanguage:='ar';
  for I:=0 to 5 do begin
    LanguageChanged(LanguageButton);
    if PathEdit.Text<>Path then RaiseException('Path changed');
    for J:=0 to 4 do if OptOn[J]<>Saved[J] then RaiseException('Feature choices changed');
    if OutputIDs[OutputCombo.ItemIndex]<>Device then RaiseException('Output changed');
    if (PathEdit.Left<0) or (PathEdit.Left+PathEdit.Width>ConfigPage.SurfaceWidth) then RaiseException('Path layout');
    if BrowseButton.Visible and (BrowseButton.Left<PathEdit.Left+PathEdit.Width) and (BrowseButton.Left+BrowseButton.Width>PathEdit.Left) then RaiseException('Browse/path overlap');
    if ConfigBottom>ConfigPage.SurfaceHeight then RaiseException('Content clipping');
    if not OutputCombo.Visible or not OptButton[OptRedirect].Visible then RaiseException('Output section hidden while Game/Chat is selected');
    Sw:=OptButton[OptEcho].Left;
    if (OptTitle[OptEcho].Left<0) or (OptTitle[OptEcho].Left+OptTitle[OptEcho].Width>ConfigPage.SurfaceWidth) then RaiseException('Feature title bounds');
    if (LanguageButton.Left<0) or (LanguageButton.Left+LanguageButton.Width>WizardForm.ClientWidth) then RaiseException('Language button bounds');
    if UILanguage='ar' then begin
      if OptTitle[OptEcho].Alignment<>taRightJustify then RaiseException('Arabic layout');
      if Copy(OptDesc[OptEcho].Caption,1,1)<>#$202B then RaiseException('Arabic paragraph direction');
      if Sw>OptTitle[OptEcho].Left then RaiseException('Arabic switch side');
      if LanguageButton.Left>WizardForm.ClientWidth div 2 then RaiseException('Arabic language button side');
    end else begin
      if OptTitle[OptEcho].Alignment<>taLeftJustify then RaiseException('English layout');
      if Sw<OptTitle[OptEcho].Left then RaiseException('English switch side');
      if LanguageButton.Left<WizardForm.ClientWidth div 2 then RaiseException('English language button side');
    end;
    if (WizardForm.CurPageID=wpWelcome) and (NextOverlay.Caption<>Tr('next')) then RaiseException('Welcome must navigate before installation');
    if (WizardForm.CurPageID=wpWelcome) and NextOverlay.Visible and (NextOverlay.Left<>WizardForm.NextButton.Left-2) then RaiseException('Next button overlay position');
  end;
  SetOption(OptDisable,False);
  if OutputCombo.Visible or OptButton[OptRedirect].Visible then RaiseException('Output section shown while Game/Chat is off');
  SetOption(OptDisable,True);
  if not OptOn[OptRedirect] then RaiseException('Redirect choice lost when hiding the output section');
  SaveStringToFile(ReportFile,'PASS: six live language callbacks; same wizard; path, saved device, feature choices, redirect and startup preserved; output section follows Game/Chat; Arabic/English mirroring, alignment and bounds valid; no configure/stop/start invoked.',False);
  QuietClose:=True;PostWizardMessage(WizardForm.Handle,$0010,0,0);
end;

// Validation-only navigation for layout checks: /UIPAGE=maintenance|config.
procedure NavigateForValidation(CurPageID: Integer);
var Target: String;
begin
  Target:=ExpandConstant('{param:UIPAGE|}');
  if (Target='') or UINavigated or not Validating then Exit;
  if CurPageID=wpWelcome then begin
    if (Target='maintenance') and not PrevInstalled then begin UINavigated:=True;Exit;end;
    PostWizardMessage(WizardForm.NextButton.Handle,$00F5,0,0);
  end else if CurPageID=MaintenancePage.ID then begin
    if Target='config' then begin SetMode(ModeModify);PostWizardMessage(WizardForm.NextButton.Handle,$00F5,0,0);end else UINavigated:=True;
  end else UINavigated:=True;
end;

procedure CurPageChanged(CurPageID: Integer);
begin
  LanguageButton.Visible:=True;
  ApplyLanguage;
  if CurPageID=ConfigPage.ID then begin PathEdit.SelLength:=0;WizardForm.ActiveControl:=OptButton[OptEcho];end
  else if CurPageID=MaintenancePage.ID then WizardForm.ActiveControl:=ModeButton[Mode]
  else if NextOverlay.Visible then WizardForm.ActiveControl:=NextOverlay;
  TestLanguageSwitches;
  NavigateForValidation(CurPageID);
end;

function OptionsValid: Boolean;
begin
  Result:=not (OptOn[OptDisable] and ((OutputCombo.ItemIndex<0) or (Pos('{',OutputCombo.Text)=1) or (Pos(' — {',OutputCombo.Text)>0)));
end;

function NextButtonClick(CurPageID: Integer): Boolean;
begin
  Result := True;
  // Explicit read-only validation mode used for locale/layout checks. It exits
  // before PrepareToInstall, file installation, configuration, or startup work.
  if Validating then begin
    Result := (CurPageID=wpWelcome) or ((CurPageID=MaintenancePage.ID) and (Mode=ModeModify));
    Exit;
  end;
  if CurPageID = MaintenancePage.ID then begin
    if Mode = ModeUninstall then begin RemoveExisting; Result := False; Exit; end;
    if (Mode = ModeUpdate) and not OptionsValid then begin
      // The saved output is gone; review the features instead of failing after copying files.
      ShowPrompt(Tr('select_an_active_physical_output_before_disabling_ga'),mbError,MB_OK);
      SetMode(ModeModify);
    end;
  end;
  if CurPageID = ConfigPage.ID then begin
    if (Pos(':\',PathEdit.Text) <> 2) or (Length(PathEdit.Text)<4) then begin
      ShowPrompt(Tr('choose_a_local_install_directory'),mbError,MB_OK); Result := False; Exit;
    end;
    if not OptionsValid then begin
      ShowPrompt(Tr('select_an_active_physical_output_before_disabling_ga'),mbError,MB_OK); Result := False; Exit;
    end;
    WizardForm.DirEdit.Text := PathEdit.Text;
  end;
  if (CurPageID = wpFinished) and (WizardForm.RunList.Items.Count>0) then WizardForm.RunList.Checked[0]:=OptOn[OptOpen];
end;

function PrepareToInstall(var NeedsRestart: Boolean): String;
var Code: Integer;
begin
  Result := '';
  if not Exec(ExpandConstant('{tmp}\SonarFix.exe'), '--stop', '', SW_HIDE, ewWaitUntilTerminated, Code) then Code := 1;
  if Code <> 0 then Result := Tr('sonarfix_could_not_stop_close_it_before_installing');
end;

function BoolJSON(B: Boolean): String;
begin if B then Result := 'true' else Result := 'false'; end;

procedure CurStepChanged(CurStep: TSetupStep);
var Request, Output, Json: String; Code: Integer;
begin
  if CurStep <> ssPostInstall then Exit;
  Output := ''; if OutputCombo.ItemIndex >= 0 then Output := OutputIDs[OutputCombo.ItemIndex];
  Request := ExpandConstant('{tmp}\sonarfix-config.json');
  Json := '{"fix_discord_echo":'+BoolJSON(OptOn[OptEcho])+',"disable_game_and_chat":'+BoolJSON(OptOn[OptDisable])+',"physical_output_id":"'+Output+'","allow_default_redirect":'+BoolJSON(OptOn[OptRedirect])+',"startup":'+BoolJSON(OptOn[OptStartup])+',"language":"'+UILanguage+'"}';
  if not SaveStringToFile(Request,Json,False) then RaiseException(Txt('Unable to save settings.','تعذر حفظ الإعدادات.'));
  if not Exec(ExpandConstant('{app}\SonarFix.exe'), '--configure "'+Request+'"', '', SW_HIDE, ewWaitUntilTerminated, Code) then Code := 1;
  if Code <> 0 then RaiseException(Tr('audio_options_were_not_applied_open_sonarfix_setting'));
  if OptOn[OptEcho] or OptOn[OptDisable] then
    if not Exec(ExpandConstant('{app}\SonarFix.exe'), '--background', '', SW_HIDE, ewNoWait, Code) then RaiseException(Txt('Unable to start SonarFix.','تعذر تشغيل SonarFix.'));
  ApplyLanguage;
end;

// All interactive entry points use the same modal in Settings.exe. Returning
// False here avoids Inno's native Yes/No dialog; confirmed removal uses /SILENT.
function InitializeUninstall: Boolean;
var Code: Integer; Manager: String;
begin
  UILanguage:=ActiveLanguage;RegQueryStringValue(HKCU,'Software\SonarFix','UILanguage',UILanguage);
  Result:=True;
  if not UninstallSilent then begin
    Manager:=ExpandConstant('{app}\SonarFix.Settings.exe');
    if FileExists(Manager) then begin
      if not Exec(Manager,'--uninstall','',SW_SHOWNORMAL,ewNoWait,Code) then RaiseException(Txt('Unable to open SonarFix.','تعذر فتح SonarFix.'));
      Result:=False;
    end;
  end;
end;

procedure InitializeUninstallProgressForm;
begin
  UninstallingUI:=True;
  LanguageButton:=CreateLanguageButton(UninstallProgressForm,UninstallProgressForm.ClientWidth,@LanguageChanged);
  UninstallProgressForm.WizardSmallBitmapImage.Visible:=False;
  UninstallProgressForm.PageNameLabel.Width:=UninstallProgressForm.ClientWidth-ScaleX(90);
  UninstallProgressForm.PageDescriptionLabel.Width:=UninstallProgressForm.PageNameLabel.Width;
  ApplyUninstallLanguage;
end;

procedure CurUninstallStepChanged(CurUninstallStep: TUninstallStep);
var Code, PID: Integer; H: THandle;
begin
  if CurUninstallStep=usAppMutexCheck then begin
    PID:=StrToIntDef(ExpandConstant('{param:WAITPID|0}'),0);
    if PID>0 then begin
      H:=OpenWaitProcess($00100000,False,PID);
      if H<>0 then try
        if WaitForProcess(H,15000)<>0 then RaiseException(Txt('SonarFix did not close. No files were removed.','لم تُغلق واجهة SonarFix. لم تُحذف أي ملفات.'));
      finally CloseWaitHandle(H);end;
    end;
  end;
  if CurUninstallStep = usUninstall then begin
    // This event runs after Inno's confirmation. Its exceptions are fatal,
    // so no application file is deleted if restoration fails.
    if not FileExists(ExpandConstant('{app}\SonarFix.exe')) then
      RaiseException(Tr('the_audio_worker_is_missing_repair_the_installation_'));
    if not Exec(ExpandConstant('{app}\SonarFix.exe'), '--prepare-uninstall', '', SW_HIDE, ewWaitUntilTerminated, Code) then Code := 1;
    if Code <> 0 then RaiseException(Tr('restoration_is_incomplete_uninstall_has_stopped_so_t'));

    // Keep preferences and recovery records by default; no second confirmation.
  end;
end;
