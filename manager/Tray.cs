using System;
using System.ComponentModel;
using System.Diagnostics;
using System.IO;
using System.Runtime.InteropServices;
using System.Threading;
using System.Threading.Tasks;
using System.Windows;
using System.Windows.Controls;
using System.Windows.Interop;
using System.Windows.Threading;
using Forms=System.Windows.Forms;

partial class SettingsApp {
    static Forms.NotifyIcon tray;
    static System.Drawing.Icon trayIcon;
    static Forms.ToolStripMenuItem trayStatus,trayOpen,trayStop,trayExit;
    static bool forceExit,trayActionBusy;
    [DllImport("kernel32.dll",CharSet=CharSet.Unicode,SetLastError=true)]static extern IntPtr OpenEvent(uint access,bool inherit,string name);
    [DllImport("kernel32.dll")]static extern bool CloseHandle(IntPtr handle);
    static bool ProbeRunning(){var h=OpenEvent(2,false,@"Local\SonarFix_DiscordSonarMic_v1_Stop");if(h==IntPtr.Zero)return false;CloseHandle(h);return true;}
    static bool HideOnClose(bool enabled,bool exiting){return enabled&&!exiting;}
    static void OpenMain(){if(window.Dispatcher.HasShutdownStarted)return;window.Show();if(window.WindowState==WindowState.Minimized)window.WindowState=WindowState.Normal;window.Activate();}
    static void DisposeTray(){if(tray!=null){tray.Visible=false;if(tray.ContextMenuStrip!=null)tray.ContextMenuStrip.Dispose();tray.Dispose();tray=null;}if(trayIcon!=null){trayIcon.Dispose();trayIcon=null;}}
    static void RefreshTray(){
        if(tray==null)return;
        tray.ContextMenuStrip.RightToLeft=arabic?Forms.RightToLeft.Yes:Forms.RightToLeft.No;
        bool available=!trayActionBusy&&!uninstalling&&Get<System.Windows.Controls.DockPanel>("Shell").IsEnabled;
        trayStatus.Text=S(running?"running":"stopped");trayOpen.Text=S("open_sonarfix");trayStop.Text=S("stop");trayStop.Enabled=running&&available;trayExit.Text=S("exit_sonarfix");trayExit.Enabled=available;
        string text="SonarFix — "+S(running?"running":"stopped");tray.Text=text.Length>63?text.Substring(0,63):text;
    }
    static void ApplyTray(){
        if(!ui.show_tray){DisposeTray();return;}
        if(tray==null){
            trayIcon=System.Drawing.Icon.ExtractAssociatedIcon(Path.Combine(directory,"SonarFix.Settings.exe"));
            tray=new Forms.NotifyIcon{Icon=trayIcon};
            var menu=new Forms.ContextMenuStrip{BackColor=System.Drawing.Color.FromArgb(26,34,48),ForeColor=System.Drawing.Color.White,ShowImageMargin=false};
            trayStatus=new Forms.ToolStripMenuItem{Enabled=false};trayOpen=new Forms.ToolStripMenuItem();trayStop=new Forms.ToolStripMenuItem();trayExit=new Forms.ToolStripMenuItem();
            menu.Items.AddRange(new Forms.ToolStripItem[]{trayStatus,new Forms.ToolStripSeparator(),trayOpen,trayStop,trayExit});
            menu.Opening+=(s,e)=>{running=ProbeRunning();RefreshTray();};
            trayOpen.Click+=(s,e)=>OpenMain();tray.DoubleClick+=(s,e)=>OpenMain();
            trayStop.Click+=async(s,e)=>await StopFromTray(false);trayExit.Click+=async(s,e)=>await StopFromTray(true);
            tray.ContextMenuStrip=menu;tray.Visible=!selfTesting;
        }RefreshTray();
    }
    static async Task StopFromTray(bool exit){
        if(trayActionBusy||uninstalling||!Get<System.Windows.Controls.DockPanel>("Shell").IsEnabled)return;
        trayActionBusy=true;Get<System.Windows.Controls.DockPanel>("Shell").IsEnabled=false;RefreshTray();
        try{
            if(await Task.Run(()=>Run("--stop"))!=0)throw new Exception(S("sonarfix_did_not_stop_check_the_logs"));
            running=ProbeRunning();ApplyLanguage();
            if(exit){forceExit=true;DisposeTray();Application.Current.Shutdown();}
        }catch(Exception e){OpenMain();Fail(e.Message);}finally{trayActionBusy=false;Get<System.Windows.Controls.DockPanel>("Shell").IsEnabled=true;RefreshTray();}
    }
    static void TrayChanged(object sender,RoutedEventArgs e){
        if(loading)return;bool previous=ui.show_tray;ui.show_tray=Is("ShowTray");
        try{ApplyTray();SaveUiPreferences();}catch(Exception ex){ui.show_tray=previous;loading=true;Get<CheckBox>("ShowTray").IsChecked=previous;loading=false;ApplyTray();Fail(ex.Message);}
    }
    static void Post(Action action){if(!window.Dispatcher.HasShutdownStarted)try{window.Dispatcher.BeginInvoke(action);}catch(InvalidOperationException){}}
    static int RunDesktop(bool requestUninstall,bool trayStartup){
        LoadUiPreferences();if(trayStartup&&!ui.show_tray)return 0;
        var app=new Application{ShutdownMode=ShutdownMode.OnExplicitShutdown,MainWindow=window};
        using(var show=new EventWaitHandle(false,EventResetMode.AutoReset,@"Local\SonarFix_Settings_Show"))
        using(var remove=new EventWaitHandle(false,EventResetMode.AutoReset,@"Local\SonarFix_Settings_Uninstall")){
            var showWait=ThreadPool.RegisterWaitForSingleObject(show,(s,t)=>Post(()=>OpenMain()),null,Timeout.Infinite,false);
            var removeWait=ThreadPool.RegisterWaitForSingleObject(remove,(s,t)=>Post(()=>{OpenMain();Uninstall();}),null,Timeout.Infinite,false);
            var statusDebounce=new DispatcherTimer{Interval=TimeSpan.FromMilliseconds(500)};
            statusDebounce.Tick+=(s,e)=>{statusDebounce.Stop();running=ProbeRunning();ApplyLanguage();};
            var deviceDebounce=new DispatcherTimer{Interval=TimeSpan.FromMilliseconds(900)};
            deviceDebounce.Tick+=async(s,e)=>{deviceDebounce.Stop();if(window.IsVisible||ui.show_tray)await RefreshHealth(false);};
            var statusTimer=new DispatcherTimer{Interval=TimeSpan.FromSeconds(10)};statusTimer.Tick+=(s,e)=>RefreshStatus();
            var lightTimer=new DispatcherTimer{Interval=TimeSpan.FromMinutes(1)};lightTimer.Tick+=(s,e)=>{running=ProbeRunning();RefreshTray();MaybeAutoUpdate();};
            Directory.CreateDirectory(data);
            using(var watcher=new FileSystemWatcher(data,"status.json")){
                FileSystemEventHandler statusChanged=(s,e)=>Post(()=>{statusDebounce.Stop();statusDebounce.Start();});
                watcher.NotifyFilter=NotifyFilters.FileName|NotifyFilters.LastWrite;watcher.Changed+=statusChanged;watcher.Created+=statusChanged;watcher.Deleted+=statusChanged;watcher.Renamed+=(s,e)=>statusChanged(s,e);watcher.EnableRaisingEvents=true;
                HwndSource source=null;
                HwndSourceHook hook=(IntPtr h,int message,IntPtr w,IntPtr l,ref bool handled)=>{if(message==0x219){deviceDebounce.Stop();deviceDebounce.Start();}return IntPtr.Zero;};
                window.SourceInitialized+=(s,e)=>{DarkTitleBar(window);source=HwndSource.FromHwnd(new WindowInteropHelper(window).Handle);if(source!=null)source.AddHook(hook);};
                window.Closing+=(s,e)=>{if(HideOnClose(ui.show_tray,forceExit)){e.Cancel=true;window.Hide();}};
                window.Closed+=(s,e)=>app.Shutdown();window.Activated+=(s,e)=>RefreshStatus();
                try{
                    LoadSettings();ApplyTray();statusTimer.Start();lightTimer.Start();
                    window.MaxHeight=SystemParameters.WorkArea.Height;
                    if(trayStartup)new WindowInteropHelper(window).EnsureHandle();else window.Show();
                    if(requestUninstall)Post(()=>Uninstall());MaybeAutoUpdate();app.Run();return 0;
                }finally{
                    statusTimer.Stop();lightTimer.Stop();statusDebounce.Stop();deviceDebounce.Stop();showWait.Unregister(null);removeWait.Unregister(null);
                    watcher.EnableRaisingEvents=false;if(source!=null)source.RemoveHook(hook);DisposeTray();
                }
            }
        }
    }
}
