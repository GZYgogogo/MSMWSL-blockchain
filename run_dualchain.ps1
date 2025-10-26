# 双链区块链系统运行脚本

Write-Host "========================================" -ForegroundColor Cyan
Write-Host "    双链区块链系统启动脚本" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

# 检查数据文件
if (-Not (Test-Path "data.xlsx")) {
    Write-Host "错误: 未找到 data.xlsx 文件" -ForegroundColor Red
    Write-Host "请确保 data.xlsx 在当前目录下" -ForegroundColor Yellow
    pause
    exit 1
}

# 检查配置文件
if (-Not (Test-Path "config/config.json")) {
    Write-Host "错误: 未找到 config/config.json 文件" -ForegroundColor Red
    pause
    exit 1
}

Write-Host "✓ 数据文件检查完成" -ForegroundColor Green
Write-Host "✓ 配置文件检查完成" -ForegroundColor Green
Write-Host ""

# 编译程序
Write-Host "正在编译双链系统..." -ForegroundColor Yellow
go build -o dualchain.exe cmd/dualchain/main.go

if ($LASTEXITCODE -ne 0) {
    Write-Host "编译失败！" -ForegroundColor Red
    pause
    exit 1
}

Write-Host "✓ 编译成功" -ForegroundColor Green
Write-Host ""

# 运行程序
Write-Host "启动双链系统..." -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

./dualchain.exe

Write-Host ""
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "程序运行完成" -ForegroundColor Green
Write-Host "详细日志已保存到: dualchain_log.txt" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Cyan

pause

