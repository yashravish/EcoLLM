# Deploy EcoLLM to Railway (API + Web + Postgres + Redis)
# Requires: Railway CLI on PATH (or ~/.railway/bin) and authentication.
#
# Auth options (pick one):
#   1. Run `railway login` in an interactive terminal
#   2. Set $env:RAILWAY_TOKEN to a token from https://railway.com/account/tokens
#
# Usage:
#   pwsh -File scripts/deploy-railway.ps1
#   pwsh -File scripts/deploy-railway.ps1 -SkipMigrations

param(
    [string]$ProjectName = "ecollm",
    [switch]$SkipMigrations
)

$ErrorActionPreference = "Stop"

$Root = Split-Path -Parent $PSScriptRoot
$RailwayBin = Join-Path $env:USERPROFILE ".railway\bin\railway.exe"
if (-not (Get-Command railway -ErrorAction SilentlyContinue)) {
    if (Test-Path $RailwayBin) {
        $env:Path = "$(Split-Path $RailwayBin -Parent);$env:Path"
    } else {
        throw "Railway CLI not found. Install from https://docs.railway.com/cli"
    }
}

function Invoke-Railway {
    param([Parameter(ValueFromRemainingArguments = $true)][string[]]$Args)
    & railway @Args
    if ($LASTEXITCODE -ne 0) {
        throw "railway $($Args -join ' ') failed with exit code $LASTEXITCODE"
    }
}

function Read-DotEnv {
    param([string]$Path)
    $vars = @{}
    if (-not (Test-Path $Path)) {
        throw "Missing $Path"
    }
    Get-Content $Path | ForEach-Object {
        if ($_ -match '^\s*([A-Z_][A-Z0-9_]*)\s*=\s*(.*)\s*$') {
            $vars[$matches[1]] = $matches[2]
        }
    }
    return $vars
}

function Set-RailwayVar {
    param(
        [string]$Service,
        [string]$Key,
        [string]$Value,
        [switch]$SkipDeploy
    )
    $args = @("variable", "set", "${Key}=${Value}", "--service", $Service)
    if ($SkipDeploy) { $args += "--skip-deploys" }
    Invoke-Railway @args
}

