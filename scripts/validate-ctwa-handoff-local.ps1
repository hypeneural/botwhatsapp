#Requires -Version 5.1
<#
Gate P0 offline, read-only regarding VPS/CRM/WhatsApp.
USO MAIS SIMPLES (normaliza apenas Go da PR e executa os gates uma vez):
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1 -Format

Abrir painel visual com DADOS FICTÍCIOS após PASS:
powershell -NoProfile -ExecutionPolicy Bypass -File .\scripts\validate-ctwa-handoff-local.ps1 -Format -Preview

Sem -Format: apenas verifica, sem tocar no codigo.
-Race: opcional, requer CGO/C compiler.
#>
param([switch]$Format,[switch]$Preview,[switch]$Race)
$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
Set-Location (Resolve-Path (Join-Path $PSScriptRoot ".."))
$started = Get-Date
$check = "NOT_RUN"
$results = New-Object System.Collections.Generic.List[string]
$report = Join-Path ([IO.Path]::GetTempPath()) ("evydencia-ctwa-gate-" + $started.ToString("yyyyMMdd-HHmmss") + ".txt")

function Run-Gate([string]$name,[scriptblock]$action) {
    Write-Host ""
    Write-Host ("== " + $name + " ==") -ForegroundColor Cyan
    & $action
    if ($LASTEXITCODE -ne 0) {
        $script:results.Add($name + ": FAIL (" + $LASTEXITCODE + ")")
        throw ($name + " falhou com exit code " + $LASTEXITCODE)
    }
    $script:results.Add($name + ": PASS")
}

