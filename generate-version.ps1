$ErrorActionPreference='Stop'
$root=$PSScriptRoot
$version=(Get-Content -LiteralPath (Join-Path $root 'VERSION') -Raw).Trim()
if($version -notmatch '^\d+\.\d+(\.\d+){0,2}$'){throw 'Invalid VERSION'}
$parts=@($version.Split('.'));while($parts.Count -lt 4){$parts+='0'}
$numeric=$parts -join '.'
# UTF-8 without BOM: go-winres and the Go toolchain reject a leading BOM, and
# Windows PowerShell 5.1 Set-Content -Encoding UTF8 always writes one.
function Write-Utf8([string]$Path,[string]$Text){[IO.File]::WriteAllText($Path,$Text,(New-Object Text.UTF8Encoding $false))}
@"
// Generated from VERSION. Do not edit.
using System.Reflection;
[assembly: AssemblyVersion("$numeric")]
[assembly: AssemblyFileVersion("$numeric")]
[assembly: AssemblyInformationalVersion("$version")]
"@ | ForEach-Object { Write-Utf8 (Join-Path $root 'manager\Version.cs') ($_+"`r`n") }
@"
//go:build windows

// Code generated from VERSION. DO NOT EDIT.
package main

const appVersion = "$version"
"@ | ForEach-Object { Write-Utf8 (Join-Path $root 'SonarFix-source\version.go') ($_+"`r`n") }
@"
; Generated from VERSION. Do not edit.
#define AppVersion "$version"
#define NumericVersion "$numeric"
"@ | ForEach-Object { Write-Utf8 (Join-Path $root 'installer\Version.iss') ($_+"`r`n") }
$p=Join-Path $root 'manager\app.manifest';$s=Get-Content -LiteralPath $p -Raw
$s=[regex]::Replace($s,'(<assemblyIdentity version=")[^"]+',('${1}'+$numeric))
Write-Utf8 $p $s
$p=Join-Path $root 'SonarFix-source\winres\winres.json';$c=Get-Content -Raw -LiteralPath $p | ConvertFrom-Json
$c.RT_MANIFEST.'#1'.'0409'.identity.version=$numeric
$v=$c.RT_VERSION.'#1'.'0409';$v.fixed.file_version=$numeric;$v.fixed.product_version=$numeric
$v.info.'0409'.FileVersion=$version;$v.info.'0409'.ProductVersion=$version
Write-Utf8 $p (($c | ConvertTo-Json -Depth 16)+"`r`n")
