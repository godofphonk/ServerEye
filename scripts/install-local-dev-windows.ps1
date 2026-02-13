# ServerEye Agent Local Development Installation Script for Windows
# This script installs and configures ServerEye agent using local binary file (dev version)

param(
    [string]$AgentPath = ".\servereye-agent-windows.exe",
    [string]$InstallDir = "C:\Program Files\ServerEye",
    [string]$ConfigDir = "C:\ProgramData\ServerEye",
    [string]$SecretKey = "srv_dev_windows_test_key",
    [switch]$Force
)

# Check if running as Administrator
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "❌ This script must be run as Administrator!" -ForegroundColor Red
    Write-Host "Please right-click PowerShell and select 'Run as Administrator'" -ForegroundColor Yellow
    exit 1
}

Write-Host "🚀 Installing ServerEye Agent for Windows (Local Dev)..." -ForegroundColor Green

# Check if agent file exists
if (-NOT (Test-Path $AgentPath)) {
    Write-Host "❌ Agent file not found: $AgentPath" -ForegroundColor Red
    Write-Host "Please make sure servereye-agent-windows.exe is in the current directory" -ForegroundColor Yellow
    Write-Host "Or specify the path with -AgentPath parameter" -ForegroundColor Yellow
    exit 1
}

# Get absolute path to agent
$AgentPath = (Resolve-Path $AgentPath).Path
Write-Host "📁 Using agent from: $AgentPath" -ForegroundColor Blue

# Check if already installed
if ((Test-Path "$InstallDir\servereye-agent.exe") -and -not $Force) {
    Write-Host "⚠️  ServerEye Agent is already installed!" -ForegroundColor Yellow
    Write-Host "Use -Force to reinstall" -ForegroundColor Yellow
    exit 1
}

# Create directories
Write-Host "📁 Creating directories..." -ForegroundColor Blue
New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
New-Item -ItemType Directory -Path $ConfigDir -Force | Out-Null
New-Item -ItemType Directory -Path "$ConfigDir\logs" -Force | Out-Null

# Copy agent to installation directory
Write-Host "📋 Copying agent to installation directory..." -ForegroundColor Blue
try {
    Copy-Item $AgentPath "$InstallDir\servereye-agent.exe" -Force
    Write-Host "✅ Agent copied successfully" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to copy agent: $_" -ForegroundColor Red
    exit 1
}

# Get agent version
Write-Host "🏷️  Getting agent version..." -ForegroundColor Blue
try {
    $agentVersion = & "$InstallDir\servereye-agent.exe" --version 2>$null
    if (-not $agentVersion) {
        $agentVersion = "1.1.0-dev"
    }
    Write-Host "✅ Agent version: $agentVersion" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Could not get agent version, using default" -ForegroundColor Yellow
    $agentVersion = "1.1.0-dev"
}

# Generate secret key if not provided
if ($SecretKey -eq "srv_dev_windows_test_key") {
    Write-Host "🔑 Generating secret key..." -ForegroundColor Blue
    $bytes = [System.Byte[]]::new(16)
    $random = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $random.GetBytes($bytes)
    $SecretKey = "srv_" + [System.BitConverter]::ToString($bytes).Replace("-", "").ToLower()
} else {
    Write-Host "🔑 Using provided secret key: $SecretKey" -ForegroundColor Blue
}

# Get system information
$hostname = $env:COMPUTERNAME
$osInfo = (Get-CimInstance -ClassName Win32_OperatingSystem).Caption + " " + (Get-CimInstance -ClassName Win32_OperatingSystem).Version

# Skip API registration for dev version
Write-Host "⚠️  Skipping API registration (dev mode)" -ForegroundColor Yellow

# Create configuration file
Write-Host "⚙️  Creating configuration..." -ForegroundColor Blue
$configContent = @"
# ServerEye Agent Configuration - Windows (Development)
server:
  name: "$hostname-Dev"
  description: "ServerEye monitored Windows machine (Development)"
  secret_key: "$SecretKey"

api:
  base_url: "https://api.servereye.dev"
  timeout: "30s"

websocket:
  enabled: true
  url: "wss://servereye-registration-worker.servereye.workers.dev/ws"
  reconnect_interval: "5s"
  max_reconnect_attempts: 10
  ping_interval: "30s"
  write_timeout: "10s"
  read_timeout: "10s"

metrics:
  cpu_usage: false      # Disabled for now (Linux only)
  memory_usage: true    # Works via Go runtime
  disk_usage: false     # Disabled for now (Linux only)
  cpu_temperature: false # Disabled for now (Linux only)
  interval: "30s"

logging:
  level: "debug"
  file: "$ConfigDir\logs\agent.log"
"@

$configPath = "$ConfigDir\config.yaml"
$configContent | Out-File -FilePath $configPath -Encoding UTF8

