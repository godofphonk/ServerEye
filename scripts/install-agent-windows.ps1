# ServerEye Agent Installation Script for Windows
# This script installs and configures ServerEye agent with automatic key generation

param(
    [string]$BackendUrl = "https://api.servereye.dev",
    [string]$InstallDir = "C:\Program Files\ServerEye",
    [string]$ConfigDir = "C:\ProgramData\ServerEye",
    [switch]$Force
)

# Check if running as Administrator
if (-NOT ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "❌ This script must be run as Administrator!" -ForegroundColor Red
    Write-Host "Please right-click PowerShell and select 'Run as Administrator'" -ForegroundColor Yellow
    exit 1
}

Write-Host "🚀 Installing ServerEye Agent for Windows..." -ForegroundColor Green

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

# Download latest agent
Write-Host "⬇️  Downloading ServerEye Agent..." -ForegroundColor Blue
$AgentUrl = "https://github.com/godofphonk/ServerEye/releases/latest/download/servereye-agent-windows-amd64.exe"
$AgentPath = "$InstallDir\servereye-agent.exe"

try {
    Invoke-WebRequest -Uri $AgentUrl -OutFile $AgentPath -UseBasicParsing
    Write-Host "✅ Agent downloaded successfully" -ForegroundColor Green
} catch {
    Write-Host "❌ Failed to download agent: $_" -ForegroundColor Red
    exit 1
}

# Generate secret key
Write-Host "🔑 Generating secret key..." -ForegroundColor Blue
$bytes = [System.Byte[]]::new(16)
$random = [System.Security.Cryptography.RandomNumberGenerator]::Create()
$random.GetBytes($bytes)
$secretKey = "srv_" + [System.BitConverter]::ToString($bytes).Replace("-", "").ToLower()

# Get system information
$hostname = $env:COMPUTERNAME
$osInfo = (Get-CimInstance -ClassName Win32_OperatingSystem).Caption + " " + (Get-CimInstance -ClassName Win32_OperatingSystem).Version
$agentVersion = & $AgentPath --version 2>$null
if (-not $agentVersion) {
    $agentVersion = "1.1.0"
}

# Register key with backend API
Write-Host "🔄 Registering key with ServerEye API..." -ForegroundColor Blue
try {
    $registrationData = @{
        secret_key = $secretKey
        agent_version = $agentVersion
        os_info = $osInfo
        hostname = $hostname
    } | ConvertTo-Json

    $response = Invoke-RestMethod -Uri "$BackendUrl/api/v1/register-key" -Method Post -Body $registrationData -ContentType "application/json" -TimeoutSec 10
    Write-Host "✅ Key successfully registered with ServerEye API!" -ForegroundColor Green
} catch {
    Write-Host "⚠️  API registration failed: $_" -ForegroundColor Yellow
    Write-Host "Agent will work anyway, but you may need to register manually" -ForegroundColor Yellow
}

# Create configuration file
Write-Host "⚙️  Creating configuration..." -ForegroundColor Blue
$configContent = @"
# ServerEye Agent Configuration - Windows
server:
  name: "$hostname"
  description: "ServerEye monitored Windows machine"
  secret_key: "$secretKey"

api:
  base_url: "$BackendUrl"
  timeout: "30s"

websocket:
  enabled: true
  url: "wss://$($BackendUrl.Replace('https://', '').Replace('http://', ''))/ws"
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
  level: "info"
  file: "$ConfigDir\logs\agent.log"
"@

$configPath = "$ConfigDir\config.yaml"
$configContent | Out-File -FilePath $configPath -Encoding UTF8

# Create Windows Service
Write-Host "🔧 Installing Windows Service..." -ForegroundColor Blue
$serviceName = "ServerEyeAgent"
$serviceDisplayName = "ServerEye Monitoring Agent"

# Remove existing service if it exists
if (Get-Service -Name $serviceName -ErrorAction SilentlyContinue) {
    Write-Host "Removing existing service..." -ForegroundColor Yellow
    Stop-Service -Name $serviceName -Force -ErrorAction SilentlyContinue
    & sc.exe delete $serviceName | Out-Null
    Start-Sleep -Seconds 2
}

# Create new service
$serviceCommand = "`"$AgentPath`" --config `"$configPath`""
& sc.exe create $serviceName binPath= $serviceCommand start= auto DisplayName= $serviceDisplayName | Out-Null
& sc.exe description $serviceName "ServerEye monitoring agent for system metrics collection" | Out-Null

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
    New-NetFirewallRule -DisplayName "ServerEye Agent" -Direction Outbound -Program $AgentPath -Action Allow -Protocol TCP -LocalPort Any | Out-Null
    Write-Host "✅ Firewall rule created" -ForegroundColor Green
} catch {
    Write-Host "⚠️  Failed to create firewall rule: $_" -ForegroundColor Yellow
}

# Success message
Write-Host ""
Write-Host "🎉 ServerEye Agent installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Installation Summary:" -ForegroundColor Cyan
Write-Host "   📁 Install Directory: $InstallDir" -ForegroundColor White
Write-Host "   ⚙️  Config File: $configPath" -ForegroundColor White
Write-Host "   📝 Log File: $ConfigDir\logs\agent.log" -ForegroundColor White
Write-Host "   🔧 Service Name: $serviceName" -ForegroundColor White
Write-Host ""
Write-Host "🔑 Your Secret Key: $secretKey" -ForegroundColor Yellow
Write-Host ""
Write-Host "📱 To connect to Telegram Bot:" -ForegroundColor Cyan
Write-Host "   1. Find @ServereyeTG_bot in Telegram" -ForegroundColor White
Write-Host "   2. Send: /start" -ForegroundColor White
Write-Host "   3. Send: /add $secretKey" -ForegroundColor White
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

# Ask if user wants to view logs
$answer = Read-Host "Would you like to view the agent logs now? (y/N)"
if ($answer -eq 'y' -or $answer -eq 'Y') {
    Write-Host "📋 Recent logs:" -ForegroundColor Cyan
    if (Test-Path "$ConfigDir\logs\agent.log") {
        Get-Content "$ConfigDir\logs\agent.log" -Tail 20
    } else {
        Write-Host "Log file not found yet. Service may still be starting..." -ForegroundColor Yellow
    }
}

Write-Host "✅ Installation complete!" -ForegroundColor Green
