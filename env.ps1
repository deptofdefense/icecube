Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
$PSDefaultParameterValues['*:ErrorAction']='Stop'
function ThrowOnNativeFailure {
    if (-not $?)
    {
        throw 'Native Failure'
    }
}

$path_parts = $env:PATH.Split(";")

if (-Not ($path_parts -contains "$env:LOCALAPPDATA\Programs\Git\bin")) {
    Write-Host "Prepending git directory to PATH: $env:LOCALAPPDATA\Programs\Git\bin"
    $env:PATH = "{0};{1}" -f "$env:LOCALAPPDATA\Programs\Git\bin", $env:PATH
}

if (-Not ($path_parts -contains "$(Split-Path -parent $PSCommandPath)\bin")) {
    Write-Host "Prepending local bin directory to PATH: $(Split-Path -parent $PSCommandPath)\bin"
    $env:PATH = "{0};{1}" -f "$(Split-Path -parent $PSCommandPath)\bin", $env:PATH
}

if (-Not ($path_parts -contains "$(Split-Path -parent $PSCommandPath)\scripts")) {
    Write-Host "Prepending local scripts directory to PATH: $(Split-Path -parent $PSCommandPath)\scripts"
    $env:PATH = "{0};{1}" -f "$(Split-Path -parent $PSCommandPath)\scripts", $env:PATH
}

if ([bool](Test-Path -Path "$(Split-Path -parent $PSCommandpath)\env.local.ps1")) {
    Write-Host "Detected local modifications to environmental configuration: $dir\env.local.ps1"
    . "$(Split-Path -parent $PSCommandpath)\env.local.ps1"
}