Write-Host "Checking Railway authentication..."
try {
    Invoke-Railway @("whoami")
} catch {
    throw @"
Not logged in to Railway.

Run ONE of the following, then re-run this script:
  railway login
  `$env:RAILWAY_TOKEN = '<token-from-railway.com/account/tokens>'
"@
}

Set-Location $Root
$envFile = Join-Path $Root ".env"
$envVars = Read-DotEnv -Path $envFile

Write-Host "Initializing Railway project '$ProjectName' (if not already linked)..."
try {
    Invoke-Railway @("status", "--json") | Out-Null
} catch {
    Invoke-Railway @("init", "--name", $ProjectName)
}

Write-Host "Adding Postgres and Redis (ignored if they already exist)..."
try { Invoke-Railway @("add", "--database", "postgres", "--json") } catch { Write-Host "  Postgres may already exist" }
try { Invoke-Railway @("add", "--database", "redis", "--json") } catch { Write-Host "  Redis may already exist" }

Write-Host "Adding application services..."
try { Invoke-Railway @("add", "--service", "ecollm-api", "--json") } catch { Write-Host "  ecollm-api may already exist" }
try { Invoke-Railway @("add", "--service", "ecollm-web", "--json") } catch { Write-Host "  ecollm-web may already exist" }

Write-Host "Configuring ecollm-api variables..."
$apiVars = @{
    "DATABASE_URL"                   = '${{Postgres.DATABASE_URL}}'
    "REDIS_URL"                      = '${{Redis.REDIS_URL}}'
    "JWT_SECRET"                     = $envVars["JWT_SECRET"]
    "INFERENCE_PHI3_URL"             = $envVars["INFERENCE_PHI3_URL"]
    "INFERENCE_MISTRAL_URL"          = $envVars["INFERENCE_MISTRAL_URL"]
    "INFERENCE_LLAMA13B_URL"         = $envVars["INFERENCE_LLAMA13B_URL"]
    "INFERENCE_LLAMA70B_URL"         = $envVars["INFERENCE_LLAMA70B_URL"]
    "INFERENCE_PHI3_MODEL"           = $envVars["INFERENCE_PHI3_MODEL"]
    "INFERENCE_MISTRAL_MODEL"        = $envVars["INFERENCE_MISTRAL_MODEL"]
    "INFERENCE_LLAMA13B_MODEL"       = $envVars["INFERENCE_LLAMA13B_MODEL"]
    "INFERENCE_LLAMA70B_MODEL"       = $envVars["INFERENCE_LLAMA70B_MODEL"]
    "INFERENCE_API_KEY"              = $envVars["INFERENCE_API_KEY"]
    "ENABLE_PROMPT_OPTIMIZATION"     = "false"
    "ENABLE_CARBON_TRACKING"         = $envVars["ENABLE_CARBON_TRACKING"]
    "ENABLE_CACHE"                   = $envVars["ENABLE_CACHE"]
    "ENABLE_FALLBACK"                = $envVars["ENABLE_FALLBACK"]
    "GRID_REGION"                    = $envVars["GRID_REGION"]
    "GRID_API_KEY"                   = $envVars["GRID_API_KEY"]
    "LOG_LEVEL"                      = $envVars["LOG_LEVEL"]
    "REQUEST_TIMEOUT"                = $envVars["REQUEST_TIMEOUT"]
    "GITHUB_CLIENT_ID"               = $envVars["GITHUB_CLIENT_ID"]
    "GITHUB_CLIENT_SECRET"           = $envVars["GITHUB_CLIENT_SECRET"]
    "GOOGLE_CLIENT_ID"               = $envVars["GOOGLE_CLIENT_ID"]
    "GOOGLE_CLIENT_SECRET"           = $envVars["GOOGLE_CLIENT_SECRET"]
    "ALLOWED_ORIGINS"                = 'https://${{ ecollm-web.RAILWAY_PUBLIC_DOMAIN }}'
    "FRONTEND_URL"                   = 'https://${{ ecollm-web.RAILWAY_PUBLIC_DOMAIN }}'
    "API_BASE_URL"                   = 'https://${{ ecollm-api.RAILWAY_PUBLIC_DOMAIN }}'
    "RAILWAY_DOCKERFILE_PATH"        = "Dockerfile"
}

foreach ($entry in $apiVars.GetEnumerator()) {
    if ([string]::IsNullOrWhiteSpace($entry.Value)) { continue }
    Set-RailwayVar -Service "ecollm-api" -Key $entry.Key -Value $entry.Value -SkipDeploy
}

Write-Host "Deploying ecollm-api..."
Invoke-Railway @("up", "./apps/api", "--path-as-root", "--service", "ecollm-api", "--detach", "--ci")

Write-Host "Generating public domain for ecollm-api..."
try {
    Invoke-Railway @("domain", "--service", "ecollm-api", "--json")
} catch {
    Write-Host "  Domain may already exist"
}

if (-not $SkipMigrations) {
    Write-Host "Running database migrations..."
    $dbUrl = (Invoke-Railway @("variable", "list", "--service", "ecollm-api", "--kv") | Select-String "^DATABASE_URL=" | ForEach-Object { $_.Line.Substring(12) })
    if (-not $dbUrl) { throw "Could not read DATABASE_URL from Railway" }
    docker run --rm `
        -v "${Root}/db/migrations:/migrations" `
        migrate/migrate:v4.17.0 `
        -path /migrations `
        -database $dbUrl `
        up
}

Write-Host "Configuring ecollm-web variables..."
Set-RailwayVar -Service "ecollm-web" -Key "NEXT_PUBLIC_API_URL" -Value 'https://${{ ecollm-api.RAILWAY_PUBLIC_DOMAIN }}' -SkipDeploy
Set-RailwayVar -Service "ecollm-web" -Key "PORT" -Value "3000" -SkipDeploy
Set-RailwayVar -Service "ecollm-web" -Key "RAILWAY_DOCKERFILE_PATH" -Value "Dockerfile" -SkipDeploy

Write-Host "Deploying ecollm-web..."
Invoke-Railway @("up", "./apps/web", "--path-as-root", "--service", "ecollm-web", "--detach", "--ci")

Write-Host "Generating public domain for ecollm-web..."
try {
    Invoke-Railway @("domain", "--service", "ecollm-web", "--json")
} catch {
    Write-Host "  Domain may already exist"
}

Write-Host "Redeploying ecollm-api with updated CORS/domain references..."
Invoke-Railway @("redeploy", "--service", "ecollm-api", "--yes")

Write-Host ""
Write-Host "Deployment complete."
Write-Host "Check URLs with: railway domain --service ecollm-web"
Write-Host ""
Write-Host "Update OAuth callback URLs to:"
Write-Host "  GitHub: https://<api-domain>/auth/github/callback"
Write-Host "  Google: https://<api-domain>/auth/google/callback"