try {
    if (-not (Test-Path ".git")) {
        # Worktrees have .git as a file, which Test-Path still recognizes.
        throw "Execute dentro da worktree do botwhatsapp; pasta .git ausente."
    }
    $branch = (& git branch --show-current).Trim()
    if ($LASTEXITCODE -ne 0) { throw "Nao foi possivel ler a branch" }
    $sha = (& git rev-parse HEAD).Trim()
    if ($LASTEXITCODE -ne 0) { throw "Nao foi possivel ler o SHA" }
    Write-Host ("Repositorio: " + $branch + " @ " + $sha)
    if ($branch -ne "fix/ctwa-native-proposal-handoff-20261027") {
        throw "Worktree em branch inesperada. Nao validar outra branch por engano."
    }
    $go = Get-Command go -ErrorAction SilentlyContinue
    if (-not $go) {
        $candidate = Join-Path $env:USERPROFILE "go-sdk\bin\go.exe"
        if (Test-Path $candidate) {
            $env:PATH = (Split-Path $candidate) + [IO.Path]::PathSeparator + $env:PATH
            $go = Get-Command go -ErrorAction SilentlyContinue
        }
    }
    if (-not $go) { throw "Go SDK nao encontrado: validar no Windows local, nao instalar na VPS." }
    if (-not (Get-Command gofmt -ErrorAction SilentlyContinue)) {
        throw "gofmt nao esta disponivel no PATH."
    }

    # Whitelist: no formatting outside of this P0 PR.
    $files = @(
        "cmd/botwhatsapp/phone_config.go",
        "cmd/botwhatsapp/phone_config_test.go",
        "cmd/botwhatsapp/main.go",
        "cmd/dashboard-preview/main.go",
        "cmd/dashboard-preview/main_test.go",
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
    foreach ($path in $files) {
        if (-not (Test-Path $path -PathType Leaf)) { throw "Arquivo requerido ausente: $path" }
    }

    Run-Gate "Go version" { & go version }
    $formatProblems = @()
    if ($Format) {
        Write-Host "Formatando arquivos Go desta PR (somente gofmt, sem alteracao semantica)..."
        & gofmt -w $files
        if ($LASTEXITCODE -ne 0) { throw "gofmt -w falhou" }
    }
    $formatProblems = @(& gofmt -l $files)
    if ($LASTEXITCODE -ne 0) { throw "gofmt -l falhou" }
    if ($formatProblems.Count -gt 0) {
        throw ("gofmt pendente nos arquivos: " + ($formatProblems -join ", ") +
            ". Execute com -Format em worktree isolada e confira git diff.")
    }
    $results.Add("gofmt: PASS")

    # Tests never use production secrets; no HTTP external test invocations.
    Run-Gate "Offline go test (uma execucao)" { & go test -count=1 ./... }
    Run-Gate "Go vet" { & go vet ./... }
    Run-Gate "Git whitespace" { & git diff --check }

    $buildDir = Join-Path ([IO.Path]::GetTempPath()) ("evydencia-build-" + [Guid]::NewGuid().ToString("N"))
    try {
        New-Item -ItemType Directory -Path $buildDir -Force | Out-Null
        Run-Gate "Build bot" { & go build -o (Join-Path $buildDir "bot.exe") ./cmd/botwhatsapp }
        Run-Gate "Build preview local" { & go build -o (Join-Path $buildDir "preview.exe") ./cmd/dashboard-preview }
    } finally {
        if (Test-Path $buildDir) { Remove-Item -LiteralPath $buildDir -Recurse -Force }
    }

    # Validate the JavaScript parsed from the actual Go raw HTML literal.
    $node = Get-Command node -ErrorAction SilentlyContinue
    if ($node) {
        $uiSource = [IO.File]::ReadAllText((Resolve-Path "internal/dashboard/ui.go"))
        $match = [regex]::Match($uiSource, '(?s)<script>(.*?)</script>')
        if (-not $match.Success) { throw "Nao foi encontrado o script da interface." }
        $jsFile = Join-Path ([IO.Path]::GetTempPath()) ("evydencia-ui-" + [Guid]::NewGuid().ToString("N") + ".js")
        try {
            [IO.File]::WriteAllText($jsFile, $match.Groups[1].Value, (New-Object System.Text.UTF8Encoding($false)))
            Run-Gate "Sintaxe JavaScript do painel" { & node --check $jsFile }
        } finally {
            if (Test-Path $jsFile) { Remove-Item -LiteralPath $jsFile -Force }
        }
    } else {
        $results.Add("JS syntax: SKIP (Node nao encontrado; teste Go do markup ainda executado)")
        Write-Host "Node ausente. Verificacao JS ignorada; nao reportar como PASS." -ForegroundColor Yellow
    }

    if ($Race) {
        Run-Gate "Race detector opcional" { & go test -race -count=1 ./internal/ingress ./internal/outbox ./internal/dashboard }
    }
    $check="PASS_OFFLINE"
    Write-Host ""
    Write-Host "VALIDACAO OFFLINE CONCLUIDA. Nenhum POST comercial, deploy ou acesso VPS realizado." -ForegroundColor Green
    Write-Host "Arquivos modificados no worktree:"
    & git status --short
} catch {
    $check="FAIL_OFFLINE"
    Write-Host ("FALHA NO GATE: " + $_.Exception.Message) -ForegroundColor Red
    $results.Add("failure: " + $_.Exception.Message)
    throw
} finally {
    $finished=Get-Date
    $summary = @(
        "Evydencia CTWA 5287 - validacao local SEM producao",
        "Started: " + $started.ToString("s"),
        "Finished: " + $finished.ToString("s"),
        "Result: " + $check,
        "Git branch: " + $branch,
        "Git SHA: " + $sha,
        "Preview requested: " + [bool]$Preview,
        ($results -join [Environment]::NewLine)
    )
    [IO.File]::WriteAllText($report,($summary -join [Environment]::NewLine))
    Write-Host ("Relatorio sanitizado: " + $report)
}
if ($Preview -and $check -eq "PASS_OFFLINE") {
    Write-Host ""
    Write-Host "=== PAINEL DEMONSTRATIVO LOCAL ===" -ForegroundColor Cyan
    Write-Host "Somente dados inventados, loopback 127.0.0.1, sem APIs externas."
    Write-Host "Copie o endereco que aparecer no terminal. Encerre com Ctrl+C."
    & go run ./cmd/dashboard-preview
    if ($LASTEXITCODE -ne 0) { throw "Preview local falhou." }
}