# Create Windows Service
Write-Host "🔧 Installing Windows Service..." -ForegroundColor Blue
$serviceName = "ServerEyeAgentDev"
$serviceDisplayName = "ServerEye Monitoring Agent (Dev)"

# Remove existing service if it exists
if (Get-Service -Name $serviceName -ErrorAction SilentlyContinue) {
    Write-Host "Removing existing service..." -ForegroundColor Yellow
    Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
    & sc.exe delete $serviceName | Out-Null
    Start-Sleep -Seconds 2
}

# Create new service
$serviceCommand = "`"$InstallDir\servereye-agent.exe`" --config `"$configPath`""
& sc.exe create $serviceName binPath= $serviceCommand start= auto DisplayName= $serviceDisplayName | Out-Null
& sc.exe description $serviceName "ServerEye monitoring agent for system metrics collection (Development)" | Out-Null

# Set service to run as Local System (default)
& sc.exe config $serviceName obj= LocalSystem | Out-Null

# Start the service
Write-Host "🚀 Starting ServerEye service..." -ForegroundColor Blue
try {
    Start-Service -Name $serviceName -ErrorAction Stop
    Write-Host "✅ ServerEye Agent service started successfully!" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Service created but failed to start. You can start it manually:" -ForegroundColor Yellow
    Write-Host "   Start-Service -Name '$serviceName'" -ForegroundColor Yellow
}

# Add to PATH (optional)
$currentPath = [Environment]::GetEnvironmentVariable("PATH", "Machine")
if ($currentPath -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("PATH", $currentPath + ";$InstallDir", "Machine")
    Write-Host "✅ Added to system PATH" -ForegroundColor Green
}

# Create firewall rule
Write-Host "🔥 Creating firewall rule..." -ForegroundColor Blue
try {
    New-NetFirewallRule -DisplayName "ServerEye Agent (Dev)" -Direction Outbound -Program "$InstallDir\servereye-agent.exe" -Action Allow -Protocol TCP -LocalPort Any | Out-Null
    Write-Host "✅ Firewall rule created" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Failed to create firewall rule: $_" -ForegroundColor Yellow
}

# Success message
Write-Host ""
Write-Host "🎉 ServerEye Agent (Dev) installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Installation Summary:" -ForegroundColor Cyan
Write-Host "   📁 Install Directory: $InstallDir" -ForegroundColor White
Write-Host "   ⚙️  Config File: $configPath" -ForegroundColor White
Write-Host "   📝 Log File: $ConfigDir\logs\agent.log" -ForegroundColor White
Write-Host "   🔧 Service Name: $serviceName" -ForegroundColor White
Write-Host ""
Write-Host "🔑 Your Secret Key: $SecretKey" -ForegroundColor Yellow
Write-Host ""
Write-Host "📱 To connect to Telegram Bot:" -ForegroundColor Cyan
Write-Host "   1. Find @ServereyeTG_bot in Telegram" -ForegroundColor White
Write-Host "   2. Send: /start" -ForegroundColor White
Write-Host "   3. Send: /add $SecretKey" -ForegroundColor White
Write-Host ""
Write-Host "🔍 Check service status:" -ForegroundColor Cyan
Write-Host "   Get-Service '$serviceName'" -ForegroundColor White
Write-Host "   Get-Content '$ConfigDir\logs\agent.log' -Tail 20" -ForegroundColor White
Write-Host ""
Write-Host "🗑️  To uninstall:" -ForegroundColor Cyan
Write-Host "   Stop-Service '$serviceName' -Force" -ForegroundColor White
Write-Host "   & sc.exe delete '$serviceName'" -ForegroundColor White
Write-Host "   Remove-Item '$InstallDir' -Recurse -Force" -ForegroundColor White
Write-Host "   Remove-Item '$ConfigDir' -Recurse -Force" -ForegroundColor White
Write-Host ""
Write-Host "💡 Development mode:" -ForegroundColor Cyan
Write-Host "   - Debug logging enabled" -ForegroundColor White
Write-Host "   - API registration skipped" -ForegroundColor White
Write-Host "   - Service name: $serviceName" -ForegroundColor White
Write-Host ""

# Ask if user wants to view logs
$answer = Read-Host "Would you like to view the agent logs now? (y/N)"
if ($answer -eq 'y' -or $answer -eq 'Y') {
    Write-Host "📋 Recent logs:" -ForegroundColor Cyan
    if (Test-Path "$ConfigDir\logs\agent.log") {
        Get-Content "$ConfigDir\logs\agent.log" -Tail 20
    } else {
        Write-Host "Log file not found yet. Service may still be starting..." -ForegroundColor Yellow
        Write-Host "Check logs in a few moments with: Get-Content '$ConfigDir\logs\agent.log' -Tail 20" -ForegroundColor Yellow
    }
}

Write-Host "✅ Development installation complete!" -ForegroundColor Green
