package dashboard

const dashboardHTML = `<!DOCTYPE html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Estúdio Evydência — Painel de Contatos & Tráfego</title>
<style>
:root {
  --bg: #0f172a;
  --card-bg: #1e293b;
  --border: #334155;
  --text: #f8fafc;
  --text-muted: #94a3b8;
  --primary: #3b82f6;
  --primary-hover: #2563eb;
  --success-bg: #064e3b;
  --success-text: #34d399;
  --success-border: #059669;
  --ads-bg: #14532d;
  --ads-text: #86efac;
  --ads-border: #22c55e;
  --organic-bg: #1e3a8a;
  --organic-text: #93c5fd;
  --organic-border: #3b82f6;
  --warn-bg: #78350f;
  --warn-text: #fcd34d;
  --danger-bg: #7f1d1d;
  --danger-text: #fca5a5;
  --radius: 12px;
}
* { box-sizing: border-box; margin: 0; padding: 0; }
body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
  background-color: var(--bg);
  color: var(--text);
  line-height: 1.5;
  padding: 24px;
}
.container { max-width: 1400px; margin: 0 auto; }
header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 16px;
  padding-bottom: 24px;
  border-bottom: 1px solid var(--border);
  margin-bottom: 24px;
}
.title-group h1 { font-size: 24px; font-weight: 700; display: flex; align-items: center; gap: 10px; }
.title-group p { font-size: 14px; color: var(--text-muted); margin-top: 4px; }
.live-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: rgba(34, 197, 94, 0.15);
  color: #4ade80;
  border: 1px solid rgba(34, 197, 94, 0.4);
  padding: 4px 10px;
  border-radius: 9999px;
  font-size: 12px;
  font-weight: 600;
}
.live-dot {
  width: 8px;
  height: 8px;
  background: #22c55e;
  border-radius: 50%;
  box-shadow: 0 0 8px #22c55e;
  animation: pulse 2s infinite;
}
@keyframes pulse { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
.header-actions { display: flex; align-items: center; gap: 12px; }
button.btn {
  background: var(--primary);
  color: #fff;
  border: none;
  padding: 8px 16px;
  border-radius: 8px;
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  transition: background 0.2s;
}
button.btn:hover { background: var(--primary-hover); }
select.dropdown {
  background: var(--card-bg);
  color: var(--text);
  border: 1px solid var(--border);
  padding: 8px 12px;
  border-radius: 8px;
  font-size: 13px;
  cursor: pointer;
}
.metrics-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}
.metric-card {
  background: var(--card-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  padding: 16px 20px;
  display: flex;
  flex-direction: column;
}
.metric-title { font-size: 13px; color: var(--text-muted); font-weight: 500; }
.metric-val { font-size: 28px; font-weight: 700; margin-top: 6px; }
.metric-sub { font-size: 12px; margin-top: 4px; color: var(--text-muted); }
.val-ads { color: var(--ads-text); }
.val-org { color: var(--organic-text); }
.val-crm { color: var(--success-text); }
.val-cooldown { color: var(--warn-text); }
.controls-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  flex-wrap: wrap;
  gap: 12px;
  margin-bottom: 16px;
  background: var(--card-bg);
  padding: 12px 16px;
  border-radius: var(--radius);
  border: 1px solid var(--border);
}
.search-box {
  display: flex;
  align-items: center;
  background: var(--bg);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 6px 12px;
  min-width: 280px;
}
.search-box input {
  background: transparent;
  border: none;
  outline: none;
  color: var(--text);
  font-size: 14px;
  width: 100%;
}
.filter-tabs { display: flex; gap: 8px; flex-wrap: wrap; }
.tab-btn {
  background: transparent;
  border: 1px solid var(--border);
  color: var(--text-muted);
  padding: 6px 14px;
  border-radius: 6px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.2s;
}
.tab-btn:hover { color: var(--text); border-color: var(--primary); }
.tab-btn.active {
  background: var(--primary);
  color: #fff;
  border-color: var(--primary);
}
.table-wrapper {
  background: var(--card-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius);
  overflow-x: auto;
}
table {
  width: 100%;
  border-collapse: collapse;
  text-align: left;
  font-size: 14px;
}
th {
  background: rgba(15, 23, 42, 0.6);
  color: var(--text-muted);
  font-weight: 600;
  font-size: 12px;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
}
td {
  padding: 14px 16px;
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
}
tr:last-child td { border-bottom: none; }
tr:hover td { background: rgba(255, 255, 255, 0.02); }
.lead-info { display: flex; align-items: center; gap: 10px; }
.avatar {
  width: 34px;
  height: 34px;
  border-radius: 50%;
  background: #334155;
  color: #e2e8f0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 13px;
  flex-shrink: 0;
}
.avatar.ads { background: #166534; color: #bbf7d0; }
.avatar.org { background: #1e40af; color: #bfdbfe; }
.lead-name { font-weight: 600; color: var(--text); }
.badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border-radius: 6px;
  font-size: 12px;
  font-weight: 600;
  white-space: nowrap;
}
.badge-ads { background: var(--ads-bg); color: var(--ads-text); border: 1px solid var(--ads-border); }
.badge-org { background: var(--organic-bg); color: var(--organic-text); border: 1px solid var(--organic-border); }
.badge-crm-done { background: var(--success-bg); color: var(--success-text); border: 1px solid var(--success-border); }
.badge-crm-none { background: rgba(148, 163, 184, 0.1); color: var(--text-muted); }
.badge-cooldown { background: var(--warn-bg); color: var(--warn-text); border: 1px solid #d97706; }
.badge-optout { background: var(--danger-bg); color: var(--danger-text); border: 1px solid #ef4444; }
.time-col { color: var(--text-muted); font-size: 13px; white-space: nowrap; }
.empty-state { text-align: center; padding: 48px; color: var(--text-muted); }
footer {
  margin-top: 24px;
  display: flex;
  justify-content: space-between;
  align-items: center;
  font-size: 12px;
  color: var(--text-muted);
}
</style>
</head>
<body>
<div class="container">
  <header>
    <div class="title-group">
      <h1>Estúdio Evydência <span class="live-badge"><span class="live-dot"></span> Ao Vivo</span></h1>
      <p>Instância Ativa: <strong>evydencia (48 99642-5287)</strong> &bull; Tráfego Pago & Atendimento</p>
    </div>
    <div class="header-actions">
      <select id="autoRefreshSelect" class="dropdown" onchange="setupAutoRefresh()">
        <option value="0">Auto-refresh: Desligado</option>
        <option value="15" selected>Atualizar a cada 15s</option>
        <option value="30">Atualizar a cada 30s</option>
        <option value="60">Atualizar a cada 60s</option>
      </select>
      <button class="btn" onclick="fetchData()">🔄 Atualizar</button>
    </div>
  </header>

  <div class="metrics-grid">
    <div class="metric-card">
      <span class="metric-title">Total de Contatos</span>
      <span class="metric-val" id="metricTotal">—</span>
      <span class="metric-sub">Base registrada em disco</span>
    </div>
    <div class="metric-card">
      <span class="metric-title">🎯 Tráfego Pago (Meta Ads)</span>
      <span class="metric-val val-ads" id="metricAds">—</span>
      <span class="metric-sub">Funil de Natal ativado</span>
    </div>
    <div class="metric-card">
      <span class="metric-title">💬 Orgânico (Saudação)</span>
      <span class="metric-val val-org" id="metricOrg">—</span>
      <span class="metric-sub">Atendimento / Menu</span>
    </div>
    <div class="metric-card">
      <span class="metric-title">📅 Agendamentos CRM 416</span>
      <span class="metric-val val-crm" id="metricCRM">—</span>
      <span class="metric-sub">Propostas confirmadas</span>
    </div>
    <div class="metric-card">
      <span class="metric-title">⏱️ Em Cooldown (24h)</span>
      <span class="metric-val val-cooldown" id="metricCooldown">—</span>
      <span class="metric-sub">Protegidos contra duplicação</span>
    </div>
  </div>

  <div class="controls-bar">
    <div class="filter-tabs">
      <button class="tab-btn active" onclick="setFilter('all', this)">Todos (<span id="countAll">0</span>)</button>
      <button class="tab-btn" onclick="setFilter('ads', this)">🎯 Meta Ads (<span id="countAds">0</span>)</button>
      <button class="tab-btn" onclick="setFilter('organic', this)">💬 Orgânico (<span id="countOrg">0</span>)</button>
      <button class="tab-btn" onclick="setFilter('crm', this)">📅 CRM 416 (<span id="countCRM">0</span>)</button>
      <button class="tab-btn" onclick="setFilter('cooldown', this)">⏱️ Cooldown (<span id="countCooldown">0</span>)</button>
    </div>
    <div class="search-box">
      <input type="text" id="searchInput" placeholder="Buscar por nome, telefone..." oninput="renderTable()">
    </div>
  </div>

  <div class="table-wrapper">
    <table>
      <thead>
        <tr>
          <th>Data / Hora (BRT)</th>
          <th>Lead / Contato</th>
          <th>Telefone</th>
          <th>Canal de Entrada</th>
          <th>Campanha CRM 416</th>
          <th>Última Etapa Executada</th>
          <th>Status do Contato</th>
        </tr>
      </thead>
      <tbody id="contactsTableBody">
        <tr><td colspan="7" class="empty-state">Carregando contatos...</td></tr>
      </tbody>
    </table>
  </div>

  <footer>
    <span>Última sincronização: <strong id="lastSyncTime">—</strong></span>
    <span>BotWhatsApp Staging &bull; Evydência CRM Read-Only &bull; Evolution Go 0.7.2</span>
  </footer>
</div>

<script>
let currentContacts = [];
let activeFilter = 'all';
let refreshTimer = null;

function getInitials(name) {
  if (!name || name === "Cliente WhatsApp") return "📱";
  const parts = name.trim().split(" ");
  if (parts.length === 1) return parts[0].substring(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

function setFilter(filter, el) {
  activeFilter = filter;
  document.querySelectorAll('.tab-btn').forEach(b => b.classList.remove('active'));
  if (el) el.classList.add('active');
  renderTable();
}

async function fetchData() {
  try {
    const res = await fetch('/api/dashboard/contacts' + window.location.search);
    if (!res.ok) throw new Error('HTTP ' + res.status);
    const data = await res.json();
    currentContacts = data.contacts || [];
    
    // Update stats cards
    document.getElementById('metricTotal').textContent = data.stats.total_contacts || 0;
    document.getElementById('metricAds').textContent = data.stats.ads_contacts || 0;
    document.getElementById('metricOrg').textContent = data.stats.organic_contacts || 0;
    document.getElementById('metricCRM').textContent = data.stats.crm_scheduled || 0;
    document.getElementById('metricCooldown').textContent = data.stats.active_cooldown || 0;
    document.getElementById('lastSyncTime').textContent = data.stats.generated_at_brt || 'Agora';

    // Update tab counts
    document.getElementById('countAll').textContent = currentContacts.length;
    document.getElementById('countAds').textContent = currentContacts.filter(c => c.is_ads).length;
    document.getElementById('countOrg').textContent = currentContacts.filter(c => !c.is_ads).length;
    document.getElementById('countCRM').textContent = currentContacts.filter(c => c.crm_schedule_id).length;
    document.getElementById('countCooldown').textContent = currentContacts.filter(c => c.cooldown_active).length;

    renderTable();
  } catch (err) {
    console.error('Falha ao buscar dados:', err);
  }
}

function renderTable() {
  const query = (document.getElementById('searchInput').value || '').toLowerCase().trim();
  const tbody = document.getElementById('contactsTableBody');
  
  const filtered = currentContacts.filter(c => {
    // Filter by tab
    if (activeFilter === 'ads' && !c.is_ads) return false;
    if (activeFilter === 'organic' && c.is_ads) return false;
    if (activeFilter === 'crm' && !c.crm_schedule_id) return false;
    if (activeFilter === 'cooldown' && !c.cooldown_active) return false;

    // Search query
    if (query) {
      const matchName = (c.name || '').toLowerCase().includes(query);
      const matchPhone = (c.phone_raw || '').includes(query) || (c.phone_formatted || '').includes(query);
      const matchStep = (c.last_step || '').toLowerCase().includes(query);
      const matchCRM = (c.crm_schedule_id || '').toLowerCase().includes(query);
      return matchName || matchPhone || matchStep || matchCRM;
    }
    return true;
  });

  if (filtered.length === 0) {
    tbody.innerHTML = '<tr><td colspan="7" class="empty-state">Nenhum contato encontrado para este filtro.</td></tr>';
    return;
  }

  tbody.innerHTML = filtered.map(c => {
    const avatarClass = c.is_ads ? 'ads' : 'org';
    const channelBadge = c.is_ads 
      ? '<span class="badge badge-ads">🎯 Meta Ads</span>' 
      : '<span class="badge badge-org">💬 Orgânico</span>';
    
    let crmBadge = '<span class="badge badge-crm-none">—</span>';
    if (c.crm_schedule_id) {
      crmBadge = '<span class="badge badge-crm-done">✅ Agendado (' + c.crm_schedule_id + ')</span>';
    } else if (c.is_ads && c.crm_status === "Pendente") {
      crmBadge = '<span class="badge badge-cooldown">⏳ Pendente</span>';
    }

    let statusBadges = [];
    if (c.opt_out) {
      statusBadges.push('<span class="badge badge-optout">🔴 Parar / Opt-Out</span>');
    } else if (c.human_paused) {
      statusBadges.push('<span class="badge badge-cooldown">⏸️ Pausa Humana</span>');
    } else if (c.cooldown_active) {
      statusBadges.push('<span class="badge badge-cooldown">⏱️ Cooldown (' + (c.cooldown_remaining || '24h') + ')</span>');
    } else {
      statusBadges.push('<span class="badge badge-crm-done">🟢 Ativo</span>');
    }

    return '<tr>' +
      '<td class="time-col">' + (c.last_interaction_brt || '—') + '</td>' +
      '<td>' +
        '<div class="lead-info">' +
          '<div class="avatar ' + avatarClass + '">' + getInitials(c.name) + '</div>' +
          '<div class="lead-name">' + (c.name || 'Cliente WhatsApp') + '</div>' +
        '</div>' +
      '</td>' +
      '<td><strong>' + c.phone_formatted + '</strong></td>' +
      '<td>' + channelBadge + '</td>' +
      '<td>' + crmBadge + '</td>' +
      '<td>' + c.last_step + '</td>' +
      '<td>' + statusBadges.join(' ') + '</td>' +
    '</tr>';
  }).join('');
}

function setupAutoRefresh() {
  if (refreshTimer) clearInterval(refreshTimer);
  const val = parseInt(document.getElementById('autoRefreshSelect').value, 10);
  if (val > 0) {
    refreshTimer = setInterval(fetchData, val * 1000);
  }
}

// Initial fetch and start auto-refresh
fetchData();
setupAutoRefresh();
</script>
</body>
</html>
`
