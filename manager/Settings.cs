using System;
using System.Collections.Generic;
using System.Diagnostics;
using System.IO;
using System.Reflection;
using System.Runtime.InteropServices;
using System.Text;
using System.Threading;
using System.Threading.Tasks;
using System.Web.Script.Serialization;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Controls.Primitives;
using System.Windows.Interop;
using System.Windows.Markup;
using System.Windows.Media;
using System.Windows.Threading;
using Microsoft.Win32;

[assembly: AssemblyTitle("SonarFix Settings")]
[assembly: AssemblyDescription("Manage independent SonarFix audio options")]
[assembly: AssemblyCompany("SonarFix Project")]
[assembly: AssemblyProduct("SonarFix")]

class OutputDevice { public string ID {get;set;} public string Name {get;set;} }
partial class SettingsApp {
    static Window window;
    static bool selfTesting, uninstalling;
    static int commandCount;
    static string ProductVersion {get{return ((AssemblyInformationalVersionAttribute)Attribute.GetCustomAttribute(Assembly.GetExecutingAssembly(),typeof(AssemblyInformationalVersionAttribute))).InformationalVersion;}}
    static bool arabic, loading, refreshing, running, echoSelected;
    static string echoState="unknown", deviceState="unknown";
    static string savedState;

    static string directory=AppDomain.CurrentDomain.BaseDirectory;
    static string data=Path.Combine(Environment.GetFolderPath(Environment.SpecialFolder.LocalApplicationData),"SonarFix");
    static JavaScriptSerializer json=new JavaScriptSerializer();
    static T Get<T>(string name) where T:class {return window.FindName(name) as T;}
    static Dictionary<string,Dictionary<string,string>> catalog;
    static string S(string key){
        if(catalog==null)using(var r=new StreamReader(Assembly.GetExecutingAssembly().GetManifestResourceStream("SonarFix.strings.json")))catalog=json.Deserialize<Dictionary<string,Dictionary<string,string>>>(r.ReadToEnd());
        return T(catalog[key]["en"],catalog[key]["ar"]);
    }
    static void Label(string name,string text){Get<TextBlock>(name).Text=text;}
    static void Content(string name,string text){var h=Get<HeaderedContentControl>(name);if(h!=null)h.Header=text;else Get<ContentControl>(name).Content=text;}
    static string T(string en,string ar){return arabic?System.Text.RegularExpressions.Regex.Replace(ar,@"Game & Chat|SonarFix|Discord|Windows|SteelSeries|Game|Chat",m=>"\u200E"+m.Value+"\u200E"):en;}
    static void Label(string name,string en,string ar){Get<TextBlock>(name).Text=T(en,ar);}
    static void Content(string name,string en,string ar){var h=Get<HeaderedContentControl>(name);if(h!=null)h.Header=T(en,ar);else Get<ContentControl>(name).Content=T(en,ar);}
    static bool Is(string name){return Get<CheckBox>(name).IsChecked==true;}
    static int Run(string arguments){
        commandCount++;
        var p=Process.Start(new ProcessStartInfo(Path.Combine(directory,"SonarFix.exe"),arguments){UseShellExecute=false,CreateNoWindow=true,WindowStyle=ProcessWindowStyle.Hidden});
        using(p){if(!p.WaitForExit(20000))throw new Exception(S("the_audio_operation_is_still_running_check_the_log_b"));return p.ExitCode;}
    }
    // Dark replacements for MessageBox, styled like the main window's buttons.
    [DllImport("dwmapi.dll")]static extern int DwmSetWindowAttribute(IntPtr hwnd,int attribute,ref int value,int size);

