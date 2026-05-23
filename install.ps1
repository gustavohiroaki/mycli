$ErrorActionPreference = "Stop"

$BinaryName = "mycli"
$ExecutableName = "$BinaryName.exe"
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\mycli"
$GoMinVersion = "1.24.1"

function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-ErrorAndExit {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
    exit 1
}

function ConvertTo-SemVer {
    param([string]$Version)

    $parts = $Version.Split(".")
    while ($parts.Count -lt 3) {
        $parts += "0"
    }

    return [version]($parts[0..2] -join ".")
}

function Test-VersionGreaterOrEqual {
    param(
        [string]$Current,
        [string]$Minimum
    )

    return (ConvertTo-SemVer $Current) -ge (ConvertTo-SemVer $Minimum)
}

function Test-Go {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        Write-ErrorAndExit "Go nao encontrado. Instale Go $GoMinVersion ou superior em https://go.dev/dl/ e execute este script novamente."
    }

    $goVersionOutput = & go version
    if ($goVersionOutput -notmatch "go([0-9]+(\.[0-9]+){1,2})") {
        Write-Warn "Nao foi possivel identificar a versao do Go. Saida: $goVersionOutput"
        return
    }

    $goVersion = $Matches[1]
    if (-not (Test-VersionGreaterOrEqual $goVersion $GoMinVersion)) {
        Write-Warn "Versao do Go ($goVersion) pode ser antiga. Recomendado: >= $GoMinVersion"
        Write-Warn "Considere instalar uma versao mais recente via https://go.dev/dl/"
    } else {
        Write-Info "Go $goVersion encontrado."
    }
}

function Build-Binary {
    Write-Info "Compilando $BinaryName..."
    Push-Location $PSScriptRoot
    try {
        & go mod download
        & go build -ldflags="-s -w" -o $ExecutableName .
    } finally {
        Pop-Location
    }
    Write-Info "Build concluido."
}

function Install-Binary {
    Write-Info "Instalando $ExecutableName em $InstallDir..."
    New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    Copy-Item -Path (Join-Path $PSScriptRoot $ExecutableName) -Destination (Join-Path $InstallDir $ExecutableName) -Force
    Write-Info "$BinaryName instalado com sucesso em $(Join-Path $InstallDir $ExecutableName)"
}

function Add-ToUserPath {
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    $pathEntries = @()

    if (-not [string]::IsNullOrWhiteSpace($userPath)) {
        $pathEntries = $userPath.Split(";") | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }
    }

    $normalizedInstallDir = $InstallDir.TrimEnd("\")
    $alreadyInPath = $pathEntries | Where-Object { $_.TrimEnd("\") -ieq $normalizedInstallDir }

    if ($alreadyInPath) {
        Write-Info "$InstallDir ja esta no PATH do usuario."
        return
    }

    $newPath = if ([string]::IsNullOrWhiteSpace($userPath)) {
        $InstallDir
    } else {
        "$userPath;$InstallDir"
    }

    [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
    $env:Path = "$env:Path;$InstallDir"
    Write-Info "$InstallDir adicionado ao PATH do usuario."
    Write-Warn "Abra um novo terminal para usar '$BinaryName' em qualquer diretorio."
}

function Clear-BuildArtifact {
    $artifact = Join-Path $PSScriptRoot $ExecutableName
    if (Test-Path $artifact) {
        Remove-Item $artifact -Force
    }
}

function Test-Install {
    $installedBinary = Join-Path $InstallDir $ExecutableName
    if (-not (Test-Path $installedBinary)) {
        Write-ErrorAndExit "Instalacao falhou: $installedBinary nao encontrado."
    }

    Write-Info "Verificacao OK: $installedBinary"
    & $installedBinary --help | Select-Object -First 5
}

function Main {
    Write-Host "========================================="
    Write-Host "  Instalador do $BinaryName para Windows"
    Write-Host "========================================="

    Test-Go
    Build-Binary
    Install-Binary
    Add-ToUserPath
    Clear-BuildArtifact
    Test-Install

    Write-Host ""
    Write-Info "Instalacao concluida! Execute '$BinaryName --help' para comecar."
}

Main
