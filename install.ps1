$ErrorActionPreference = "Stop"

$repo = "judahpaul16/plume"
$asset = "plume-windows-amd64.exe"
$url = "https://github.com/$repo/releases/latest/download/$asset"
$dest = Join-Path $env:LOCALAPPDATA "Programs\plume"

New-Item -ItemType Directory -Force -Path $dest | Out-Null
$out = Join-Path $dest "plume.exe"

Write-Host "plume: downloading $asset..."
Invoke-WebRequest -Uri $url -OutFile $out

Write-Host "plume: installed to $out"
if (($env:PATH -split ';') -notcontains $dest) {
  Write-Host "plume: add $dest to your PATH"
}
