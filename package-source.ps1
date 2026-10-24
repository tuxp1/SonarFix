param([string]$OutputPath)
$ErrorActionPreference='Stop'
Add-Type -AssemblyName System.IO.Compression,System.IO.Compression.FileSystem
$root=$PSScriptRoot
$version=(Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()
if(!$OutputPath){$OutputPath=Join-Path $root ('dist\SonarFix-'+$version+'-source.zip')}
$OutputPath=[IO.Path]::GetFullPath($OutputPath)
if([IO.Path]::GetExtension($OutputPath) -ne '.zip'){throw 'Source output must be a ZIP file'}
$files=@()
foreach($dir in @('SonarFix-source','manager','installer','localization','LICENSES','tests','.github','docs')){
 $files+=@(Get-ChildItem -LiteralPath (Join-Path $root $dir) -File -Recurse | Where-Object {$_.Extension -notin @('.syso','.exe','.dll','.log','.tmp','.pfx','.p12','.pem','.key')})
}
foreach($name in @('VERSION','.gitignore','build.ps1','generate-version.ps1','generate-translations.ps1','test-ui.ps1','test-release.ps1','package-source.ps1','verify-release.ps1','README.md','README.ar.md','CONTRIBUTING.md','SECURITY.md','CHANGELOG.md','HANDOFF.md','TEST-REPORT.md','LICENSE','THIRD_PARTY_NOTICES.md')){$files+=Get-Item -LiteralPath (Join-Path $root $name)}
$secretPattern='gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{30,}|AKIA[0-9A-Z]{16}|-----BEGIN [A-Z ]*PRIVATE KEY-----'
foreach($file in $files){
 if($file.Extension -in @('.png','.ico')){continue}
 $text=Get-Content -Raw -LiteralPath $file.FullName
 if($text -match $secretPattern){throw ('Possible secret in '+$file.Name)}
 $normalized=$text.Replace('\\','\')
 if($normalized -match '(?i)[A-Z]:\\Users\\(?!u\\|fixture\\|Public\\|Default\\)[^\\\s]+\\'){throw ('Developer profile path in '+$file.Name)}
}
New-Item -ItemType Directory -Path (Split-Path $OutputPath -Parent) -Force | Out-Null
$temp=$OutputPath+'.'+[Guid]::NewGuid().ToString('N')+'.tmp'
$zip=[IO.Compression.ZipFile]::Open($temp,[IO.Compression.ZipArchiveMode]::Create)
try{foreach($file in $files){$relative=$file.FullName.Substring($root.Length+1).Replace('\','/');[IO.Compression.ZipFileExtensions]::CreateEntryFromFile($zip,$file.FullName,('SonarFix/'+$relative)) | Out-Null}}finally{$zip.Dispose()}
Move-Item -LiteralPath $temp -Destination $OutputPath -Force
# Record the source archive next to the build checksums from build.ps1.
$sums=Join-Path $root 'dist\SHA256.json'
if((Test-Path -LiteralPath $sums) -and ((Split-Path $OutputPath -Parent) -eq (Join-Path $root 'dist'))){
 $entries=@(Get-Content -Raw -LiteralPath $sums | ConvertFrom-Json | ForEach-Object {$_} | Where-Object {$_.File -ne (Split-Path $OutputPath -Leaf)})
 $entries+=[pscustomobject]@{Hash=(Get-FileHash -LiteralPath $OutputPath).Hash;File=(Split-Path $OutputPath -Leaf)}
 $entries | ConvertTo-Json | Set-Content -LiteralPath $sums
}
@{Result='PASS';Version=$version;Files=$files.Count;SensitiveDataExcluded=$true;SourceSHA256=(Get-FileHash -LiteralPath $OutputPath).Hash} | ConvertTo-Json | Set-Content -Encoding UTF8 -LiteralPath (Join-Path $root 'reports\source-hygiene.json')
Write-Output ('Source package: '+$OutputPath)
