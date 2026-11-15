#Requires -Version 5.1
[CmdletBinding()]
param()

# 设置输出编码为 UTF-8
[Console]::OutputEncoding = [System.Text.Encoding]::UTF8
$OutputEncoding = [System.Text.Encoding]::UTF8

# 配置变量
$Script:FrontendDir = "web"
$Script:BackendDir = $PSScriptRoot
$Script:BackendOutput = "app.exe"

# 错误处理函数
function Exit-WithError {
    param([string]$Message)
    Write-Host "错误: $Message" -ForegroundColor Red
    Read-Host "按任意键退出"
    exit 1
}

# 检查命令是否存在
function Test-Command {
    param([string]$Command)
    try {
        $null = Get-Command $Command -ErrorAction Stop
        return $true
    } catch {
        return $false
    }
}

# 打印分隔线
function Write-Separator {
    param([string]$Title)
    Write-Host ""
    Write-Host "========================================" -ForegroundColor Cyan
    Write-Host "          $Title" -ForegroundColor Cyan
    Write-Host "========================================" -ForegroundColor Cyan
}

# 主函数
function Start-Build {
    Write-Separator "自动构建脚本"

    # 检查 Go 是否安装
    if (-not (Test-Command "go")) {
        Exit-WithError "Go 未安装"
    }
    Write-Host "✓ Go 已安装" -ForegroundColor Green

    # 检查 pnpm 是否安装
    if (-not (Test-Command "pnpm")) {
        Exit-WithError "pnpm 未安装"
    }
    Write-Host "✓ pnpm 已安装" -ForegroundColor Green

    # 构建前端
    Write-Separator "开始构建前端"

    $frontendPath = Join-Path $Script:BackendDir $Script:FrontendDir
    if (-not (Test-Path $frontendPath)) {
        Exit-WithError "前端目录不存在: $frontendPath"
    }

    Push-Location $frontendPath

    # 无条件安装：node_modules 存在不代表和 pnpm-lock.yaml 一致，改过依赖却没重装时
    # 会静默构建出旧包。--frozen-lockfile 在全命中缓存时只是秒级的幂等校验。
    Write-Host "正在校验并安装依赖..." -ForegroundColor Yellow
    pnpm install --frozen-lockfile
    if ($LASTEXITCODE -ne 0) {
        Pop-Location
        Exit-WithError "依赖安装失败"
    }

    # 构建前端
    Write-Host "正在构建前端..." -ForegroundColor Yellow
    pnpm run build
    if ($LASTEXITCODE -ne 0) {
        Pop-Location
        Exit-WithError "前端构建失败"
    }

    Write-Host "✅ 前端构建成功！" -ForegroundColor Green
    Pop-Location

    # 构建后端
    Write-Separator "开始构建后端"

    # 确保在后端目录
    Set-Location $Script:BackendDir

    Write-Host "正在整理依赖..." -ForegroundColor Yellow
    go mod tidy
    if ($LASTEXITCODE -ne 0) {
        Exit-WithError "Go 依赖整理失败"
    }

    Write-Host "正在构建 Go 项目..." -ForegroundColor Yellow
    $buildArgs = @(
        "-ldflags=-s -w"
        "-o", $Script:BackendOutput
        "."
    )
    & go build $buildArgs 2>&1
    if ($LASTEXITCODE -ne 0) {
        Exit-WithError "后端构建失败"
    }

    Write-Host "✅ 后端构建成功！" -ForegroundColor Green

    # 完成
    Write-Separator "构建完成"
}

# 执行构建
try {
    Start-Build
    Write-Host ""
    Write-Host "构建成功，3秒后自动退出..." -ForegroundColor Green
    for ($i = 3; $i -gt 0; $i--) {
        Write-Host "$i..." -ForegroundColor Yellow -NoNewline
        Start-Sleep -Seconds 1
    }
    Write-Host ""
    exit 0
} catch {
    Exit-WithError "构建过程中发生异常: $($_.Exception.Message)"
}