    static void DarkTitleBar(Window w){
        try{int on=1;var h=new WindowInteropHelper(w).Handle;if(DwmSetWindowAttribute(h,20,ref on,4)!=0)DwmSetWindowAttribute(h,19,ref on,4);}catch(Exception){}
    }
    static SolidColorBrush Brush(string hex){return (SolidColorBrush)new BrushConverter().ConvertFromString(hex);}
    static object LanguageGlyph(){using(var stream=Assembly.GetExecutingAssembly().GetManifestResourceStream("SonarFix.Language.xaml"))return XamlReader.Load(stream);}
    static void PersistLanguage(){if(!selfTesting)using(var key=Registry.CurrentUser.CreateSubKey(@"Software\SonarFix"))key.SetValue("UILanguage",arabic?"ar":"en");}
    static void ToggleLanguage(){arabic=!arabic;PersistLanguage();Get<TextBlock>("Feedback").Text="";ApplyLanguage();}
    static Button NewLanguageButton(){return new Button{Content=LanguageGlyph(),Style=(Style)window.Resources[typeof(Button)],Padding=new Thickness(5,2,5,2),Margin=new Thickness(8,0,0,0),Background=Brush("#1A2230")};}
    static Window NewDialog(){
        var d=new Window{Title="SonarFix",Background=Brush("#10151F"),Foreground=Brush("#EDF2FA"),FontFamily=new FontFamily("Segoe UI"),FontSize=14,ResizeMode=ResizeMode.NoResize,SizeToContent=SizeToContent.WidthAndHeight,ShowInTaskbar=false};
        if(window!=null&&window.IsVisible){d.Owner=window;d.WindowStartupLocation=WindowStartupLocation.CenterOwner;}else d.WindowStartupLocation=WindowStartupLocation.CenterScreen;
        d.Resources.MergedDictionaries.Add(window.Resources);d.SourceInitialized+=(s,e)=>DarkTitleBar(d);return d;
    }
    static Button DialogButton(string name,string caption,string color){return new Button{Name=name,Content=caption,Style=(Style)window.Resources[typeof(Button)],MinWidth=110,Background=Brush(color)};}
    static Window BuildDialog(string message,bool error){
        var d=NewDialog();d.FlowDirection=arabic?FlowDirection.RightToLeft:FlowDirection.LeftToRight;
        var root=new StackPanel{Margin=new Thickness(22),MaxWidth=440};
        root.Children.Add(new TextBlock{Text=message,TextWrapping=TextWrapping.Wrap,Margin=new Thickness(0,0,0,14)});
        var ok=DialogButton("Ok",S("ok"),error?"#8C3643":"#267B71");ok.IsDefault=true;ok.IsCancel=true;
        ok.Click+=(s,e)=>{if(d.IsVisible)d.DialogResult=true;};root.Children.Add(ok);d.Content=root;return d;
    }
    // Text and layout update in the existing modal, without recreating controls.
    static Window BuildConfirmation(string kind){
        bool about=kind=="about",danger=kind=="uninstall";
        var d=NewDialog();var root=new StackPanel{Margin=new Thickness(22),Width=440};
        var header=new DockPanel{Margin=new Thickness(0,0,0,14)};var lang=NewLanguageButton();lang.Name="DialogLanguage";DockPanel.SetDock(lang,Dock.Right);header.Children.Add(lang);
        var title=new TextBlock{Name="DialogTitle",FontWeight=FontWeights.SemiBold,FontSize=17,VerticalAlignment=VerticalAlignment.Center};
        if(danger){var warning=new TextBlock{Text="!",FontSize=22,FontWeight=FontWeights.Bold,Foreground=Brush("#F0808B"),Margin=new Thickness(0,0,12,0)};DockPanel.SetDock(warning,Dock.Left);header.Children.Add(warning);}
        header.Children.Add(title);root.Children.Add(header);
        var question=new TextBlock{Name="Question",TextWrapping=TextWrapping.Wrap,FontWeight=FontWeights.SemiBold,Margin=new Thickness(0,0,0,8)};root.Children.Add(question);
        var description=new TextBlock{Name="Description",TextWrapping=TextWrapping.Wrap,Foreground=Brush("#AAB8CD"),Margin=new Thickness(0,0,0,16),LineHeight=21};root.Children.Add(description);
        var buttons=new WrapPanel();var confirm=DialogButton("Confirm","",danger?"#8C3643":"#267B71");
        var cancel=DialogButton("Cancel","","#253349");cancel.Click+=(s,e)=>{if(d.IsVisible)d.DialogResult=false;};cancel.IsCancel=true;cancel.IsDefault=!about;confirm.IsDefault=about;cancel.Visibility=about?Visibility.Collapsed:Visibility.Visible;
        confirm.Click+=(s,e)=>{if(d.IsVisible)d.DialogResult=true;};buttons.Children.Add(confirm);buttons.Children.Add(cancel);root.Children.Add(buttons);d.Content=root;
        Action update=()=>{
            d.FlowDirection=arabic?FlowDirection.RightToLeft:FlowDirection.LeftToRight;
            d.Title=S(kind+"_title");title.Text=d.Title;
            question.Text=about?S("current_version")+": "+ProductVersion:S(kind+"_question");question.FlowDirection=about?FlowDirection.LeftToRight:d.FlowDirection;
            description.Text=S(kind+"_description");confirm.Content=S(about?"close":kind+"_action");cancel.Content=S("cancel");
            lang.ToolTip=S("switch_language");System.Windows.Automation.AutomationProperties.SetName(lang,S("switch_language"));
        };
        if(about)AttachAbout(root,d);
        Action translate=()=>{update();if(about&&updateAbout!=null)updateAbout();};
        d.Tag=translate;lang.Click+=(s,e)=>{ToggleLanguage();translate();};translate();
        d.Loaded+=(s,e)=>{if(about)confirm.Focus();else cancel.Focus();};return d;
    }
    static void Notify(string message){BuildDialog(message,false).ShowDialog();}
    static void Fail(string message){BuildDialog(message,true).ShowDialog();}
    static string SavedText(){return S("changes_saved");}
    static void StartWorker(){Process.Start(new ProcessStartInfo(Path.Combine(directory,"SonarFix.exe"),"--background"){UseShellExecute=false,CreateNoWindow=true,WindowStyle=ProcessWindowStyle.Hidden});}
    static void ApplyLanguage(){
        Get<StackPanel>("Root").FlowDirection=arabic?FlowDirection.RightToLeft:FlowDirection.LeftToRight;
        Get<DockPanel>("Shell").FlowDirection=arabic?FlowDirection.RightToLeft:FlowDirection.LeftToRight;

        Label("StatusTitle",S(running?"running":"stopped"));
        string statusText=S("status_1");
        if(!running){statusText=S("status_2");}
        else if(!echoSelected){statusText=S("status_3");}
        else if(echoState=="blocked"){statusText=S("status_4");}
        else if(echoState=="ready"){statusText=S("status_5");}
        else if(echoState=="waiting"){statusText=S("status_6");}
        else if(echoState=="ambiguous"){statusText=S("status_7");}
        else if(echoState=="device_unavailable"){statusText=S("status_device_unavailable");}
        else if(echoState=="unknown"){statusText=S("status_unavailable");}
        Label("Status",statusText);
        Content("Echo",S("prevent_duplicate_mic_audio"));
        Label("EchoDescription",S("prevents_duplicate_mic_audio_during_discord_screen_s"));
        Content("Disable",S("disable_game_chat"));
        Label("DisableDescription",S("disables_sonar_game_and_chat_outputs_while_keeping_y"));
        statusText=S("status_8");
        if(deviceState=="off"){statusText=S("status_9");}
        else if(deviceState=="disabled"){statusText=S("status_10");}
        else if(deviceState=="output_needed"){statusText=S("status_11");}
        else if(deviceState=="pending"){statusText=S("status_12");}
        else if(deviceState=="restore_pending"){statusText=S("status_13");}
        Label("DeviceStatus",statusText);
        Label("OutputLabel",S("audio_output_device"));
        Content("Redirect",S("set_as_windows_default_output"));
        Content("Startup",S("start_with_windows"));
        Content("ShowTray",S("show_tray"));Content("Health",S("check_devices"));
        Content("Save",S("save_changes"));
        Content("Stop",S(running?"stop":"start"));
        Content("Diagnostics",S("export_diagnostics"));Content("Logs",S("open_logs"));
        Content("About",S("about_sonarfix"));Content("Restore",S("restore_audio_settings"));
        Content("Advanced",S("advanced_settings"));Content("Uninstall",S("uninstall_sonarfix"));

        Get<TextBox>("Location").Text=directory;Get<TextBox>("Location").ToolTip=directory;
        foreach(var pair in new[]{new[]{"StartupSection","startup_section"},new[]{"MaintenanceSection","audio_diagnostics_section"},new[]{"ApplicationSection","application_section"},new[]{"UninstallSection","uninstall_section"},new[]{"FolderLabel","installation_folder"}})Label(pair[0],S(pair[1]));
        Get<Button>("LanguageToggle").ToolTip=S("switch_language");System.Windows.Automation.AutomationProperties.SetName(Get<Button>("LanguageToggle"),S("switch_language"));
        Get<Button>("OpenFolder").ToolTip=S("open_folder");Get<Button>("CopyFolder").ToolTip=S("copy_path");
        System.Windows.Automation.AutomationProperties.SetName(Get<Button>("OpenFolder"),S("open_folder"));System.Windows.Automation.AutomationProperties.SetName(Get<Button>("CopyFolder"),S("copy_path"));
        if(health!=null)Get<TextBlock>("HealthStatus").Text=S("health_"+health.Status);
        RefreshTray();
        Get<StackPanel>("OutputSettings").Visibility=Is("Disable")?Visibility.Visible:Visibility.Collapsed;
    }
    // The window grows with its content (Advanced, output options) up to the work area, then scrolls.
    static void FitHeight(){
        if(window.WindowState!=WindowState.Normal||!window.IsVisible)return;
        window.SizeToContent=SizeToContent.Height;
        window.Dispatcher.BeginInvoke(DispatcherPriority.Loaded,new Action(()=>{
            var area=SystemParameters.WorkArea;
            if(window.Top+window.ActualHeight>area.Bottom)window.Top=Math.Max(area.Top,area.Bottom-window.ActualHeight);
        }));
    }
    static Dictionary<string,string> ReadInfo(string path){
        var result=new Dictionary<string,string>();string section="";
        foreach(var line in File.ReadAllLines(path,Encoding.Unicode)){
            if(line.StartsWith("[")){section=line.Trim('[',']');continue;}
            int i=line.IndexOf('=');if(i>0)result[section+"/"+line.Substring(0,i)]=line.Substring(i+1);
        }return result;
    }
    static string Val(Dictionary<string,string> d,string k,string fallback){string v;return d.TryGetValue(k,out v)?v:fallback;}
    static Dictionary<string,string> SetupInfo(){
        string temp=Path.GetTempFileName();
        try{
            if(Run("--setup-info \""+temp+"\"")!=0)throw new Exception("Unable to inspect audio settings. Check SonarFix.log.");
            return ReadInfo(temp);
        }finally{File.Delete(temp);}
    }
    static void ReadStatus(Dictionary<string,string> info){

        echoState=Val(info,"Settings/EchoState","unknown"); deviceState=Val(info,"Settings/DeviceState","unknown");
        running=Val(info,"Settings/Running","false")=="true";
        echoSelected=Val(info,"Settings/Echo","false")=="true";
    }
    // Refresh only the status lines, never the checkboxes, so unsaved edits stay.
    static async void RefreshStatus(){
        if(refreshing||loading||!Get<DockPanel>("Shell").IsEnabled||!window.IsActive)return;
        refreshing=true;
        try{var info=await Task.Run(()=>SetupInfo());if(Get<DockPanel>("Shell").IsEnabled){ReadStatus(info);ApplyLanguage();}}
        catch(Exception){Get<TextBlock>("Status").Text=S("status_unavailable");Get<TextBlock>("DeviceStatus").Text=S("status_unavailable");}finally{refreshing=false;}
    }
    // Save is enabled only when the audio options differ from what was last loaded or saved.
    // Language and tray apply immediately and are not part of it.
    static string CurrentState(){var o=Get<ComboBox>("Output").SelectedItem as OutputDevice;return Is("Echo")+"|"+Is("Disable")+"|"+(o==null?"":o.ID)+"|"+Is("Redirect")+"|"+Is("Startup");}
    static bool HasChanges {get{return savedState!=null&&CurrentState()!=savedState;}}
    static void UpdateSave(){Get<Button>("Save").IsEnabled=HasChanges;}
    static void LoadSettings(){
        loading=true;
        try{
            var info=SetupInfo();arabic=Val(info,"Settings/Language","en")=="ar";
            ReadStatus(info);
            using(var key=Registry.CurrentUser.OpenSubKey(@"Software\SonarFix")){if(key!=null)arabic=(string)key.GetValue("UILanguage",arabic?"ar":"en")=="ar";}

            Get<CheckBox>("Echo").IsChecked=Val(info,"Settings/Echo","false")=="true";
            Get<CheckBox>("Disable").IsChecked=Val(info,"Settings/Disable","false")=="true";
            Get<CheckBox>("Redirect").IsChecked=Val(info,"Settings/Redirect","false")=="true";
            Get<CheckBox>("Startup").IsChecked=Val(info,"Settings/Startup","false")=="true";
            Get<CheckBox>("ShowTray").IsChecked=ui.show_tray;
            int count=int.Parse(Val(info,"Outputs/Count","0"));var outputs=new List<OutputDevice>();
            string chosen=Val(info,"Settings/Output","");if(chosen=="")chosen=Val(info,"Outputs/Default","");
            for(int i=0;i<count;i++)outputs.Add(new OutputDevice{ID=Val(info,"Output"+i+"/ID",""),Name=Val(info,"Output"+i+"/Name","")});
            if(chosen!=""&&!outputs.Exists(o=>o.ID==chosen))outputs.Add(new OutputDevice{ID=chosen,Name=S("unavailable_saved_output")+" — "+chosen});
            Get<ComboBox>("Output").ItemsSource=outputs;
            foreach(var o in outputs)if(o.ID==chosen)Get<ComboBox>("Output").SelectedItem=o;
            ApplyLanguage();
            savedState=CurrentState();
        }finally{loading=false;UpdateSave();}
    }
    static async void Save(object sender,RoutedEventArgs args){
        var output=Get<ComboBox>("Output").SelectedItem as OutputDevice;
        if(Is("Disable")&&output==null){Notify(S("select_an_active_physical_output_first"));return;}
        var request=new Dictionary<string,object>{{"fix_discord_echo",Is("Echo")},{"disable_game_and_chat",Is("Disable")},{"physical_output_id",output==null?"":output.ID},{"allow_default_redirect",Is("Redirect")},{"startup",Is("Startup")},{"language",arabic?"ar":"en"}};
        string temp=Path.GetTempFileName();Get<DockPanel>("Shell").IsEnabled=false;Get<TextBlock>("Feedback").Text=S("applying");
        try{
            File.WriteAllText(temp,json.Serialize(request),new UTF8Encoding(false));
            int code=await Task.Run(()=>Run("--configure \""+temp+"\""));
            if(code!=0)throw new Exception(S("options_were_not_fully_applied_check_sonarfix_log_fo"));
            if(Is("Echo")||Is("Disable"))StartWorker();
            LoadSettings();Get<TextBlock>("Feedback").Text=SavedText();
        }catch(Exception e){Get<TextBlock>("Feedback").Text="";Fail(e.Message);}finally{File.Delete(temp);Get<DockPanel>("Shell").IsEnabled=true;}
    }
    static string UninstallArguments {get{return "/SILENT /NORESTART /WAITPID="+Process.GetCurrentProcess().Id;}}
    static void Uninstall(){
        if(uninstalling||!Get<DockPanel>("Shell").IsEnabled)return;uninstalling=true;
        try{
            if(BuildConfirmation("uninstall").ShowDialog()!=true)return;
            var exe=Path.Combine(directory,"unins000.exe");
            if(!File.Exists(exe))throw new Exception(S("uninstaller_not_found_run_the_installer_to_repair_th"));
            Process.Start(new ProcessStartInfo(exe,UninstallArguments){UseShellExecute=true});
            forceExit=true;window.Close();
        }catch(Exception e){Fail(e.Message);}finally{uninstalling=false;}
    }
    static async void Stop(object sender,RoutedEventArgs args){
        Get<DockPanel>("Shell").IsEnabled=false;
        try{
            if(running){
                if(await Task.Run(()=>Run("--stop"))!=0)throw new Exception(S("sonarfix_did_not_stop_check_the_logs"));
            }else StartWorker();
            await Task.Delay(500);ReadStatus(await Task.Run(()=>SetupInfo()));ApplyLanguage();
        }catch(Exception e){Fail(e.Message);}finally{Get<DockPanel>("Shell").IsEnabled=true;}
    }
    static async void Diagnostics(object sender,RoutedEventArgs args){
        var dialog=new SaveFileDialog{Filter="JSON (*.json)|*.json",FileName="SonarFix-diagnostics.json"};
        if(dialog.ShowDialog()!=true)return;
        string path=dialog.FileName;Get<DockPanel>("Shell").IsEnabled=false;
        try{if(await Task.Run(()=>Run("--diagnose-file \""+path+"\""))!=0)throw new Exception(S("diagnostic_export_failed_check_sonarfix_log"));}
        catch(Exception e){Fail(e.Message);}finally{Get<DockPanel>("Shell").IsEnabled=true;}
    }
    static async void Restore(object sender,RoutedEventArgs args){
        if(BuildConfirmation("restore").ShowDialog()!=true)return;
        Get<DockPanel>("Shell").IsEnabled=false;string temp=Path.GetTempFileName();
        try{
            var info=await Task.Run(()=>SetupInfo());
            var request=new Dictionary<string,object>{{"fix_discord_echo",false},{"disable_game_and_chat",false},{"physical_output_id",Val(info,"Settings/Output","")},{"allow_default_redirect",Val(info,"Settings/Redirect","false")=="true"},{"startup",Val(info,"Settings/Startup","false")=="true"},{"language",Val(info,"Settings/Language","en")}};
            File.WriteAllText(temp,json.Serialize(request),new UTF8Encoding(false));
            if(await Task.Run(()=>Run("--configure \""+temp+"\""))!=0)throw new Exception(S("restoration_is_incomplete_recovery_records_are_retai"));
            info=await Task.Run(()=>SetupInfo());
            if(Val(info,"Settings/RestorePending","true")!="false")throw new Exception(S("some_audio_changes_are_still_pending_restoration_ope"));
            LoadSettings();Get<TextBlock>("Feedback").Text=S("recorded_audio_changes_restored_audio_features_are_o");
        }catch(Exception e){Fail(e.Message);LoadSettings();}finally{File.Delete(temp);Get<DockPanel>("Shell").IsEnabled=true;}
    }
    [STAThread]static int Main(string[] args){
        bool selfTest=args.Length==2&&args[0]=="--self-test";selfTesting=selfTest;
        bool requestUninstall=Array.IndexOf(args,"--uninstall")>=0;bool trayStartup=Array.IndexOf(args,"--tray")>=0;
        string mutexName=selfTest?@"Local\SonarFix_Settings_Test_"+Process.GetCurrentProcess().Id:@"Local\SonarFix_Settings";
        bool created;using(var mutex=new Mutex(true,mutexName,out created)){
            if(!created){
                if(trayStartup)return 0;
                string signalName=requestUninstall?@"Local\SonarFix_Settings_Uninstall":@"Local\SonarFix_Settings_Show";
                for(int i=0;i<50;i++)try{using(var signal=EventWaitHandle.OpenExisting(signalName))signal.Set();return 0;}catch(WaitHandleCannotBeOpenedException){Thread.Sleep(100);}
                return 1;
            }
            try{
                using(var stream=Assembly.GetExecutingAssembly().GetManifestResourceStream("SonarFix.Settings.xaml"))window=(Window)XamlReader.Load(stream);
                Get<Button>("Save").Click+=Save;
                foreach(var name in new[]{"Echo","Disable","Redirect","Startup"}){Get<CheckBox>(name).Checked+=(s,e)=>UpdateSave();Get<CheckBox>(name).Unchecked+=(s,e)=>UpdateSave();}
                Get<ComboBox>("Output").SelectionChanged+=(s,e)=>UpdateSave();
                Get<Button>("Stop").Click+=Stop;
                Get<Button>("Restore").Click+=Restore;
                Get<Button>("Health").Click+=async(s,e)=>await RefreshHealth(true);
                Get<CheckBox>("ShowTray").Checked+=TrayChanged;Get<CheckBox>("ShowTray").Unchecked+=TrayChanged;
                Get<Button>("About").Click+=(s,e)=>BuildConfirmation("about").ShowDialog();
                Get<Button>("LanguageToggle").Content=LanguageGlyph();Get<Button>("LanguageToggle").Click+=(s,e)=>ToggleLanguage();
                Get<Button>("OpenFolder").Click+=(s,e)=>Process.Start(new ProcessStartInfo(directory){UseShellExecute=true});
                Get<Button>("CopyFolder").Click+=(s,e)=>{Clipboard.SetText(directory);Get<TextBlock>("Feedback").Text=S("path_copied");};
                Get<CheckBox>("Disable").Checked+=(s,e)=>{ApplyLanguage();FitHeight();};Get<CheckBox>("Disable").Unchecked+=(s,e)=>{ApplyLanguage();FitHeight();};
                Get<Expander>("Advanced").Expanded+=(s,e)=>FitHeight();Get<Expander>("Advanced").Collapsed+=(s,e)=>FitHeight();
                // Two action columns, one when the window is too narrow for both.
                window.SizeChanged+=(s,e)=>{int columns=Get<StackPanel>("Root").ActualWidth<470?1:2;Get<UniformGrid>("AudioActions").Columns=columns;Get<UniformGrid>("ApplicationActions").Columns=columns;};
                Get<Button>("Logs").Click+=(s,e)=>{Directory.CreateDirectory(data);Process.Start(data);};
                Get<Button>("Diagnostics").Click+=Diagnostics;
                Get<Button>("Uninstall").Click+=(s,e)=>Uninstall();
                if(selfTest){RunSelfTests(args[1]);return 0;}
                return RunDesktop(requestUninstall,trayStartup);
            }catch(Exception e){if(selfTest){File.WriteAllText(args[1],e.ToString());return 1;}Fail(e.Message);return 1;}
        }
    }
}











