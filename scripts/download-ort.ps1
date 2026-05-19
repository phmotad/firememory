# Downloads the ONNX Runtime DLL for Windows x64 into .\ort-lib\.
param([string]$OrtVersion = $env:ORT_VERSION)
if (-not $OrtVersion) { $OrtVersion = "1.25.0" }

$Base = "https://github.com/microsoft/onnxruntime/releases/download/v${OrtVersion}"
$File = "onnxruntime-win-x64-${OrtVersion}.zip"
$Url  = "${Base}/${File}"
$Tmp  = Join-Path $env:TEMP "ort-${OrtVersion}"
$Zip  = Join-Path $Tmp "ort.zip"

Write-Host "Downloading ${File}..."

New-Item -ItemType Directory -Force -Path $Tmp    | Out-Null
New-Item -ItemType Directory -Force -Path ort-lib | Out-Null

Invoke-WebRequest -Uri $Url -OutFile $Zip -UseBasicParsing
Expand-Archive -Path $Zip -DestinationPath $Tmp -Force

$DllSrc = Join-Path $Tmp "onnxruntime-win-x64-${OrtVersion}\lib\onnxruntime.dll"
Copy-Item $DllSrc -Destination "ort-lib\onnxruntime.dll" -Force

Remove-Item $Tmp -Recurse -Force
Write-Host "ORT ${OrtVersion} ready in ort-lib\"
