using System;
using System.Collections.Generic;
using System.Globalization;
using System.Net;
using System.Net.Http;
using System.Numerics;
using System.Text.RegularExpressions;
using System.Threading.Tasks;
using System.Web.Script.Serialization;

sealed class SemVersion : IComparable<SemVersion> {
    public BigInteger Major,Minor,Patch;
    public string[] Pre;
    public bool Stable {get{return Pre.Length==0;}}
    public static SemVersion Parse(string text){
        if(text==null||text.Length>256)return null;
        var m=Regex.Match(text.Trim(),@"^[vV]?(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:\.(0|[1-9][0-9]*))?(?:-([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?(?:\+([0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*))?$");
        if(!m.Success)return null;
        var pre=m.Groups[4].Success?m.Groups[4].Value.Split('.'):new string[0];
        foreach(var p in pre)if(Regex.IsMatch(p,@"^\d+$")&&p.Length>1&&p[0]=='0')return null;
        return new SemVersion{Major=BigInteger.Parse(m.Groups[1].Value,CultureInfo.InvariantCulture),Minor=BigInteger.Parse(m.Groups[2].Value,CultureInfo.InvariantCulture),Patch=BigInteger.Parse(m.Groups[3].Success?m.Groups[3].Value:"0",CultureInfo.InvariantCulture),Pre=pre};
    }
    public int CompareTo(SemVersion b){
        int n=Major.CompareTo(b.Major);if(n!=0)return n;n=Minor.CompareTo(b.Minor);if(n!=0)return n;n=Patch.CompareTo(b.Patch);if(n!=0)return n;
        if(Stable||b.Stable)return Stable==b.Stable?0:(Stable?1:-1);
        for(int i=0;i<Math.Min(Pre.Length,b.Pre.Length);i++){
            BigInteger aNum=0,bNum=0;bool a=Regex.IsMatch(Pre[i],@"^[0-9]+$"),bb=Regex.IsMatch(b.Pre[i],@"^[0-9]+$");
            if(a)aNum=BigInteger.Parse(Pre[i],CultureInfo.InvariantCulture);if(bb)bNum=BigInteger.Parse(b.Pre[i],CultureInfo.InvariantCulture);
            n=a&&bb?aNum.CompareTo(bNum):(a!=bb?(a?-1:1):String.CompareOrdinal(Pre[i],b.Pre[i]));if(n!=0)return n;
        }return Pre.Length.CompareTo(b.Pre.Length);
    }
}
class UpdateResult {
    public string Status,Version,Notes,Url;
    public UpdateResult(string status){Status=status;}
}
sealed class ReleaseClient {
    readonly HttpClient http;
    readonly string current;
    public ReleaseClient(HttpClient client,string currentVersion){http=client;current=currentVersion;}
    public static string NormalizeRepository(string text){
        text=(text??"").Trim();if(text=="")return "";
        if(text.StartsWith("https://github.com/",StringComparison.OrdinalIgnoreCase))text=text.Substring(19).TrimEnd('/');
        if(!Regex.IsMatch(text,@"^[A-Za-z0-9](?:[A-Za-z0-9-]{0,38})/[A-Za-z0-9_.-]{1,100}$"))return null;
        var name=text.Substring(text.IndexOf('/')+1);if(name=="."||name=="..")return null;return text;
    }
    public static bool MayCheck(string last,DateTime now){DateTime parsed;return !DateTime.TryParse(last,CultureInfo.InvariantCulture,DateTimeStyles.RoundtripKind,out parsed)||(now-parsed.ToUniversalTime()).TotalHours>=24||parsed.ToUniversalTime()>now.AddDays(1);}
    public static UpdateResult Decode(string raw,string repo,string current){
        var d=new JavaScriptSerializer{MaxJsonLength=1048576}.Deserialize<Dictionary<string,object>>(raw);
        if(d==null||!d.ContainsKey("draft")||!d.ContainsKey("prerelease")||!(d["draft"] is bool)||!(d["prerelease"] is bool))return new UpdateResult("unavailable");
        if((bool)d["draft"]||(bool)d["prerelease"])return new UpdateResult("ignored");
        object value;string tag=d.TryGetValue("tag_name",out value)?value as string:null;
        var latest=SemVersion.Parse(tag);var installed=SemVersion.Parse(current);
        if(latest==null||!latest.Stable||installed==null)return new UpdateResult("ignored");
        string url=d.TryGetValue("html_url",out value)?value as string:null;
        Uri uri;if(!Uri.TryCreate(url,UriKind.Absolute,out uri)||uri.Scheme!="https"||!String.Equals(uri.Host,"github.com",StringComparison.OrdinalIgnoreCase)||!String.IsNullOrEmpty(uri.UserInfo)||!uri.AbsolutePath.StartsWith("/"+repo+"/releases/",StringComparison.OrdinalIgnoreCase))return new UpdateResult("unavailable");
        string notes=d.TryGetValue("body",out value)?value as string:"";notes=notes??"";if(notes.Length>8000)notes=notes.Substring(0,8000)+"…";
        return new UpdateResult(latest.CompareTo(installed)>0?"available":"current"){Version=tag,Url=uri.AbsoluteUri,Notes=notes};
    }
    public async Task<UpdateResult> Check(string repository){
        string repo=NormalizeRepository(repository);if(String.IsNullOrEmpty(repo))return new UpdateResult("unconfigured");
        try{
            using(var request=new HttpRequestMessage(HttpMethod.Get,"https://api.github.com/repos/"+repo+"/releases/latest")){
                request.Headers.UserAgent.ParseAdd("SonarFix/"+current);request.Headers.Accept.ParseAdd("application/vnd.github+json");request.Headers.Add("X-GitHub-Api-Version","2026-03-10");
                using(var response=await http.SendAsync(request).ConfigureAwait(false)){
                    if(!response.IsSuccessStatusCode)return new UpdateResult("unavailable");
                    return Decode(await response.Content.ReadAsStringAsync().ConfigureAwait(false),repo,current);
                }
            }
        }catch(HttpRequestException){return new UpdateResult("unavailable");}catch(TaskCanceledException){return new UpdateResult("unavailable");}catch(ArgumentException){return new UpdateResult("unavailable");}catch(InvalidOperationException){return new UpdateResult("unavailable");}
    }
}
