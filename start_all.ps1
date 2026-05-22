# start_all.ps1 — VNU-LEO Full Project Launcher (Single Window)
# Chạy toàn bộ project trong MỘT cửa sổ PowerShell duy nhất.
# Mỗi module chạy như một background job, log được stream ra với màu riêng.
#
# Cách dùng:
#   .\start_all.ps1            # Start tất cả
#   .\start_all.ps1 -NoSim     # Start không có traffic simulator
#
# Để dừng: nhấn Ctrl+C

param(
    [switch]$NoSim   # Bỏ qua traffic simulator nếu chỉ muốn dev
)

$ROOT = $PSScriptRoot

# ── Màu sắc cho từng module ────────────────────────────────────────────────────
$Colors = @{
    Core   = 'Green'
    Web    = 'Cyan'
    Client = 'Magenta'
    Sim    = 'Yellow'
    System = 'DarkGray'
}

# ── Prefix timestamp + label cho mỗi dòng log ─────────────────────────────────
function Write-Log {
    param([string]$Label, [string]$Color, [string]$Message)
    $ts = (Get-Date).ToString("HH:mm:ss")
    Write-Host "[$ts] " -NoNewline -ForegroundColor DarkGray
    Write-Host "$($Label.PadRight(7))" -NoNewline -ForegroundColor $Color
    Write-Host " $Message"
}

# ── Banner ─────────────────────────────────────────────────────────────────────
Clear-Host
Write-Host ""
Write-Host "  ╔══════════════════════════════════════════════╗" -ForegroundColor Cyan
Write-Host "  ║        VNU-LEO Satellite ISP Platform        ║" -ForegroundColor Cyan
Write-Host "  ║              Full Project Launcher           ║" -ForegroundColor Cyan
Write-Host "  ╚══════════════════════════════════════════════╝" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Core Network  →  http://localhost:8081" -ForegroundColor DarkGray
Write-Host "  Web Admin     →  http://localhost:5173" -ForegroundColor DarkGray
Write-Host "  Client App    →  http://localhost:5174" -ForegroundColor DarkGray
Write-Host ""
Write-Host "  Press Ctrl+C to stop all modules." -ForegroundColor DarkGray
Write-Host "  " + ("─" * 46) -ForegroundColor DarkGray
Write-Host ""

# ── Khởi tạo background jobs ───────────────────────────────────────────────────
$jobs = @()

# 1. Core Network (Go)
Write-Log "SYSTEM" $Colors.System "Starting Core Network (Go)..."
$jobs += Start-Job -Name "CoreNetwork" -ScriptBlock {
    param($dir)
    Set-Location $dir
    go run .
} -ArgumentList (Join-Path $ROOT "core_network")

# 2. Web Admin (SvelteKit)
Write-Log "SYSTEM" $Colors.System "Starting Web Admin (SvelteKit)..."
$jobs += Start-Job -Name "WebAdmin" -ScriptBlock {
    param($dir)
    Set-Location $dir
    npm run dev -- --port 5173 2>&1
} -ArgumentList (Join-Path $ROOT "web_admin")

# 3. Client App (Vite)
Write-Log "SYSTEM" $Colors.System "Starting Client App (Vite)..."
$jobs += Start-Job -Name "ClientApp" -ScriptBlock {
    param($dir)
    Set-Location $dir
    # Cài node_modules nếu chưa có
    if (-not (Test-Path "node_modules")) {
        npm install 2>&1
    }
    npm run dev -- --port 5174 2>&1
} -ArgumentList (Join-Path $ROOT "client_app")

# 4. Traffic Simulator (Node.js) — đợi Core Network sẵn sàng
if (-not $NoSim) {
    Write-Log "SYSTEM" $Colors.System "Scheduling Traffic Simulator (starts in 10s)..."
    $simPath = Join-Path $ROOT "simulate_traffic.js"
    $jobs += Start-Job -Name "Simulator" -ScriptBlock {
        param($script, $dir)
        Set-Location $dir
        Start-Sleep -Seconds 10
        node $script 2>&1
    } -ArgumentList $simPath, $ROOT
}

Write-Log "SYSTEM" $Colors.System "All jobs started. Streaming logs below..."
Write-Host ""

# ── Map job name → màu + label ────────────────────────────────────────────────
$JobMeta = @{
    "CoreNetwork" = @{ Label = "CORE   "; Color = $Colors.Core   }
    "WebAdmin"    = @{ Label = "WEB    "; Color = $Colors.Web    }
    "ClientApp"   = @{ Label = "CLIENT "; Color = $Colors.Client }
    "Simulator"   = @{ Label = "SIM    "; Color = $Colors.Sim    }
}

# ── Log streaming loop ─────────────────────────────────────────────────────────
# Tiếp tục chạy đến khi người dùng nhấn Ctrl+C
try {
    while ($true) {
        foreach ($job in $jobs) {
            $meta = $JobMeta[$job.Name]
            if (-not $meta) { continue }

            # Lấy output mới từ job (non-blocking)
            $output = $job | Receive-Job -ErrorAction SilentlyContinue
            foreach ($line in $output) {
                if ($line -and $line.ToString().Trim() -ne "") {
                    $ts = (Get-Date).ToString("HH:mm:ss")
                    Write-Host "[$ts] " -NoNewline -ForegroundColor DarkGray
                    Write-Host $meta.Label -NoNewline -ForegroundColor $meta.Color
                    Write-Host " $line"
                }
            }

            # Kiểm tra job bị crash
            if ($job.State -eq "Failed" -or $job.State -eq "Stopped") {
                Write-Log $meta.Label.Trim() "Red" "PROCESS EXITED (State: $($job.State))"
            }
        }
        Start-Sleep -Milliseconds 300
    }
}
finally {
    # ── Cleanup khi Ctrl+C ────────────────────────────────────────────────────
    Write-Host ""
    Write-Log "SYSTEM" $Colors.System "Stopping all modules..."
    foreach ($job in $jobs) {
        Stop-Job  -Job $job -ErrorAction SilentlyContinue
        Remove-Job -Job $job -ErrorAction SilentlyContinue
        Write-Log "SYSTEM" $Colors.System "Stopped: $($job.Name)"
    }
    Write-Host ""
    Write-Log "SYSTEM" $Colors.System "All modules stopped. Goodbye!"
    Write-Host ""
}
