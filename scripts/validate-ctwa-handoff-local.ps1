#Requires -Version 5.1
<#
Local CTWA 5287 / CRM V4 / painel gate.
Exclusivamente offline: sem VPS, sem Evolution, sem CRM e sem disparos reais.
Executar na raiz: powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1
Use -Format somente se GoFmt acusar formato: atualiza apenas os arquivos Go desta PR.
#>
param(
    [switch]$Format,
    [switch]$Race
)
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
if (-not $go) { throw "Go SDK nao encontrado. Nao instalar na VPS." }
$gofmt = Get-Command gofmt -ErrorAction SilentlyContinue
if (-not $gofmt) { throw "gofmt ausente do Go SDK." }

$files = @(
    "cmd/botwhatsapp/phone_config.go",
    "cmd/botwhatsapp/phone_config_test.go",
    "cmd/botwhatsapp/main.go",
    "internal/crm/native.go",
    "internal/crm/native_test.go",
    "internal/ingress/audit.go",
    "internal/ingress/audit_test.go",
    "internal/ingress/maintenance.go",
    "internal/ingress/maintenance_test.go",
    "internal/ingress/server.go",
    "internal/ingress/server_test.go",
    "internal/outbox/types.go",
    "internal/outbox/worker.go",
    "internal/dashboard/events.go",
    "internal/dashboard/events_test.go",
    "internal/dashboard/handler.go",
    "internal/dashboard/dashboard_test.go",
    "internal/dashboard/service.go",
    "internal/dashboard/ui.go"
)
Write-Host "=== Verificacao CTWA 5287 - offline ==="
& go version
if ($LASTEXITCODE -ne 0) { throw "go version falhou" }
if ($Format) {
    foreach ($file in $files) {
        if (Test-Path $file) { & gofmt -w $file; if ($LASTEXITCODE -ne 0) { throw "gofmt falhou: $file" } }
    }
}
$unformatted = @()
foreach ($file in $files) {
    if (-not (Test-Path $file)) { throw "Arquivo esperado nao encontrado: $file" }
    $bad = @(& gofmt -l $file)
    if ($LASTEXITCODE -ne 0) { throw "gofmt -l falhou: $file" }
    if ($bad.Count -gt 0) { $unformatted += $bad }
}
if ($unformatted.Count -gt 0) {
    throw ("Arquivos Go sem gofmt. Execute uma unica vez com -Format e revise o diff: " + ($unformatted -join ", "))
}

& git diff --check
if ($LASTEXITCODE -ne 0) { throw "git diff --check falhou" }
Write-Host "=== Testes focados (sem chamadas externas) ==="
& go test -count=1 ./internal/core ./internal/ingress ./internal/crm ./internal/outbox ./internal/dashboard ./cmd/botwhatsapp
if ($LASTEXITCODE -ne 0) { throw "Testes CTWA ou painel falharam" }

Write-Host "=== Vet de todos os modulos ==="
& go vet ./...
if ($LASTEXITCODE -ne 0) { throw "go vet falhou" }

$temp = Join-Path ([IO.Path]::GetTempPath()) ("botwhatsapp-gate-" + [Guid]::NewGuid().ToString("N"))
try {
    New-Item -ItemType Directory -Path $temp -Force | Out-Null
    & go build -o (Join-Path $temp "botwhatsapp.exe") ./cmd/botwhatsapp
    if ($LASTEXITCODE -ne 0) { throw "Build bot falhou" }
}
finally {
    if (Test-Path $temp) { Remove-Item -LiteralPath $temp -Recurse -Force }
}

if ($Race) {
    Write-Host "=== Race detector (opcional; Windows requer toolchain CGO) ==="
    & go test -race -count=1 ./internal/ingress ./internal/outbox ./internal/dashboard
    if ($LASTEXITCODE -ne 0) { throw "Race detector falhou ou ambiente CGO indisponivel" }
}
Write-Host "=== PASS OFFLINE. Nenhuma API real chamada. NAO significa deploy liberado. ==="
& git status --short
