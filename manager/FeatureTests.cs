using System;
using System.Collections.Generic;
using System.Net;
using System.Net.Http;
using System.Threading;
using System.Threading.Tasks;
using System.Windows;
using System.Windows.Controls;

sealed class FixtureHttp : HttpMessageHandler {
    public int Calls; public bool Offline; public HttpStatusCode Code=HttpStatusCode.OK; public string Body;
    protected override Task<HttpResponseMessage> SendAsync(HttpRequestMessage request,CancellationToken cancellation){
        Calls++;if(request.RequestUri.Scheme!="https"||request.RequestUri.Host!="api.github.com")throw new Exception("Unsafe update endpoint");
        if(Offline)throw new HttpRequestException("Offline fixture");
        return Task.FromResult(new HttpResponseMessage(Code){Content=new StringContent(Body??"")});
    }
}
partial class SettingsApp {
    static void RunFeatureTests(List<string> results){
        Check(!new UiPreferences().show_tray&&!new UiPreferences().automatic_updates,"New options must default off");
        int calls=commandCount;bool wasRunning=running;var original=ui;ui=new UiPreferences();
        Get<CheckBox>("ShowTray").IsChecked=true;Check(ui.show_tray&&tray!=null&&tray.Icon!=null,"Tray icon not created");
        Check(!tray.Visible,"Test should not add a real shell icon");
        Get<CheckBox>("ShowTray").IsChecked=false;Check(!ui.show_tray&&tray==null,"Tray not disposed");
        Check(running==wasRunning&&calls==commandCount,"Tray toggle affected audio");
        Get<DockPanel>("Shell").IsEnabled=false;StopFromTray(false).GetAwaiter().GetResult();Uninstall();Get<DockPanel>("Shell").IsEnabled=true;
        Check(commandCount==calls,"Tray/uninstall interrupted another audio operation");
        Check(HideOnClose(true,false)&&!HideOnClose(false,false)&&!HideOnClose(true,true),"UI/tray exit policy");
        var round=json.Deserialize<UiPreferences>(json.Serialize(new UiPreferences{show_tray=true}));Check(round.show_tray,"UI prefs round trip");var legacy=json.Deserialize<UiPreferences>("{\"show_tray\":true,\"repository\":\"someone/else\"}");Check(legacy.show_tray&&EffectiveRepository=="tuxp1/SonarFix","Legacy repository setting ignored");ui=original;
        string[] ordered={"1.0.0-alpha","1.0.0-alpha.1","1.0.0-alpha.beta","1.0.0-beta","1.0.0-beta.2","1.0.0-beta.11","1.0.0-rc.1","1.0.0"};
        for(int i=1;i<ordered.Length;i++)Check(SemVersion.Parse(ordered[i]).CompareTo(SemVersion.Parse(ordered[i-1]))>0,"SemVer prerelease ordering");
        Check(SemVersion.Parse("v0.2").CompareTo(SemVersion.Parse("0.2.0+build.7"))==0,"Legacy two-part version/build metadata");
        Check(SemVersion.Parse("0.10").CompareTo(SemVersion.Parse("0.9.9"))>0,"Numeric SemVer ordering");
        Check(SemVersion.Parse("1.0.0--1").CompareTo(SemVersion.Parse("1.0.0-1"))>0,"Alphanumeric prerelease identifier");
        foreach(var invalid in new[]{"01.0.0","1.0.0-01","1.0.0+","version two","1.0.0.4"})Check(SemVersion.Parse(invalid)==null,"Invalid version accepted");
        Check(ReleaseClient.NormalizeRepository("")==""&&ReleaseClient.NormalizeRepository("https://github.com/fixture/fixture")=="fixture/fixture","Repository config");
        foreach(var invalid in new[]{"http://github.com/fixture/fixture","https://evil.invalid/fixture/fixture","fixture/../x","fixture/repo?token=secret"})Check(ReleaseClient.NormalizeRepository(invalid)==null,"Unsafe repo accepted");
        var fixture=new FixtureHttp();using(var http=new HttpClient(fixture)){
            var client=new ReleaseClient(http,"0.2");
            Check(client.Check("").GetAwaiter().GetResult().Status=="unconfigured"&&fixture.Calls==0,"Unconfigured repository contacted GitHub");
            string stable="{\"draft\":false,\"prerelease\":false,\"tag_name\":\"v0.3.0\",\"html_url\":\"https://github.com/fixture/fixture/releases/tag/v0.3.0\",\"body\":\"fixture notes\"}";
            fixture.Body=stable;var release=client.Check("fixture/fixture").GetAwaiter().GetResult();Check(release.Status=="available"&&release.Notes=="fixture notes","Stable release parsing");
            fixture.Body=stable.Replace("v0.3.0","v0.2");Check(client.Check("fixture/fixture").GetAwaiter().GetResult().Status=="current","Current version comparison");
            fixture.Body=stable.Replace("\"draft\":false","\"draft\":true");Check(client.Check("fixture/fixture").GetAwaiter().GetResult().Status=="ignored","Draft accepted");
            fixture.Body=stable.Replace("\"prerelease\":false","\"prerelease\":true");Check(client.Check("fixture/fixture").GetAwaiter().GetResult().Status=="ignored","Prerelease accepted");
            fixture.Body=stable.Replace("v0.3.0","v0.3.0-rc.1");Check(client.Check("fixture/fixture").GetAwaiter().GetResult().Status=="ignored","Unmarked prerelease tag accepted");
            fixture.Body=stable.Replace("https://github.com/","http://github.com/");Check(client.Check("fixture/fixture").GetAwaiter().GetResult().Status=="unavailable","Insecure release link accepted");
            fixture.Body="broken json";Check(client.Check("fixture/fixture").GetAwaiter().GetResult().Status=="unavailable","Bad response not contained");
            fixture.Code=HttpStatusCode.Forbidden;Check(client.Check("fixture/fixture").GetAwaiter().GetResult().Status=="unavailable","Rate limit not contained");
            fixture.Offline=true;Check(client.Check("fixture/fixture").GetAwaiter().GetResult().Status=="unavailable","Offline not contained");
        }
        DateTime now=DateTime.UtcNow;
        Check(ReleaseClient.MayCheck(null,now)&&!ReleaseClient.MayCheck(now.ToString("o"),now.AddHours(23))&&ReleaseClient.MayCheck(now.ToString("o"),now.AddDays(1)),"Daily update throttling");
        Check(commandCount==calls&&running==wasRunning,"Update failure changed monitoring");
        var h=new HealthSnapshot{Status="ready",Sonar=true,Gaming="disabled",Chat="disabled",Microphone="active",Output="active"};Check(HealthSummary(h).Contains(S("health_ready")),"Health summary translation");
        results.Add("Tray default/off/on/dispose and lifecycle; SemVer ordering; stable/draft/prerelease; HTTPS; missing repo/offline/rate-limit/malformed response; daily throttle; no audio commands: PASS");
    }
}
