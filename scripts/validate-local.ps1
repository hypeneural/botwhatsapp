#Requires -Version 5.1
# Auditoria local SEM conexão com VPS, WhatsApp ou CRM.
# Antigravity: executar UMA VEZ após atualizar a branch e conferir git status.
$ErrorActionPreference = "Stop"
Set-Location (Resolve-Path (Join-Path $PSScriptRoot ".."))
$go = Get-Command go -ErrorAction SilentlyContinue
if (-not $go) {
    $candidate = Join-Path $env:USERPROFILE "go-sdk\bin\go.exe"
    if (Test-Path $candidate) {
        $env:PATH = (Split-Path $candidate) + [IO.Path]::PathSeparator + $env:PATH
        $go = Get-Command go -ErrorAction SilentlyContinue
    }
}
if (-not $go) { throw "Go nao encontrado. Usar SDK local, nunca instalar na VPS para teste." }
Write-Host "=== Go local ==="
& go version
if ($LASTEXITCODE -ne 0) { throw "go version falhou" }
Write-Host "=== Revisao de codigo (go vet) ==="
& go vet ./...
if ($LASTEXITCODE -ne 0) { throw "go vet falhou" }
Write-Host "=== Testes offline de TODOS os modulos (uma vez) ==="
& go test -count=1 ./...
if ($LASTEXITCODE -ne 0) { throw "go test falhou" }
Write-Host "=== Build dos dois binarios sem artefatos locais ==="
& go build ./cmd/botwhatsapp ./cmd/webhook-relay
if ($LASTEXITCODE -ne 0) { throw "go build falhou" }
$docker = Get-Command docker -ErrorAction SilentlyContinue
if ($docker) {
    Write-Host "=== Docker Compose APENAS config (sem up/down/pull) ==="
    & docker compose -f deploy/compose.staging.yaml config --quiet
    if ($LASTEXITCODE -ne 0) { throw "docker compose config falhou" }
} else {
    Write-Host "SKIP Docker Compose: Docker nao instalado neste host; validar apenas no ambiente autorizado depois."
}
Write-Host "=== PASS na validacao LOCAL; NAO implica liberacao de producao. ==="
