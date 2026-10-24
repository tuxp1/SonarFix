using System;
using System.IO;
using System.Threading.Tasks;
using System.Windows;
using System.Windows.Controls;

class HealthSnapshot {
    public string Status {get;set;} public string Gaming {get;set;} public string Chat {get;set;} public string Microphone {get;set;} public string Output {get;set;}
    public bool Sonar {get;set;} public bool DisableExpected {get;set;} public bool WorkerRunning {get;set;}
}
partial class SettingsApp {
    static bool healthBusy;
    static HealthSnapshot health;
    static string HealthSummary(HealthSnapshot h){
        if(h==null)return S("health_unknown");
        string text=S("health_"+h.Status)+Environment.NewLine;
        text+="SteelSeries Sonar: "+S(h.Sonar?"health_present":"health_unavailable")+Environment.NewLine;
        text+="Gaming: "+S("device_"+h.Gaming)+Environment.NewLine+"Chat: "+S("device_"+h.Chat)+Environment.NewLine;
        text+="Sonar Microphone: "+S("device_"+h.Microphone)+Environment.NewLine+S("audio_output_device")+": "+S("device_"+h.Output);
        if(h.Status=="action")text+=Environment.NewLine+S(h.DisableExpected?(h.WorkerRunning?"health_monitor_retry":"health_start_monitor"):"health_review_settings");
        return text;
    }
    // Read-only result card: overall state, one row per device and the next step.
    static Window BuildHealthDialog(HealthSnapshot h){
        var d=NewDialog();d.FlowDirection=arabic?FlowDirection.RightToLeft:FlowDirection.LeftToRight;
        var root=new StackPanel{Margin=new Thickness(22),Width=380};
        root.Children.Add(new TextBlock{Text=S("health_title"),FontWeight=FontWeights.SemiBold,FontSize=17,Margin=new Thickness(0,0,0,4)});
        string state=h==null?"unknown":h.Status;
        root.Children.Add(new TextBlock{Text=S("health_"+state),TextWrapping=TextWrapping.Wrap,FontWeight=FontWeights.SemiBold,Margin=new Thickness(0,0,0,12),Foreground=Brush(state=="ready"?"#71D6C8":state=="action"?"#F2C46D":"#F0808B")});
        if(h!=null){
            var grid=new System.Windows.Controls.Grid();grid.ColumnDefinitions.Add(new System.Windows.Controls.ColumnDefinition{Width=GridLength.Auto});grid.ColumnDefinitions.Add(new System.Windows.Controls.ColumnDefinition());
            var rows=new[]{new[]{S("health_sonar"),S(h.Sonar?"health_present":"health_unavailable"),h.Sonar?"ok":"bad"},new[]{T("Game","Game"),S("device_"+h.Gaming),h.Gaming},new[]{T("Chat","Chat"),S("device_"+h.Chat),h.Chat},new[]{S("health_microphone"),S("device_"+h.Microphone),h.Microphone},new[]{S("audio_output_device"),S("device_"+h.Output),h.Output}};
            for(int i=0;i<rows.Length;i++){
                grid.RowDefinitions.Add(new System.Windows.Controls.RowDefinition());
                var name=new System.Windows.Controls.TextBlock{Text=rows[i][0],Foreground=Brush("#AAB8CD"),Margin=new Thickness(0,4,18,4)};
                string tone=rows[i][2]=="unavailable"||rows[i][2]=="bad"?"#F0808B":rows[i][2]=="mixed"?"#F2C46D":"#EDF2FA";
                var value=new System.Windows.Controls.TextBlock{Text=rows[i][1],Foreground=Brush(tone),FontWeight=FontWeights.SemiBold,Margin=new Thickness(0,4,0,4)};
                System.Windows.Controls.Grid.SetRow(name,i);System.Windows.Controls.Grid.SetRow(value,i);System.Windows.Controls.Grid.SetColumn(value,1);grid.Children.Add(name);grid.Children.Add(value);
            }
            root.Children.Add(new System.Windows.Controls.Border{Background=Brush("#1A2230"),CornerRadius=new CornerRadius(8),Padding=new Thickness(14,8,14,8),Margin=new Thickness(0,0,0,12),Child=grid});
            if(h.Status=="action")root.Children.Add(new System.Windows.Controls.TextBlock{Text=S(h.DisableExpected?(h.WorkerRunning?"health_monitor_retry":"health_start_monitor"):"health_review_settings"),TextWrapping=TextWrapping.Wrap,Foreground=Brush("#AAB8CD"),Margin=new Thickness(0,0,0,14)});
        }
        var close=DialogButton("Confirm",S("close"),"#267B71");close.IsDefault=true;close.IsCancel=true;close.HorizontalAlignment=HorizontalAlignment.Left;
        close.Click+=(s,e)=>{if(d.IsVisible)d.DialogResult=true;};root.Children.Add(close);d.Content=root;
        d.Title=S("health_title");d.Loaded+=(s,e)=>close.Focus();return d;
    }
    static async Task RefreshHealth(bool show){
        if(healthBusy)return;healthBusy=true;string temp=Path.GetTempFileName();
        try{
            if(await Task.Run(()=>Run("--health-info \""+temp+"\""))!=0)throw new IOException("Health inspection failed");
            health=json.Deserialize<HealthSnapshot>(File.ReadAllText(temp));
            if(health==null||Array.IndexOf(new[]{"ready","action","unavailable"},health.Status)<0)throw new IOException("Invalid health result");
            Get<System.Windows.Controls.TextBlock>("HealthStatus").Text=S("health_"+health.Status);
            if(show)BuildHealthDialog(health).ShowDialog();
        }catch(Exception){health=null;Get<System.Windows.Controls.TextBlock>("HealthStatus").Text=S("health_unknown");if(show)BuildHealthDialog(null).ShowDialog();}
        finally{File.Delete(temp);healthBusy=false;}
    }
}
