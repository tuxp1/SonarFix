using System;
using System.IO;
using System.Text;
using System.Threading;

class UiPreferences {
    public bool show_tray {get;set;}
    public bool automatic_updates {get;set;}
    public string last_update_check {get;set;}
}
partial class SettingsApp {
    static UiPreferences ui=new UiPreferences();
    // Official update source. Fixed in the app; users cannot change it from the UI.
    const string OfficialRepository="tuxp1/SonarFix";
    static string EffectiveRepository {get{return OfficialRepository;}}
    static void LoadUiPreferences(){
        var path=Path.Combine(data,"ui-settings.json");
        try{if(File.Exists(path))ui=json.Deserialize<UiPreferences>(File.ReadAllText(path))??new UiPreferences();}catch(IOException){ui=new UiPreferences();}catch(ArgumentException){ui=new UiPreferences();}catch(InvalidOperationException){ui=new UiPreferences();}
    }
    static void SaveUiPreferences(){
        if(selfTesting)return;
        Directory.CreateDirectory(data);string path=Path.Combine(data,"ui-settings.json"),temp=path+"."+Guid.NewGuid().ToString("N")+".tmp";
        try{
            File.WriteAllText(temp,json.Serialize(ui),new UTF8Encoding(false));
            for(int attempt=0;;attempt++)try{if(File.Exists(path))File.Replace(temp,path,null);else File.Move(temp,path);break;}catch(IOException){if(attempt>=4)throw;Thread.Sleep(50);}
        }finally{if(File.Exists(temp))File.Delete(temp);}
    }
}
