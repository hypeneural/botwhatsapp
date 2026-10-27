package dashboard

// dashboardHTML is an isolated, dependency-free and read-only manager view.
// Every value loaded from the journal must be escaped before HTML rendering.
const dashboardHTML = `<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
<title>Evydência | Monitor de Mensagens e Leads CTWA</title>
<style>
:root {
  font-family: system-ui,-apple-system,"Segoe UI",Roboto,Helvetica,Arial,sans-serif;
  color-scheme: light;
  --bg: #f4f6f5;
  --surface: #ffffff;
  --ink: #13221c;
  --muted: #53645c;
  --line: #e2e8e4;
  --line-subtle: #edf1ee;
  --accent: #156d51;
  --accent-hover: #0f543e;
  --accent-soft: #eaf5ef;
  --wa: #25D366;
  --wa-dark: #128C7E;
  --warn: #92540b;
  --warn-bg: #fff5e6;
  --danger: #a53235;
  --danger-bg: #fff1f0;
  --info: #0e7090;
  --info-bg: #e0f2fe;
  --shadow-sm: 0 1px 3px rgba(18,36,28,.04);
  --shadow: 0 4px 20px rgba(18,36,28,.06);
  --radius-lg: 16px;
  --radius-md: 10px;
  --radius-sm: 6px;
}
* { box-sizing: border-box; }
body { margin: 0; background: var(--bg); color: var(--ink); font-size: 14px; line-height: 1.5; }
button, input, select { font: inherit; }
button, select { cursor: pointer; }
button:focus-visible, input:focus-visible, select:focus-visible, summary:focus-visible { outline: 3px solid #6ab69b; outline-offset: 2px; }
.shell { max-width: 1480px; padding: 24px clamp(14px, 3vw, 40px) 48px; margin: auto; }

/* Top Header */
.top { display: flex; justify-content: space-between; align-items: center; gap: 20px; margin-bottom: 22px; }
.brand-eyebrow { display: flex; align-items: center; gap: 8px; color: var(--accent); font-size: 11px; text-transform: uppercase; letter-spacing: .12em; font-weight: 800; }
.brand-badge { background: #dcf2e7; color: var(--accent); padding: 2px 7px; border-radius: 4px; font-weight: 800; }
h1 { font-size: clamp(24px, 2.8vw, 32px); line-height: 1.15; letter-spacing: -.03em; margin: 4px 0 6px; font-weight: 800; color: #0e1e17; }
.description { color: var(--muted); margin: 0; max-width: 720px; font-size: 14px; }
.top-actions { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; }
.select, .search { background: #fff; border: 1px solid #ccd8d1; border-radius: var(--radius-md); padding: 9px 12px; color: var(--ink); min-height: 40px; font-size: 13px; }
.btn { display: inline-flex; align-items: center; justify-content: center; gap: 8px; border: 1px solid var(--accent); background: var(--accent); color: #fff; font-weight: 700; border-radius: var(--radius-md); padding: 9px 16px; min-height: 40px; transition: background .15s ease; font-size: 13px; }
.btn:hover { background: var(--accent-hover); }
.btn[disabled] { opacity: .65; cursor: wait; }

/* Status Bar */
.statusline { display: flex; align-items: center; justify-content: space-between; flex-wrap: wrap; gap: 8px; padding: 10px 16px; background: #eaf4ed; border: 1px solid #d4e6d8; border-radius: var(--radius-md); color: #28543b; margin-bottom: 20px; font-size: 13px; }
.statusline.error { background: var(--warn-bg); border-color: #f7d5a5; color: var(--warn); }
.statusline .minor { color: var(--muted); font-size: 12px; }

/* Metric KPI Cards */
.metrics { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 14px; margin-bottom: 22px; }
.metric { position: relative; border-radius: var(--radius-lg); background: var(--surface); border: 1px solid var(--line); padding: 18px 20px; box-shadow: var(--shadow-sm); overflow: hidden; border-top: 4px solid #ccd8d1; }
.metric.kpi-leads { border-top-color: var(--accent); }
.metric.kpi-accepted { border-top-color: var(--info); }
.metric.kpi-attention { border-top-color: var(--danger); }
.metric.kpi-total { border-top-color: #8da498; }
.metric .heading { font-weight: 700; color: #495c52; font-size: 13px; display: flex; align-items: center; gap: 6px; }
.metric .value { font-size: 32px; font-weight: 800; line-height: 1.1; font-variant-numeric: tabular-nums; display: block; margin: 10px 0 4px; }
.metric.kpi-leads .value { color: var(--accent); }
.metric.kpi-accepted .value { color: var(--info); }
.metric.kpi-attention .value { color: var(--danger); }
.metric .detail { font-size: 12px; color: var(--muted); }

/* Main Section */
.section { background: var(--surface); border: 1px solid var(--line); border-radius: var(--radius-lg); box-shadow: var(--shadow); margin-bottom: 20px; overflow: hidden; }
.section-head { display: flex; justify-content: space-between; align-items: center; gap: 14px; flex-wrap: wrap; padding: 20px 22px 14px; }
.section-head h2 { font-size: 18px; letter-spacing: -.02em; margin: 0; font-weight: 800; }
.section-head .muted { color: var(--muted); font-size: 12px; margin: 4px 0 0; }

/* Priority Tab Bar (Quick Filters) */
.priority-bar { display: flex; align-items: center; gap: 8px; padding: 0 22px 16px; overflow-x: auto; -webkit-overflow-scrolling: touch; }
.tab-chip { display: inline-flex; align-items: center; gap: 8px; padding: 8px 14px; border-radius: 30px; font-size: 13px; font-weight: 650; color: #37473f; background: #eef3f0; border: 1px solid #d8e2dc; transition: all .15s ease; white-space: nowrap; user-select: none; }
.tab-chip:hover { background: #e4ece7; border-color: #cbd8d1; }
.tab-chip.active { background: #156d51; color: #ffffff; border-color: #156d51; box-shadow: 0 2px 6px rgba(21,109,81,.25); }
.tab-chip .chip-dot { width: 8px; height: 8px; border-radius: 50%; display: inline-block; }
.dot-leads { background: #25D366; }
.dot-attention { background: #e53e3e; }
.dot-accepted { background: #38bdf8; }
.dot-private { background: #818cf8; }
.dot-muted { background: #94a3b8; }
.tab-chip.active .chip-dot { background: #ffffff; }
.chip-count { font-size: 11px; padding: 1px 7px; border-radius: 12px; background: rgba(0,0,0,.08); font-variant-numeric: tabular-nums; font-weight: 750; }
.tab-chip.active .chip-count { background: rgba(255,255,255,.25); color: #fff; }

/* Tools Bar */
.tools { display: grid; grid-template-columns: 1.8fr 1fr 1fr auto; gap: 10px; padding: 0 22px 18px; }
.search-wrapper { position: relative; width: 100%; }
.search-wrapper .search-ico { position: absolute; left: 12px; top: 50%; transform: translateY(-50%); width: 16px; height: 16px; color: #8da498; pointer-events: none; }
.search-wrapper .search { width: 100%; padding-left: 36px; }
.tools .select { width: 100%; min-width: 0; }
.results-bar { display: flex; justify-content: space-between; align-items: center; font-size: 12px; color: var(--muted); padding: 10px 22px; border-top: 1px solid var(--line-subtle); background: #fbfdfc; }
.results-bar strong { color: var(--ink); }

/* Data Table */
.scroll { overflow-x: auto; }
.data-table { width: 100%; border-collapse: collapse; table-layout: fixed; }
.data-table th { text-align: left; padding: 13px 16px; color: #53645c; background: #f7faf8; font-size: 11px; font-weight: 800; letter-spacing: .05em; text-transform: uppercase; white-space: nowrap; border-bottom: 1px solid var(--line); }
.data-table td { vertical-align: top; padding: 16px; border-bottom: 1px solid var(--line-subtle); overflow-wrap: anywhere; }
.data-table tr:last-child td { border-bottom: 0; }
.data-table tbody tr:hover { background: #fafcfb; }
.date-time { font-weight: 700; font-variant-numeric: tabular-nums; font-size: 13px; color: #1e332a; }
.cell-name { font-weight: 800; font-size: 14px; color: #0d1e17; margin-bottom: 2px; }
.sub { font-size: 12px; color: var(--muted); margin-top: 3px; }

/* Contact column links */
.wa-link { display: inline-flex; align-items: center; gap: 5px; color: #0e6b46; font-weight: 750; text-decoration: none; font-size: 13px; margin: 3px 0; border-radius: 4px; padding: 2px 4px; margin-left: -4px; transition: background .12s; }
.wa-link:hover { background: #e3f5eb; color: #084d31; text-decoration: underline; }
.wa-link svg { width: 15px; height: 15px; fill: var(--wa-dark); flex-shrink: 0; }

/* Status Badges */
.pill { font-weight: 750; font-size: 11px; border-radius: 30px; padding: 4px 9px; display: inline-flex; align-items: center; gap: 5px; line-height: 1.3; }
.pill.ads { background: #e2f5ec; color: #116246; border: 1px solid #bfe8d3; }
.pill.normal { background: #edf2f0; color: #4e6057; border: 1px solid #dbe4e0; }
.pill.success { background: #e1f5e8; color: #0e633d; border: 1px solid #bde8cb; }
.pill.danger { background: #ffeded; color: #9f2427; border: 1px solid #fcc; }
.pill.warning { background: #fff2db; color: #874e07; border: 1px solid #f9d8a5; }
.pill.info { background: #e0f2fe; color: #0369a1; border: 1px solid #bae6fd; }
.pill.neutral { background: #eff3f1; color: #52635a; border: 1px solid #dbe2de; }

/* Message / Trigger */
.message { font-size: 13px; color: #172a23; max-width: 440px; }
.trigger-badge { display: inline-flex; align-items: center; gap: 5px; font-size: 12px; color: var(--accent); font-weight: 750; margin-top: 6px; background: #e9f6ef; border: 1px solid #c9ebd8; padding: 3px 8px; border-radius: 6px; }

/* Receipt Box with Copy */
.receipt-box { display: inline-flex; align-items: center; gap: 6px; background: #f3f7f5; border: 1px solid #d7e4dc; border-radius: 6px; padding: 3px 6px; margin-top: 5px; font-size: 11px; font-family: ui-monospace, SFMono-Regular, Menlo, monospace; color: #234436; }
.receipt-box code { font-size: 11px; word-break: break-all; }
.btn-copy { background: #fff; border: 1px solid #c8d8ce; border-radius: 4px; padding: 2px 5px; font-size: 10px; color: #375345; font-weight: 700; cursor: pointer; transition: all .12s; display: inline-flex; align-items: center; gap: 3px; }
.btn-copy:hover { background: #e7f3ec; border-color: var(--accent); color: var(--accent); }
.btn-copy.copied { background: #156d51; border-color: #156d51; color: #fff; }

/* API Steps Timeline */
.steps { margin: 8px 0 0; }
.steps summary { display: inline-flex; align-items: center; gap: 6px; color: var(--accent); font-weight: 750; cursor: pointer; font-size: 12px; list-style: none; user-select: none; }
.steps summary::-webkit-details-marker { display: none; }
.steps summary:hover { color: var(--accent-hover); text-decoration: underline; }
.steps summary .chevron { width: 14px; height: 14px; transition: transform .15s ease; fill: currentColor; }
.steps[open] summary .chevron { transform: rotate(90deg); }
.steps[open] { background: #f7faf8; border: 1px solid var(--line); border-radius: var(--radius-md); padding: 12px; margin-top: 8px; }
.timeline { list-style: none; margin: 10px 0 0; padding: 0; }
.timeline li { border-left: 2px solid #cbdad0; margin-left: 8px; padding: 0 0 14px 16px; position: relative; font-size: 12px; line-height: 1.5; }
.timeline li:last-child { padding-bottom: 2px; }
.timeline li:before { content: ""; width: 10px; height: 10px; border-radius: 50%; background: var(--accent); position: absolute; left: -6px; top: 2px; border: 2px solid #fff; }
.timeline li.step-crm:before { background: var(--info); }
.timeline strong { display: block; font-size: 13px; color: #12281f; margin-bottom: 2px; }
.timeline .resp { font-weight: 700; color: #1f3b2f; }
.plain-note { color: var(--muted); font-size: 12px; }

/* Secondary panel & footer */
.secondary-panel { padding: 0 20px; }
.secondary-panel > summary { padding: 18px 2px; font-size: 15px; font-weight: 750; cursor: pointer; }
.secondary-body { border-top: 1px solid var(--line); padding: 18px 0; }
.contact-table { width: 100%; border-collapse: collapse; }
.contact-table th, .contact-table td { text-align: left; border-bottom: 1px solid var(--line); padding: 12px 10px; font-size: 12px; }
.contact-table th { color: var(--muted); font-weight: 750; }
.empty { text-align: center; padding: 40px 16px; color: var(--muted); font-size: 13px; }
footer { color: #64756c; font-size: 12px; padding: 12px 2px 0; }

/* Responsive Adjustments */
@media(max-width: 1060px) {
  .metrics { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .tools { grid-template-columns: 1fr 1fr; }
  .tools .search-wrapper { grid-column: 1 / -1; }
}
@media(max-width:700px) {
  .shell { padding: 14px 10px 30px; }
  .top { display: block; padding: 4px; }
  .description { margin-bottom: 14px; }
  .top-actions { justify-content: space-between; }
  .top-actions select { flex: 1; }
  .metrics { gap: 8px; margin-bottom: 14px; }
  .metric { padding: 14px 12px; border-radius: 12px; }
  .metric .value { font-size: 26px; }
  .priority-bar { padding: 0 14px 12px; }
  .tab-chip { padding: 6px 12px; font-size: 12px; }
  .section { border-radius: 13px; }
  .section-head { padding: 16px 14px 10px; }
  .tools { padding: 0 14px 12px; gap: 8px; grid-template-columns: 1fr 1fr; }
  .tools .search-wrapper { grid-column: 1 / -1; }
  .results-bar { padding: 8px 14px; font-size: 11px; }
  .scroll { overflow: visible; }
  .data-table, .data-table tbody, .data-table tr, .data-table td { display: block; width: 100%; }
  .data-table thead { display: none; }
  .data-table tbody tr { padding: 14px 12px; border-bottom: 8px solid var(--bg); background: #fff; }
  .data-table td { border: 0; padding: 4px 0; }
  .data-table td:before { content: attr(data-label); font-size: 11px; display: block; color: #687970; font-weight: 750; margin-bottom: 2px; }
  .data-table td[data-label="Contato"]:before { display: none; }
  .data-table td[data-label="Resultado"] { padding-top: 8px; }
  .message { max-width: none; }
  .steps[open] { padding: 10px; }
}
@media(prefers-reduced-motion: reduce) { * { animation: none !important; transition: none !important; } }
</style>
</head>
<body>
<div class="shell">
  <!-- Header Principal -->
  <header class="top">
    <div>
      <div class="brand-eyebrow">
        <span>Estúdio Evydência</span>
        <span class="brand-badge">Natal 2026</span>
      </div>
      <h1>Monitor de mensagens</h1>
      <p class="description">Acompanhamento em tempo real de contatos recebidos no WhatsApp, roteamento pelo bot e propostas encaminhadas ao CRM.</p>
    </div>
    <div class="top-actions">
      <select id="refreshInterval" class="select" aria-label="Atualização automática">
        <option value="0">Atualização manual</option>
        <option value="30" selected>Atualizar a cada 30 s</option>
        <option value="60">Atualizar a cada 1 min</option>
        <option value="120">Atualizar a cada 2 min</option>
      </select>
      <button class="btn" id="refreshBtn" type="button">
        <svg viewBox="0 0 20 20" width="15" height="15" fill="currentColor"><path fill-rule="evenodd" d="M4 2a1 1 0 011 1v2.101a7.002 7.002 0 0111.601 2.566 1 1 0 11-1.885.666A5.002 5.002 0 005.999 7H9a1 1 0 010 2H4a1 1 0 01-1-1V3a1 1 0 011-1zm.008 9.057a1 1 0 011.276.61A5.002 5.002 0 0014.001 13H11a1 1 0 110-2h5a1 1 0 011 1v5a1 1 0 11-2 0v-2.101a7.002 7.002 0 01-11.601-2.566 1 1 0 01.61-1.276z" clip-rule="evenodd"/></svg>
        <span>Atualizar agora</span>
      </button>
    </div>
  </header>

  <!-- Barra de Conexão -->
  <div class="statusline" id="connectionLine" role="status" aria-live="polite">
    <span id="statusText">Carregando histórico…</span>
    <span class="minor" id="updatedAt">Fonte: registros locais da stack BotWhatsApp.</span>
  </div>

  <!-- Cartões de Métricas (KPIs) -->
  <div class="metrics" aria-label="Resumo dos registros carregados">
    <div class="metric kpi-leads">
      <div class="heading">
        <svg viewBox="0 0 20 20" width="16" height="16" fill="currentColor"><path d="M10 2a8 8 0 1 0 0 16 8 8 0 0 0 0-16zm1 11.93V14a1 1 0 1 1-2 0v-.07A6.002 6.002 0 0 1 4.07 9H4a1 1 0 1 1 0-2h.07A6.002 6.002 0 0 1 9 4.07V4a1 1 0 1 1 2 0v.07A6.002 6.002 0 0 1 15.93 9H16a1 1 0 1 1 0 2h-.07A6.002 6.002 0 0 1 11 13.93zM10 12a2 2 0 1 0 0-4 2 2 0 0 0 0 4z"/></svg>
        Mensagens com sinal CTWA
      </div>
      <span class="value" id="kpiAds">—</span>
      <span class="detail">Origem indicativa · não representa leads únicos</span>
    </div>

    <div class="metric kpi-accepted">
      <div class="heading">
        <svg viewBox="0 0 20 20" width="16" height="16" fill="currentColor"><path fill-rule="evenodd" d="M10 18a8 8 0 1 0 0-16 8 8 0 0 0 0 16zm3.707-9.293a1 1 0 0 0-1.414-1.414L9 10.586 7.707 9.293a1 1 0 0 0-1.414 1.414l2 2a1 1 0 0 0 1.414 0l4-4z" clip-rule="evenodd"/></svg>
        Propostas Agendadas no CRM
      </div>
      <span class="value" id="kpiAccepted">—</span>
      <span class="detail">Registradas no CRM · entrega ainda não comprovada</span>
    </div>

    <div class="metric kpi-attention">
      <div class="heading">
        <svg viewBox="0 0 20 20" width="16" height="16" fill="currentColor"><path fill-rule="evenodd" d="M8.257 3.099c.765-1.36 2.722-1.36 3.486 0l5.58 9.92c.75 1.334-.213 2.98-1.742 2.98H4.42c-1.53 0-2.493-1.646-1.743-2.98l5.58-9.92zM11 13a1 1 0 1 1-2 0 1 1 0 0 1 2 0zm-1-8a1 1 0 0 0-1 1v3a1 1 0 0 0 2 0V6a1 1 0 0 0-1-1z" clip-rule="evenodd"/></svg>
        Precisam de Atenção
      </div>
      <span class="value" id="kpiAttention">—</span>
      <span class="detail">Rejeições, incertezas ou bloqueios</span>
    </div>

    <div class="metric kpi-total">
      <div class="heading">
        <svg viewBox="0 0 20 20" width="16" height="16" fill="currentColor"><path fill-rule="evenodd" d="M18 10c0 3.866-3.582 7-8 7a8.841 8.841 0 0 1-4.083-.98L2 17l1.338-3.123C2.493 12.767 2 11.434 2 10c0-3.866 3.582-7 8-7s8 3.134 8 7zM7 9H5v2h2V9zm8 0h-2v2h2V9zm-4 0h-2v2h2V9z" clip-rule="evenodd"/></svg>
        Total de Mensagens
      </div>
      <span class="value" id="kpiReceived">—</span>
      <span class="detail">Na janela consultada (inclui broadcast)</span>
    </div>
  </div>

  <!-- Seção Principal de Atendimentos -->
  <section class="section" aria-labelledby="historyHeading">
    <div class="section-head">
      <div>
        <h2 id="historyHeading">Histórico de atendimentos</h2>
        <p class="muted">Mensagens mais recentes primeiro · fuso de Brasília (BRT) · somente leitura</p>
      </div>
    </div>

    <!-- Barra de Filtros Prioritários (Abas Rápidas) -->
    <div class="priority-bar" role="tablist" aria-label="Filtros prioritários">
      <button type="button" class="tab-chip active" id="tabLeads" data-filter="leads">
        <span class="chip-dot dot-leads"></span>
        <span>🎯 Mensagens CTWA</span>
        <span class="chip-count" id="chipLeadsCount">0</span>
      </button>
      <button type="button" class="tab-chip" id="tabAttention" data-filter="attention">
        <span class="chip-dot dot-attention"></span>
        <span>⚠️ Precisam de atenção</span>
        <span class="chip-count" id="chipAttentionCount">0</span>
      </button>
      <button type="button" class="tab-chip" id="tabAccepted" data-filter="accepted">
        <span class="chip-dot dot-accepted"></span>
        <span>✅ Propostas agendadas</span>
        <span class="chip-count" id="chipAcceptedCount">0</span>
      </button>
      <button type="button" class="tab-chip" id="tabPrivate" data-filter="private">
        <span class="chip-dot dot-private"></span>
        <span>💬 Todas as conversas privadas</span>
        <span class="chip-count" id="chipPrivateCount">0</span>
      </button>
      <button type="button" class="tab-chip tab-chip-muted" id="tabAll" data-filter="all">
        <span class="chip-dot dot-muted"></span>
        <span>📦 Ver tudo (inclui status arquivados)</span>
        <span class="chip-count" id="chipAllCount">0</span>
      </button>
    </div>

    <!-- Ferramentas Secundárias (Busca e Filtros Finos) -->
    <div class="tools">
      <div class="search-wrapper">
        <svg class="search-ico" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd"/></svg>
        <input id="searchEvents" class="search" type="search" aria-label="Pesquisar histórico" placeholder="Buscar por nome, telefone ou mensagem...">
      </div>
      <select id="filterSource" class="select" aria-label="Filtrar origem">
        <option value="all">Todas as origens</option>
        <option value="meta_ads">Tráfego pago identificado</option>
        <option value="normal">Sem marcação de anúncio</option>
      </select>
      <select id="filterStatus" class="select" aria-label="Filtrar situação">
        <option value="all">Todas as situações</option>
        <option value="attention">Precisam de atenção</option>
        <option value="crm_aceito">CRM agendou</option>
        <option value="crm_rejeitado">CRM rejeitou</option>
        <option value="bloqueado_local">Bot bloqueou antes do CRM</option>
        <option value="crm_nao_registrado">Sem etapa CRM registrada</option>
        <option value="incerto">Resultado incerto</option>
        <option value="pendente">Pendente</option>
        <option value="sem_tarefas">Sem execução registrada</option>
        <option value="finalizado_local">Atendimento local finalizado</option>
        <option value="ignorado">Ignorados / bloqueados</option>
      </select>
      <select id="pageLimit" class="select" aria-label="Tamanho da janela">
        <option value="50">50 mensagens</option>
        <option value="150" selected>150 mensagens</option>
        <option value="500">500 mensagens</option>
      </select>
    </div>

    <div class="results-bar">
      <span id="resultCount">Aguardando registros...</span>
      <span id="activeFilterDesc">Modo: <strong>🎯 Mensagens CTWA</strong> (status e grupos ocultados)</span>
    </div>

    <div class="scroll">
      <table class="data-table" aria-label="Eventos recebidos pelo bot">
        <colgroup>
          <col style="width: 14%">
          <col style="width: 22%">
          <col style="width: 29%">
          <col style="width: 18%">
          <col style="width: 17%">
        </colgroup>
        <thead>
          <tr>
            <th>Recebido</th>
            <th>Contato</th>
            <th>Mensagem / Gatilho</th>
            <th>Resultado</th>
            <th>Histórico de APIs</th>
          </tr>
        </thead>
        <tbody id="eventRows">
          <tr><td colspan="5" class="empty">Carregando eventos...</td></tr>
        </tbody>
      </table>
    </div>
  </section>

  <!-- Seção Opcional de Contatos Agrupados -->
  <section class="section" aria-label="Resumo de contatos">
    <details class="secondary-panel" id="contactsDetails">
      <summary>Ver contatos únicos agrupados por telefone (visão complementar)</summary>
      <div class="secondary-body">
        <p class="description">Esta visão consolida mensagens repetidas do mesmo telefone em um único registro.</p>
        <div class="search-wrapper" style="max-width:340px; margin-bottom:14px;">
          <svg class="search-ico" viewBox="0 0 20 20" fill="currentColor"><path fill-rule="evenodd" d="M8 4a4 4 0 100 8 4 4 0 000-8zM2 8a6 6 0 1110.89 3.476l4.817 4.817a1 1 0 01-1.414 1.414l-4.816-4.816A6 6 0 012 8z" clip-rule="evenodd"/></svg>
          <input id="searchContacts" class="search" type="search" placeholder="Buscar contato ou telefone..." aria-label="Buscar contatos">
        </div>
        <div class="scroll">
          <table class="contact-table">
            <thead>
              <tr>
                <th>Última Atividade (BRT)</th>
                <th>Nome</th>
                <th>Telefone</th>
                <th>Origem do Fluxo</th>
                <th>CRM</th>
              </tr>
            </thead>
            <tbody id="contactRows">
              <tr><td colspan="5" class="empty">Abra para carregar a lista de contatos.</td></tr>
            </tbody>
          </table>
        </div>
      </div>
    </details>
  </section>

  <footer>Fonte: journal e outbox seguros do BotWhatsApp · Sem ações destrutivas pelo painel · Estúdio Evydência.</footer>
</div>

<script>
'use strict';
const state = {
  events: [],
  contacts: [],
  contactsLoaded: false,
  loading: false,
  timer: null,
  lastSuccess: null,
  currentTab: 'leads'
};

const $ = id => document.getElementById(id);

function apiURL(path) {
  return new URL(path, window.location.origin);
}

function safe(value) {
  return String(value == null ? '' : value).replace(/[&<>"']/g, ch =>
    ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
}

function badge(label, kind, icon) {
  return '<span class="pill '+kind+'">'+(icon ? icon+' ' : '')+safe(label)+'</span>';
}

const statusLabels = {
  crm_aceito: ['CRM agendou', 'success'],
  crm_rejeitado: ['CRM rejeitou', 'danger'],
  incerto: ['Resultado incerto', 'danger'],
  pendente: ['Pendente', 'warning'],
  em_processamento: ['Em andamento', 'info'],
  sem_tarefas: ['Sem execução registrada', 'neutral'],
  bloqueado_local: ['Bloqueado antes do CRM', 'warning'],
  crm_nao_registrado: ['Etapa CRM não registrada', 'warning'],
  finalizado_local: ['Atendimento local finalizado', 'neutral'],
  ignorado: ['Ignorado', 'neutral'],
  opt_out: ['Contato pediu para parar', 'warning'],
  telefone_nao_resolvido: ['Telefone pendente de identificação', 'warning'],
  recebido: ['Recebido', 'neutral']
};

function formatReason(code) {
  switch (code) {
    case 'non_private_chat': return 'Status ou grupo (não privado)';
    case 'unresolved_phone_or_non_private_chat': return 'Identificador sem telefone comprovado';
    case 'not_new_inbound': return 'Mensagem do próprio bot / sistema';
    case 'unknown_instance': return 'Instância do WhatsApp não configurada';
    case 'no_text': return 'Mídia recebida sem legenda / texto';
    case 'opt_out_requires_persistent_suppression': return 'Contato pediu para parar (opt-out)';
    default: return code || '';
  }
}

function showStatus(msg, error) {
  $('connectionLine').classList.toggle('error', Boolean(error));
  $('statusText').textContent = msg;
  $('updatedAt').textContent = state.lastSuccess ? 'Última leitura: ' + state.lastSuccess + ' (BRT)' : 'Fonte: registros locais do bot.';
}

window.copyReceipt = function(text, btn) {
  if (!navigator.clipboard) return;
  navigator.clipboard.writeText(text).then(() => {
    const orig = btn.innerHTML;
    btn.innerHTML = '✓ Copiado';
    btn.classList.add('copied');
    setTimeout(() => {
      btn.innerHTML = orig;
      btn.classList.remove('copied');
    }, 1800);
  });
};

function stepsHTML(steps, status) {
  if (!Array.isArray(steps) || !steps.length) {
    if (status === 'telefone_nao_resolvido') return '<span class="plain-note">Bot bloqueou as chamadas: telefone ainda não confirmado. Nenhuma proposta foi solicitada ao CRM.</span>';
    if (status === 'ignorado') return '<span class="plain-note">Mensagem ignorada pelas regras do bot. Nenhuma chamada de API comercial foi necessária.</span>';
    if (status === 'sem_tarefas') return '<span class="plain-note">Evento recebido, sem tarefas duráveis registradas. Conferir journal antes de tentar novamente.</span>';
    return '<span class="plain-note">Não há tarefas de API vinculadas a este evento.</span>';
  }
  const count = steps.length;
  return '<details class="steps"><summary>' +
    '<svg class="chevron" viewBox="0 0 20 20"><path d="M7.293 14.707a1 1 0 010-1.414L10.586 10 7.293 6.707a1 1 0 011.414-1.414l4 4a1 1 0 010 1.414l-4 4a1 1 0 01-1.414 0z"/></svg>' +
    '<span>Ver ' + count + ' etapa(s) registrada(s)</span></summary>' +
    '<ol class="timeline">' +
    steps.map(t => {
      const isCRM = t.provider && t.provider.includes('CRM');
      const cls = isCRM ? ' class="step-crm"' : '';
      return '<li' + cls + '><strong>' + safe(t.step) + '</strong>' +
        '<div>' + safe(t.provider) + ' · <span class="resp">' + safe(t.api_response) + '</span></div>' +
        (t.started_brt ? '<div class="sub">Início: ' + safe(t.started_brt) + '</div>' : '') +
        (t.completed_brt ? '<div class="sub">Fim: ' + safe(t.completed_brt) + '</div>' : '') +
        (t.external_id ? '<div class="sub">Recibo: ' + safe(t.external_id) + '</div>' : '') +
        (t.reason ? '<div class="sub">Motivo: ' + safe(formatReason(t.reason)) + '</div>' : '') +
        '</li>';
    }).join('') +
    '</ol></details>';
}

function updateTabCounts() {
  let leadsCount = 0, attentionCount = 0, acceptedCount = 0, privateCount = 0;
  for (const e of state.events) {
    const isStatusOrGroup = e.decision_reason === 'non_private_chat' || (e.phone && e.phone.includes('@broadcast'));
    if (!isStatusOrGroup) {
      privateCount++;
      const isLead = e.source === 'meta_ads' || e.trigger_matched || e.has_ad_context || e.decision === 'ads_proposal';
      if (isLead) leadsCount++;
      if (e.needs_attention) attentionCount++;
      if (e.status === 'crm_aceito' || e.crm_receipt) acceptedCount++;
    }
  }
  $('chipLeadsCount').textContent = leadsCount;
  $('chipAttentionCount').textContent = attentionCount;
  $('chipAcceptedCount').textContent = acceptedCount;
  $('chipPrivateCount').textContent = privateCount;
  $('chipAllCount').textContent = state.events.length;
}

function renderEvents() {
  const currentTab = state.currentTab;
  const source = $('filterSource').value;
  const filter = $('filterStatus').value;
  const term = $('searchEvents').value.trim().toLocaleLowerCase('pt-BR');

  const descriptions = {
    leads: 'Modo: 🎯 Mensagens CTWA (status e grupos ocultados)',
    attention: 'Modo: ⚠️ Precisam de Atenção (rejeições, incertezas ou bloqueios)',
    accepted: 'Modo: ✅ Propostas Agendadas no CRM',
    private: 'Modo: 💬 Todas as Conversas Privadas (status e grupos ocultados)',
    all: 'Modo: 📦 Visão Completa (inclui status arquivados e transmissões)'
  };
  $('activeFilterDesc').innerHTML = descriptions[currentTab] || '';

  const found = state.events.filter(e => {
    const isStatusOrGroup = e.decision_reason === 'non_private_chat' || (e.phone && e.phone.includes('@broadcast'));

    // Priority Tab rules:
    if (currentTab === 'leads') {
      if (isStatusOrGroup) return false;
      const isLead = e.source === 'meta_ads' || e.trigger_matched || e.has_ad_context || e.decision === 'ads_proposal';
      if (!isLead) return false;
    } else if (currentTab === 'attention') {
      if (isStatusOrGroup) return false;
      if (!e.needs_attention) return false;
    } else if (currentTab === 'accepted') {
      if (e.status !== 'crm_aceito' && !e.crm_receipt) return false;
    } else if (currentTab === 'private') {
      if (isStatusOrGroup) return false;
    }
    // 'all' includes everything.

    // Secondary dropdown filters:
    if (source !== 'all' && source !== e.source) return false;
    if (filter === 'attention' && !e.needs_attention) return false;
    if (filter === 'ignorado' && e.status !== 'ignorado' && e.status !== 'opt_out') return false;
    if (filter !== 'all' && filter !== 'attention' && filter !== 'ignorado' && filter !== e.status) return false;

    // Search term:
    return !term || [e.name, e.phone, e.inbound_text, e.trigger_text, e.ad_source_id, e.crm_receipt, e.decision_reason]
      .some(t => String(t || '').toLocaleLowerCase('pt-BR').includes(term));
  });

  $('resultCount').innerHTML = 'Exibindo <strong>' + found.length + '</strong> de ' + state.events.length + ' mensagens carregadas';

  if (!found.length) {
    $('eventRows').innerHTML = '<tr><td colspan="5" class="empty">Nenhum atendimento encontrado nesta visão. Escolha outra aba ou ajuste a busca.</td></tr>';
    return;
  }

  $('eventRows').innerHTML = found.map(e => {
    const label = statusLabels[e.status] || ['Estado não identificado', 'neutral'];
    const sourceTag = e.source === 'meta_ads'
      ? badge(e.ad_source_id ? 'Referência de anúncio' : 'Indício de anúncio', 'ads', '<svg viewBox="0 0 20 20" width="12" height="12" fill="currentColor"><path d="M10 2a8 8 0 100 16 8 8 0 000-16zm1 11.93V14a1 1 0 11-2 0v-.07A6.002 6.002 0 014.07 9H4a1 1 0 110-2h.07A6.002 6.002 0 019 4.07V4a1 1 0 112 0v.07A6.002 6.002 0 0115.93 9H16a1 1 0 110 2h-.07A6.002 6.002 0 0111 13.93zM10 12a2 2 0 100-4 2 2 0 000 4z"/></svg>')
      : badge('Sem marcação', 'normal');

    const nameSource = e.name_origin === 'crm' ? 'Nome confirmado no CRM' :
      (e.name_origin === 'whatsapp' ? 'Nome informado pelo WhatsApp' : 'Nome não disponível');

    // Only form links for validated BR national formats, never from a LID,
    // arbitrary text, or a prefixed number that could become 5555...
    const rawDigits = String(e.phone || '').replace(/\D/g, '');
    const validNational = /^\d{2}\s*\d{4,5}\s*\d{4}$/.test(
      String(e.phone || '').replace(/[()\-]/g, ' ').replace(/\s+/g, ' ').trim()
    ) && (rawDigits.length === 10 || rawDigits.length === 11);
    const phoneDisplay = validNational
      ? '<a href="https://wa.me/55' + rawDigits + '" target="_blank" rel="noopener noreferrer" class="wa-link" title="Abrir conversa no WhatsApp Web">' +
        '<svg viewBox="0 0 24 24"><path d="M12.04 2c-5.46 0-9.91 4.45-9.91 9.91 0 1.75.46 3.45 1.32 4.95L2.05 22l5.25-1.38c1.45.79 3.08 1.21 4.74 1.21 5.46 0 9.91-4.45 9.91-9.91 0-2.65-1.03-5.14-2.9-7.01A9.82 9.82 0 0 0 12.04 2zm0 18.15c-1.48 0-2.93-.4-4.2-1.15l-.3-.18-3.12.82.83-3.04-.2-.31a8.19 8.19 0 0 1-1.26-4.38c0-4.54 3.7-8.24 8.24-8.24 2.2 0 4.27.86 5.82 2.42a8.18 8.18 0 0 1 2.41 5.83c.01 4.54-3.69 8.23-8.22 8.23z"/></svg>' +
        '<span>' + safe(e.phone) + '</span></a>'
      : '<div class="sub">' + safe(e.phone || 'Telefone não identificado') + '</div>';

    const message = safe(e.inbound_text || '(texto não registrado)');
    const reference = e.ad_source_id ?
      '<div class="sub">Referência de origem: <code>' + safe(e.ad_source_id) + '</code> · conferir Ad ID na Meta</div>' : '';
    const trigger = e.trigger_matched ?
      '<div class="trigger-badge">✓ Gatilho: ' + safe(e.trigger_text || '') + '</div>' :
      '<div class="sub">Sem gatilho exato' + (e.has_ad_context ? ' · contexto Meta presente' : '') + '</div>';

    let receiptHTML = '';
    if (e.crm_receipt) {
      receiptHTML = '<div class="receipt-box"><code>' + safe(e.crm_receipt) + '</code>' +
        '<button type="button" class="btn-copy" onclick="copyReceipt(\'' + safe(e.crm_receipt) + '\', this)" title="Copiar recibo">Copiar</button></div>';
    }

    return '<tr>' +
      '<td data-label="Recebido"><div class="date-time">' + safe(e.received_at_brt || 'Data indisponível') + '</div></td>' +
      '<td data-label="Contato"><div class="cell-name">' + safe(e.name || 'Nome não informado') + '</div>' +
        '<div class="sub">' + safe(nameSource) + '</div>' +
        phoneDisplay +
        '<div style="margin-top:6px">' + sourceTag + '</div></td>' +
      '<td data-label="Mensagem / Gatilho"><div class="message">' + message + '</div>' +
        '<div class="sub">Tipo: ' + safe(e.inbound_type || 'Não informado') + '</div>' + trigger + reference + '</td>' +
      '<td data-label="Resultado">' + badge(label[0], label[1]) +
        receiptHTML +
        (e.status === 'crm_aceito' ? '<div class="sub">Despacho registrado, entrega ao cliente não verificada.</div>' : '') +
        (e.status === 'crm_nao_registrado' ? '<div class="sub">O CRM não tem uma etapa confirmada neste evento; verificar a fila sem reenviar automaticamente.</div>' : '') +
        (e.decision_reason ? '<div class="sub">' + safe(formatReason(e.decision_reason)) + '</div>' : '') + '</td>' +
      '<td data-label="Histórico de APIs">' + stepsHTML(e.steps, e.status) + '</td>' +
      '</tr>';
  }).join('');
}

function renderContacts() {
  const term = $('searchContacts').value.trim().toLocaleLowerCase('pt-BR');
  const found = state.contacts.filter(c => !term || [c.name, c.phone_raw, c.phone_formatted]
    .some(x => String(x || '').toLocaleLowerCase('pt-BR').includes(term)));
  $('contactRows').innerHTML = found.length ? found.map(c =>
    '<tr><td>' + safe(c.last_interaction_brt) + '</td>' +
    '<td><strong>' + safe(c.name) + '</strong></td>' +
    '<td>' + safe(c.phone_formatted) + '</td>' +
    '<td>' + safe(c.channel) + '</td>' +
    '<td>' + safe(c.crm_status) + '</td></tr>').join('') :
    '<tr><td colspan="5" class="empty">Nenhum contato encontrado.</td></tr>';
}

async function loadContacts() {
  if (state.contactsLoaded) return;
  try {
    const res = await fetch(apiURL('/api/dashboard/contacts'), { credentials: 'same-origin', cache: 'no-store' });
    if (!res.ok) throw new Error('Falha HTTP ' + res.status);
    const data = await res.json();
    state.contacts = Array.isArray(data.contacts) ? data.contacts : [];
    state.contactsLoaded = true;
    renderContacts();
  } catch (_) {
    $('contactRows').innerHTML = '<tr><td colspan="5" class="empty">Contatos indisponíveis no momento. Tente fechar e reabrir esta seção.</td></tr>';
  }
}

async function refreshData() {
  if (state.loading) return;
  state.loading = true;
  $('refreshBtn').disabled = true;
  showStatus('Consultando histórico do bot...', false);
  try {
    const res = await fetch(apiURL('/api/dashboard/events?limit=' + encodeURIComponent($('pageLimit').value)),
      { credentials: 'same-origin', cache: 'no-store' });
    if (!res.ok) throw new Error('Falha HTTP ' + res.status);
    const data = await res.json();
    if (!Array.isArray(data.events) || !data.stats) throw new Error('Formato inesperado');
    state.events = data.events;
    $('kpiReceived').textContent = data.stats.received || 0;
    $('kpiAds').textContent = data.stats.meta_ads || 0;
    $('kpiAccepted').textContent = data.stats.crm_accepted || 0;
    $('kpiAttention').textContent = data.stats.attention == null
      ? (data.stats.crm_rejected || 0) + (data.stats.uncertain || 0)
      : data.stats.attention;
    state.lastSuccess = new Date().toLocaleString('pt-BR', { timeZone: 'America/Sao_Paulo', hour12: false });
    showStatus('Histórico consultado com sucesso.', false);
    updateTabCounts();
    renderEvents();
    if ($('contactsDetails').open && state.contactsLoaded) {
      state.contactsLoaded = false;
      await loadContacts();
    }
  } catch (_) {
    showStatus('Não foi possível atualizar. Os dados anteriores continuam visíveis.', true);
    if (!state.lastSuccess) $('eventRows').innerHTML = '<tr><td colspan="5" class="empty">Histórico indisponível. Verifique o acesso ao bot.</td></tr>';
  } finally {
    state.loading = false;
    $('refreshBtn').disabled = false;
  }
}

function configureTimer() {
  if (state.timer) clearInterval(state.timer);
  const seconds = Number($('refreshInterval').value);
  if (seconds > 0) state.timer = setInterval(() => {
    if (!document.hidden) refreshData();
  }, seconds * 1000);
}

// Priority Tab click handling
document.querySelectorAll('.tab-chip').forEach(btn => {
  btn.addEventListener('click', () => {
    document.querySelectorAll('.tab-chip').forEach(b => b.classList.remove('active'));
    btn.classList.add('active');
    state.currentTab = btn.getAttribute('data-filter') || 'leads';
    renderEvents();
  });
});

$('refreshBtn').addEventListener('click', refreshData);
$('refreshInterval').addEventListener('change', configureTimer);
$('pageLimit').addEventListener('change', refreshData);
['searchEvents', 'filterSource', 'filterStatus'].forEach(id => {
  $(id).addEventListener(id === 'searchEvents' ? 'input' : 'change', renderEvents);
});
$('contactsDetails').addEventListener('toggle', () => {
  if ($('contactsDetails').open) loadContacts();
});
$('searchContacts').addEventListener('input', renderContacts);

configureTimer();
refreshData();
</script>
</body>
</html>`
