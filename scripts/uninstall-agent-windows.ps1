# ServerEye Agent Uninstallation Script for Windows
# This script completely removes ServerEye agent from Windows

param(
    [switch]$Force
)

# Check if running as Administrator
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "❌ This script must be run as Administrator!" -ForegroundColor Red
    Write-Host "Please right-click PowerShell and select 'Run as Administrator'" -ForegroundColor Yellow
    exit 1
}

$serviceName = "ServerEyeAgent"
$installDir = "C:\Program Files\ServerEye"
$configDir = "C:\ProgramData\ServerEye"

Write-Host "🗑️  Uninstalling ServerEye Agent..." -ForegroundColor Red

# Confirm uninstallation
if (-not $Force) {
    $answer = Read-Host "Are you sure you want to uninstall ServerEye Agent? (y/N)"
    if ($answer -ne 'y' -and $answer -ne 'Y') {
        Write-Host "❌ Uninstallation cancelled" -ForegroundColor Yellow
        exit 0
    }
}

# Stop and remove service
Write-Host "🔧 Removing Windows Service..." -ForegroundColor Blue
try {
    if (Get-Service -Name $serviceName -ErrorAction SilentlyContinue) {
        Write-Host "Stopping service..." -ForegroundColor Yellow
        Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
        Start-Sleep -Seconds 2
        
        Write-Host "Deleting service..." -ForegroundColor Yellow
        & sc.exe delete $serviceName | Out-Null
        Write-Host "✅ Service removed" -ForegroundColor Green
    } else {
        Write-Host "⚠️  Service not found" -ForegroundColor Yellow
    }
} catch {
    Write-Host "⚠️  Failed to remove service: $_" -ForegroundColor Yellow
}

# Remove firewall rule
Write-Host "🔥 Removing firewall rule..." -ForegroundColor Blue
try {
    Remove-NetFirewallRule -DisplayName "ServerEye Agent" -ErrorAction SilentlyContinue
    Write-Host "✅ Firewall rule removed" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Failed to remove firewall rule: $_" -ForegroundColor Yellow
}

# Remove from PATH
Write-Host "📝 Removing from PATH..." -ForegroundColor Blue
try {
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
    if ($currentPath -like "*$installDir*") {
        $newPath = $currentPath.Replace(";$installDir", "").Replace("$installDir;", "").Replace($installDir, "")
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "Machine")
        Write-Host "✅ Removed from PATH" -ForegroundColor Green
    } else {
        Write-Host "⚠️  Not in PATH" -ForegroundColor Yellow
    }
} catch {
    Write-Host "⚠️  Failed to remove from PATH: $_" -ForegroundColor Yellow
}

# Remove installation directory
Write-Host "📁 Removing installation directory..." -ForegroundColor Blue
if (Test-Path $installDir) {
    try {
        Remove-Item $installDir -Recurse -Force
        Write-Host "✅ Installation directory removed" -ForegroundColor Green
    } catch {
        Write-Host "⚠️  Failed to remove installation directory: $_" -ForegroundColor Yellow
        Write-Host "   You may need to manually delete: $installDir" -ForegroundColor Yellow
    }
} else {
    Write-Host "⚠️  Installation directory not found" -ForegroundColor Yellow
}

# Remove configuration directory
Write-Host "📁 Removing configuration directory..." -ForegroundColor Blue
if (Test-Path $configDir) {
    try {
        Remove-Item $configDir -Recurse -Force
        Write-Host "✅ Configuration directory removed" -ForegroundColor Green
    } catch {
        Write-Host "⚠️  Failed to remove configuration directory: $_" -ForegroundColor Yellow
        Write-Host "   You may need to manually delete: $configDir" -ForegroundColor Yellow
    }
} else {
    Write-Host "⚠️  Configuration directory not found" -ForegroundColor Yellow
}

# Clean up scheduled tasks (if any)
Write-Host "📅 Cleaning up scheduled tasks..." -ForegroundColor Blue
try {
    Get-ScheduledTask -TaskName "ServerEye*" -ErrorAction SilentlyContinue | ForEach-Object {
        Unregister-ScheduledTask -TaskName $_.TaskName -Confirm:$false
        Write-Host "✅ Removed scheduled task: $($_.TaskName)" -ForegroundColor Green
    }
} catch {
    Write-Host "⚠️  No scheduled tasks found" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "✅ ServerEye Agent uninstalled successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Summary:" -ForegroundColor Cyan
Write-Host "   🗑️  Service: $serviceName" -ForegroundColor White
Write-Host "   📁 Install Dir: $installDir" -ForegroundColor White
Write-Host "   ⚙️  Config Dir: $configDir" -ForegroundColor White
Write-Host ""
Write-Host "🔄 Note: You may need to restart your computer for all changes to take effect." -ForegroundColor Yellow
Write-Host ""

# Ask for restart
$answer = Read-Host "Would you like to restart your computer now? (y/N)"
if ($answer -eq 'y' -or $answer -eq 'Y') {
    Write-Host "🔄 Restarting computer in 10 seconds..." -ForegroundColor Yellow
    Start-Sleep -Seconds 10
    Restart-Computer -Force
} else {
    Write-Host "✅ Uninstallation complete!" -ForegroundColor Green
}
