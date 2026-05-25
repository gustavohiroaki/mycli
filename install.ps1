$ErrorActionPreference = "Stop"

$BinaryName = "mycli"
$ExecutableName = "$BinaryName.exe"
$InstallDir = Join-Path $env:LOCALAPPDATA "Programs\mycli"
$GoMinVersion = "1.25.0"
$GoCommand = "go"

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
    $found = Find-Go
    if (-not $found) {
        Write-Warn "Go nao encontrado. Instalando Go via winget..."
        Install-Go
        $found = Find-Go
    }

    if (-not $found) {
        Write-ErrorAndExit "Go nao encontrado apos instalacao."
    }

    $script:GoCommand = $found
    $goVersionOutput = & $script:GoCommand version
    if ($goVersionOutput -notmatch "go([0-9]+(\.[0-9]+){1,2})") {
        Write-Warn "Nao foi possivel identificar a versao do Go. Saida: $goVersionOutput"
        return
    }

    $goVersion = $Matches[1]
    if (-not (Test-VersionGreaterOrEqual $goVersion $GoMinVersion)) {
        Write-Warn "Versao do Go ($goVersion) antiga. Instalando/atualizando Go via winget..."
        Install-Go
        $found = Find-Go
        if (-not $found) {
            Write-ErrorAndExit "Go nao encontrado apos atualizacao."
        }
        $script:GoCommand = $found
        $goVersionOutput = & $script:GoCommand version
        if ($goVersionOutput -notmatch "go([0-9]+(\.[0-9]+){1,2})") {
            Write-ErrorAndExit "Nao foi possivel identificar a versao do Go apos atualizacao. Saida: $goVersionOutput"
        }
        $goVersion = $Matches[1]
        if (-not (Test-VersionGreaterOrEqual $goVersion $GoMinVersion)) {
            Write-ErrorAndExit "Go $goVersion instalado, mas esperado >= $GoMinVersion."
        }
    } else {
        Write-Info "Go $goVersion encontrado."
    }
}

function Find-Go {
    $existing = Get-Command go -ErrorAction SilentlyContinue
    if ($existing) {
        return $existing.Source
    }

    $programFilesGo = Join-Path $env:ProgramFiles "Go\bin\go.exe"
    if (Test-Path $programFilesGo) {
        return $programFilesGo
    }

    $localGo = Join-Path $env:LOCALAPPDATA "Programs\Go\bin\go.exe"
    if (Test-Path $localGo) {
        return $localGo
    }

    return $null
}

function Install-Go {
    $winget = Get-Command winget -ErrorAction SilentlyContinue
    if (-not $winget) {
        Write-ErrorAndExit "winget nao esta disponivel. Instale Go $GoMinVersion ou superior em https://go.dev/dl/."
    }

    & winget install --id GoLang.Go --exact --silent --accept-package-agreements --accept-source-agreements
    $goBin = Join-Path $env:ProgramFiles "Go\bin"
    if (Test-Path $goBin) {
        $env:Path = "$env:Path;$goBin"
    }
}

function Install-ExifTool {
    $existing = Get-Command exiftool -ErrorAction SilentlyContinue
    if ($existing) {
        $version = & exiftool -ver
        Write-Info "ExifTool encontrado: $version"
        return
    }

    $winget = Get-Command winget -ErrorAction SilentlyContinue
    if (-not $winget) {
        Write-ErrorAndExit "ExifTool nao encontrado e winget nao esta disponivel. Instale ExifTool manualmente ou instale App Installer/winget."
    }

    Write-Info "ExifTool nao encontrado. Instalando via winget..."
    & winget install --id OliverBetz.ExifTool --exact --silent --accept-package-agreements --accept-source-agreements

    $installed = Get-Command exiftool -ErrorAction SilentlyContinue
    if (-not $installed) {
        Write-ErrorAndExit "Instalacao do ExifTool falhou ou exiftool nao entrou no PATH. Abra um novo terminal e verifique."
    }

    $version = & exiftool -ver
    Write-Info "ExifTool instalado: $version"
}

function Install-FFmpeg {
    $ffmpeg = Get-Command ffmpeg -ErrorAction SilentlyContinue
    $ffprobe = Get-Command ffprobe -ErrorAction SilentlyContinue
    if ($ffmpeg -and $ffprobe) {
        $version = & ffmpeg -version | Select-Object -First 1
        Write-Info "FFmpeg encontrado: $version"
        return
    }

    $winget = Get-Command winget -ErrorAction SilentlyContinue
    if (-not $winget) {
        Write-ErrorAndExit "FFmpeg nao encontrado e winget nao esta disponivel. Instale FFmpeg manualmente."
    }

    Write-Info "FFmpeg nao encontrado. Instalando via winget..."
    & winget install --id Gyan.FFmpeg --exact --silent --accept-package-agreements --accept-source-agreements

    $installed = Get-Command ffmpeg -ErrorAction SilentlyContinue
    if (-not $installed) {
        Write-Warn "FFmpeg instalado, mas pode exigir novo terminal para entrar no PATH."
        return
    }
    $version = & ffmpeg -version | Select-Object -First 1
    Write-Info "FFmpeg instalado: $version"
}

function Build-Binary {
    Write-Info "Compilando $BinaryName..."
    Push-Location $PSScriptRoot
    try {
        & $script:GoCommand mod download
        & $script:GoCommand build -ldflags="-s -w" -o $ExecutableName .
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
    Install-ExifTool
    Install-FFmpeg
    Build-Binary
    Install-Binary
    Add-ToUserPath
    Clear-BuildArtifact
    Test-Install

    Write-Host ""
    Write-Info "Instalacao concluida! Execute '$BinaryName --help' para comecar."
}

Main
