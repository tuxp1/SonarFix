using System;
using System.Diagnostics;
using System.Globalization;
using System.Net;
using System.Net.Http;
using System.Threading.Tasks;
using System.Windows;
using System.Windows.Controls;

partial class SettingsApp {
    static bool updateBusy;
    static UpdateResult updateResult;
    static Action updateAbout;
    static HttpClient updateHttp;
    static async Task CheckUpdates(bool manual){
        if(updateBusy)return;
        string repo=EffectiveRepository;
        if(String.IsNullOrEmpty(repo)){updateResult=new UpdateResult("unconfigured");if(updateAbout!=null)updateAbout();return;}
        if(!manual&&(!ui.automatic_updates||!ReleaseClient.MayCheck(ui.last_update_check,DateTime.UtcNow)))return;
        updateBusy=true;if(updateAbout!=null)updateAbout();
        try{
            ui.last_update_check=DateTime.UtcNow.ToString("o",CultureInfo.InvariantCulture);SaveUiPreferences();
            if(updateHttp==null){ServicePointManager.SecurityProtocol|=SecurityProtocolType.Tls12;updateHttp=new HttpClient(new HttpClientHandler{AllowAutoRedirect=false}){Timeout=TimeSpan.FromSeconds(8),MaxResponseContentBufferSize=1048576};}
            var result=await new ReleaseClient(updateHttp,ProductVersion).Check(repo);
            if(repo==EffectiveRepository)updateResult=result;
        }catch(Exception){updateResult=new UpdateResult("unavailable");}
        finally{updateBusy=false;if(updateAbout!=null)updateAbout();}
    }
    static async void MaybeAutoUpdate(){if(selfTesting||!ui.automatic_updates||String.IsNullOrEmpty(EffectiveRepository))return;await CheckUpdates(false);}
    static void AttachAbout(StackPanel root,Window d){
        var panel=new StackPanel{Margin=new Thickness(0,0,0,14)};
        var row=new WrapPanel{Margin=new Thickness(0,8,0,0)};var check=DialogButton("CheckUpdates","","#253349");check.IsDefault=false;var open=DialogButton("OpenRepository","GitHub","#253349");open.ToolTip="https://github.com/"+OfficialRepository;row.Children.Add(check);row.Children.Add(open);
        var automatic=new CheckBox{IsChecked=ui.automatic_updates,Margin=new Thickness(0,6,0,6)};
        var status=new TextBlock{Name="UpdateStatus",TextWrapping=TextWrapping.Wrap,Margin=new Thickness(0,8,0,5),Foreground=Brush("#AAB8CD")};
        var notes=new TextBlock{TextWrapping=TextWrapping.Wrap};var notesScroll=new ScrollViewer{Content=notes,MaxHeight=140,VerticalScrollBarVisibility=ScrollBarVisibility.Auto,HorizontalScrollBarVisibility=ScrollBarVisibility.Disabled,Focusable=false};
        var release=DialogButton("ViewRelease","","#267B71");release.IsDefault=false;release.HorizontalAlignment=HorizontalAlignment.Left;
        panel.Children.Add(automatic);panel.Children.Add(row);panel.Children.Add(status);panel.Children.Add(notesScroll);panel.Children.Add(release);
        root.Children.Insert(root.Children.Count-1,panel);
        Action update=()=>{
            System.Windows.Automation.AutomationProperties.SetName(open,S("open_repository"));
            automatic.Content=S("automatic_updates");check.Content=S("check_updates");check.IsEnabled=!updateBusy;
            status.Text=S(updateBusy?"update_checking":"update_"+(updateResult==null?(String.IsNullOrEmpty(EffectiveRepository)?"unconfigured":"not_checked"):updateResult.Status));
            bool available=updateResult!=null&&updateResult.Status=="available";
            if(available)status.Text+=" "+updateResult.Version;
            notes.Text=available?updateResult.Notes:"";notesScroll.Visibility=available&&!String.IsNullOrEmpty(notes.Text)?Visibility.Visible:Visibility.Collapsed;
            release.Content=S("view_release");release.Visibility=available?Visibility.Visible:Visibility.Collapsed;
        };
        automatic.Checked+=(s,e)=>{ui.automatic_updates=true;try{SaveUiPreferences();MaybeAutoUpdate();}catch(Exception){status.Text=S("preferences_failed");}};
        automatic.Unchecked+=(s,e)=>{ui.automatic_updates=false;try{SaveUiPreferences();}catch(Exception){status.Text=S("preferences_failed");}};
        check.Click+=async(s,e)=>await CheckUpdates(true);
        open.Click+=(s,e)=>Process.Start(new ProcessStartInfo("https://github.com/"+OfficialRepository){UseShellExecute=true});
        release.Click+=(s,e)=>{if(updateResult!=null&&updateResult.Status=="available")Process.Start(new ProcessStartInfo(updateResult.Url){UseShellExecute=true});};
        updateAbout=update;d.Closed+=(s,e)=>{if(updateAbout==update)updateAbout=null;};update();
        d.Content=new ScrollViewer{Content=root,MaxHeight=Math.Max(300,SystemParameters.WorkArea.Height-90),VerticalScrollBarVisibility=ScrollBarVisibility.Auto,HorizontalScrollBarVisibility=ScrollBarVisibility.Disabled,Focusable=false};
    }
}
