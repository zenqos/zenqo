$ErrorActionPreference = "Stop"

$Repo    = "zenqos/zenqo"
$InstallDir = "$env:LOCALAPPDATA\zenqo"

# Fetch latest version
$Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
$Version = $Release.tag_name

$Filename = "zenqo_windows_amd64.zip"
$Url      = "https://github.com/$Repo/releases/download/$Version/$Filename"

Write-Host "Installing zenqo $Version (windows/amd64)..."

# Download and extract
$Tmp = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
try {
    Invoke-WebRequest -Uri $Url -OutFile "$Tmp\$Filename" -UseBasicParsing
    Expand-Archive -Path "$Tmp\$Filename" -DestinationPath $Tmp -Force

    # Install binary
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir | Out-Null
    }
    Move-Item -Force "$Tmp\zenqo.exe" "$InstallDir\zenqo.exe"
} finally {
    Remove-Item -Recurse -Force $Tmp -ErrorAction SilentlyContinue
}

# Add to PATH if not already there
$CurrentPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($CurrentPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$CurrentPath;$InstallDir", "User")
    $env:Path += ";$InstallDir"
    Write-Host "Added $InstallDir to PATH."
}

Write-Host ""
Write-Host "zenqo $Version installed to $InstallDir\zenqo.exe"
Write-Host ""
Write-Host "Get started:"
Write-Host "  zenqo new my-app"
Write-Host "  cd my-app && zenqo dev"
