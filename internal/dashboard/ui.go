package dashboard

// dashboardHTML is an isolated, dependency-free and read-only manager view.
// Every value loaded from the journal must be escaped before HTML rendering.
const dashboardHTML = `<!doctype html>
<html lang="pt-BR">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<meta name="color-scheme" content="light">
<title>Evydência | Monitor de mensagens</title>
<style>
:root {
  font-family: system-ui,-apple-system,"Segoe UI",Roboto,Arial,sans-serif;
  color-scheme:light;
  --bg:#f6f6f4; --surface:#fff; --ink:#172522; --muted:#576760;
  --line:#e2e8e3; --accent:#176b50; --soft:#eaf5ef;
  --warn:#915309; --warnbg:#fff3df; --danger:#a33135; --dangerbg:#fff0ef;
  --shadow:0 3px 18px rgba(21,43,34,.045);
}
*{box-sizing:border-box}
body{margin:0;background:var(--bg);color:var(--ink);font-size:14px;line-height:1.48}
button,input,select{font:inherit}
button,select{cursor:pointer}
button:focus-visible,input:focus-visible,select:focus-visible,summary:focus-visible{outline:3px solid #76baa0;outline-offset:3px}
.shell{max-width:1460px;padding:26px clamp(14px,3vw,42px) 40px;margin:auto}
.top{display:flex;justify-content:space-between;align-items:center;gap:20px;margin-bottom:25px}
.eyebrow{color:var(--accent);font-size:11px;text-transform:uppercase;letter-spacing:.12em;font-weight:800}
h1{font-size:clamp(25px,3vw,32px);line-height:1.16;letter-spacing:-.035em;margin:5px 0 7px}
h2{font-size:19px;letter-spacing:-.02em;margin:0}
.description{color:var(--muted);margin:0;max-width:700px}
.top-actions{display:flex;align-items:center;gap:9px;flex-wrap:wrap}
.select,.search{background:#fff;border:1px solid #ccd8d1;border-radius:10px;padding:10px 12px;color:var(--ink);min-height:42px}
.btn{border:1px solid var(--accent);background:var(--accent);color:#fff;font-weight:700;border-radius:10px;padding:10px 17px;min-height:42px}
.btn:hover{background:#11543e}
.btn[disabled]{opacity:.68;cursor:wait}
.statusline{display:flex;align-items:center;justify-content:space-between;flex-wrap:wrap;gap:8px;padding:12px 16px;background:#edf5ee;border:1px solid #d7e6da;border-radius:12px;color:#325d44;margin-bottom:18px}
.statusline.error{background:var(--warnbg);border-color:#ebcf9f;color:var(--warn)}
.statusline .minor{color:var(--muted);font-size:12px}
.metrics{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:12px;margin-bottom:24px}
.metric{border-radius:14px;background:var(--surface);border:1px solid var(--line);padding:17px 18px;box-shadow:var(--shadow)}
.metric .value{font-size:32px;font-weight:800;line-height:1.1;font-variant-numeric:tabular-nums;display:block;margin:9px 0 5px}
.metric .heading{font-weight:650;color:#52635b;font-size:13px}
.metric .detail{font-size:12px;color:var(--muted)}
.metric.attention .value{color:var(--danger)}
.metric.positive .value{color:var(--accent)}
.section{background:var(--surface);border:1px solid var(--line);border-radius:16px;box-shadow:var(--shadow);margin-bottom:16px;overflow:hidden}
.section-head{display:flex;justify-content:space-between;align-items:center;gap:12px;flex-wrap:wrap;padding:21px 21px 14px}
.section-head .muted{color:var(--muted);font-size:12px;margin:4px 0 0}
.tools{display:grid;grid-template-columns:1.6fr 1fr 1fr auto;gap:9px;padding:0 21px 18px}
.search{min-width:0;width:100%}
.tools .select{width:100%;min-width:0}
.results{font-size:12px;color:var(--muted);padding:9px 21px;border-top:1px solid var(--line);background:#fafcfb}
.scroll{overflow-x:auto}
.data-table{width:100%;border-collapse:collapse;table-layout:fixed}
.data-table th{text-align:left;padding:12px 15px;color:#64726a;background:#f9fbf9;font-size:11px;font-weight:800;letter-spacing:.045em;text-transform:uppercase;white-space:nowrap;border-bottom:1px solid var(--line)}
.data-table td{vertical-align:top;padding:17px 15px;border-bottom:1px solid #edf0ed;overflow-wrap:anywhere}
.data-table tr:last-child td{border-bottom:0}
.data-table tbody tr:hover{background:#fbfcfb}
.cell-main{font-weight:750}
.sub{font-size:12px;color:var(--muted);margin-top:4px}
.date{font-weight:750;font-variant-numeric:tabular-nums}
.pill{font-weight:750;font-size:11px;border-radius:30px;padding:5px 9px;display:inline-flex;align-items:center;line-height:1.3}
.pill.ads{background:#e3f3ec;color:#165b42}
.pill.normal{background:#edf0f4;color:#536275}
.pill.success{background:#e3f4e8;color:#14603f}
.pill.danger{background:#ffedeb;color:#9c2b2d}
.pill.warning{background:#fff1d9;color:#875313}
.pill.neutral{background:#eef1ef;color:#526258}
.plain-note{color:var(--muted);font-size:12px}
.message{font-size:13px;overflow-wrap:anywhere;max-width:420px}
.trigger{font-size:12px;color:var(--accent);font-weight:650;margin-top:5px}
.steps{margin:8px 0 0}
.steps summary{color:var(--accent);font-weight:750;cursor:pointer;font-size:12px;list-style-position:inside}
.steps[open]{background:#f8fbf8;border:1px solid var(--line);border-radius:10px;padding:9px}
.timeline{list-style:none;margin:11px 0 0;padding:0}
.timeline li{border-left:2px solid #cbdcd0;margin-left:6px;padding:0 0 16px 14px;position:relative;font-size:12px;line-height:1.5}
.timeline li:last-child{padding-bottom:2px}
.timeline li:before{content:"";width:9px;height:9px;border-radius:50%;background:var(--accent);position:absolute;left:-6px;top:3px}
.timeline strong{display:block;font-size:13px}
.timeline .resp{font-weight:650}
.hint{padding:13px 20px;color:var(--muted);font-size:12px;border-top:1px solid var(--line);background:#fbfcfb}
.secondary-panel{padding:0 19px}
.secondary-panel>summary{padding:19px 2px;font-size:15px;font-weight:750;cursor:pointer}
.secondary-body{border-top:1px solid var(--line);padding:17px 0}
.contact-table{width:100%;border-collapse:collapse}
.contact-table th,.contact-table td{text-align:left;border-bottom:1px solid var(--line);padding:11px 8px;font-size:12px}
.contact-table th{color:var(--muted)}
#contactSearch{max-width:320px;margin-bottom:12px}
.empty{text-align:center;padding:34px 15px;color:var(--muted)}
footer{color:#65746b;font-size:12px;padding:12px 2px 0}
.hide{display:none!important}
@media(max-width:1060px){
 .metrics{grid-template-columns:repeat(2,minmax(0,1fr))}
 .tools{grid-template-columns:1fr 1fr}
 .tools .search{grid-column:1/-1}
}
@media(max-width:700px){
 .shell{padding:14px 10px 25px}
 .top{display:block;padding:8px 5px 0}
 .description{margin-bottom:14px}
 .top-actions{justify-content:space-between}
 .top-actions select{flex:1}
 .metrics{gap:8px;margin-bottom:14px}
 .metric{padding:14px 13px;border-radius:12px}
 .metric .value{font-size:27px}
 .metric .heading{font-size:12px}
 .metric .detail{font-size:11px}
 .section{border-radius:13px}
 .section-head{padding:16px 15px 13px}
 .tools{padding:0 14px 13px;gap:8px;grid-template-columns:1fr 1fr}
 .tools .search{grid-column:1/-1}
 .tools .select{font-size:12px}
 .results{padding:8px 14px}
 .scroll{overflow:visible}
 .data-table,.data-table tbody,.data-table tr,.data-table td{display:block;width:100%}
 .data-table thead{display:none}
 .data-table tbody tr{padding:13px 14px;border-bottom:9px solid var(--bg)}
 .data-table tbody tr:last-child{border-bottom:0}
 .data-table td{border:0;padding:3px 0}
 .data-table td:before{content:attr(data-label);font-size:11px;display:block;color:#6b786e;font-weight:650;margin-bottom:3px}
 .data-table td[data-label="Contato"]:before{display:none}
 .data-table td[data-label="Resultado"]{padding-top:8px}
 .message{max-width:none}
 .steps[open]{padding:8px}
 .contact-table{display:block;overflow-x:auto}
 footer{padding:12px}
}
@media(prefers-reduced-motion:reduce){*{scroll-behavior:auto!important;animation:none!important}}
</style>
</head>
<body>
<div class="shell">
  <header class="top">
    <div>
      <div class="eyebrow">Estúdio Evydência · Natal 2026</div>
      <h1>Monitor de mensagens</h1>
      <p class="description">Veja quem entrou pelo WhatsApp, como o bot respondeu e se a proposta chegou ao CRM.</p>
    </div>
    <div class="top-actions">
      <select id="refreshInterval" class="select" aria-label="Atualização automática">
        <option value="0">Atualização manual</option>
        <option value="30" selected>Atualizar a cada 30 s</option>
        <option value="60">Atualizar a cada 1 min</option>
        <option value="120">Atualizar a cada 2 min</option>
      </select>
      <button class="btn" id="refreshBtn" type="button">Atualizar agora</button>
    </div>
  </header>

  <div class="statusline" id="connectionLine" role="status" aria-live="polite">
    <span id="statusText">Carregando histórico…</span>
    <span class="minor" id="updatedAt">Dados registrados pelo bot; não é confirmação de entrega.</span>
  </div>

  <div class="metrics" aria-label="Resumo dos registros carregados">
    <div class="metric"><span class="heading">Mensagens recebidas</span><span class="value" id="kpiReceived">—</span><span class="detail">Na janela consultada</span></div>
    <div class="metric"><span class="heading">Identificadas como anúncio</span><span class="value" id="kpiAds">—</span><span class="detail">Gatilho ou indicação Meta</span></div>
    <div class="metric positive"><span class="heading">Propostas aceitas pelo CRM</span><span class="value" id="kpiAccepted">—</span><span class="detail">Agendadas, não entregues</span></div>
    <div class="metric attention"><span class="heading">Precisam de atenção</span><span class="value" id="kpiAttention">—</span><span class="detail">Rejeitadas, incertas ou bloqueadas</span></div>
  </div>

  <section class="section" aria-labelledby="historyHeading">
    <div class="section-head">
      <div>
        <h2 id="historyHeading">Histórico de atendimentos</h2>
        <p class="muted">Eventos mais recentes primeiro · fuso de Brasília · somente leitura</p>
      </div>
    </div>
    <div class="tools">
      <input id="searchEvents" class="search" type="search" aria-label="Pesquisar histórico" placeholder="Buscar nome, telefone ou mensagem">
      <select id="filterSource" class="select" aria-label="Filtrar origem">
        <option value="all">Todas as origens</option>
        <option value="meta_ads">Tráfego pago identificado</option>
        <option value="normal">Sem marcação de anúncio</option>
      </select>
      <select id="filterStatus" class="select" aria-label="Filtrar situação">
        <option value="all">Todas as situações</option>
        <option value="attention">Precisam de atenção</option>
        <option value="crm_aceito">CRM aceitou</option>
        <option value="crm_rejeitado">CRM rejeitou</option>
        <option value="bloqueado_local">Bot bloqueou antes do CRM</option>
        <option value="crm_nao_registrado">Sem etapa CRM registrada</option>
        <option value="incerto">Resultado incerto</option>
        <option value="pendente">Pendente</option>
        <option value="sem_tarefas">Sem execução registrada</option>
        <option value="telefone_nao_resolvido">Telefone ainda não identificado</option>
        <option value="finalizado_local">Atendimento local</option>
        <option value="ignorado">Ignorados / bloqueados</option>
      </select>
      <select id="pageLimit" class="select" aria-label="Tamanho da janela">
        <option value="50">50 mensagens</option>
        <option value="150" selected>150 mensagens</option>
        <option value="500">500 mensagens</option>
      </select>
    </div>
    <div class="results" id="resultCount">Aguardando registros...</div>
    <div class="scroll">
      <table class="data-table" aria-label="Eventos recebidos pelo bot">
        <colgroup><col style="width:15%"><col style="width:21%"><col style="width:29%"><col style="width:18%"><col style="width:17%"></colgroup>
        <thead><tr><th>Recebido</th><th>Contato</th><th>Mensagem / gatilho</th><th>Resultado</th><th>Histórico de APIs</th></tr></thead>
        <tbody id="eventRows"><tr><td colspan="5" class="empty">Carregando...</td></tr></tbody>
      </table>
    </div>
    <div class="hint">O filtro "Precisam de atenção" também inclui mensagens de anúncio sem encaminhamento após cinco minutos e bloqueios internos do bot. "CRM rejeitou" só aparece quando houve uma recusa HTTP comprovada. "CRM aceitou" significa que a proposta foi agendada. Não significa que o destinatário recebeu ou leu a mensagem. "Sem marcação de anúncio" não comprova origem orgânica. Histórico anterior pode estar incompleto.</div>
  </section>

  <section class="section" aria-label="Resumo de contatos">
    <details class="secondary-panel" id="contactsDetails">
      <summary>Ver contatos agrupados por telefone (visão complementar)</summary>
      <div class="secondary-body">
        <p class="description">Esta visão agrupa várias mensagens do mesmo contato. Os totais podem ser diferentes dos eventos acima.</p>
        <input id="searchContacts" class="search" type="search" placeholder="Buscar contato ou telefone" aria-label="Buscar contatos">
        <div class="scroll">
          <table class="contact-table"><thead><tr><th>Última atividade (BRT)</th><th>Nome</th><th>Telefone</th><th>Origem do fluxo</th><th>CRM</th></tr></thead>
          <tbody id="contactRows"><tr><td colspan="5" class="empty">Abra para carregar os contatos.</td></tr></tbody></table>
        </div>
      </div>
    </details>
  </section>
  <footer>Fonte: journal e outbox do BotWhatsApp · Sem disparos pelo painel · Acesso restrito.</footer>
</div>
<script>
'use strict';
const state = { events:[], contacts:[], contactsLoaded:false, loading:false, timer:null, lastSuccess:null };
const $ = id => document.getElementById(id);
function apiURL(path) { return new URL(path, window.location.origin); }
function safe(value) {
  return String(value == null ? '' : value).replace(/[&<>"']/g, ch =>
    ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[ch]));
}
function badge(label, kind) { return '<span class="pill '+kind+'">'+safe(label)+'</span>'; }
const statusLabels = {
  crm_aceito:['CRM aceitou','success'], crm_rejeitado:['CRM rejeitou','danger'],
  incerto:['Resultado incerto','danger'], pendente:['Pendente','warning'],
  em_processamento:['Em andamento','warning'], sem_tarefas:['Sem execução registrada','neutral'],
  bloqueado_local:['Bloqueado antes do CRM','warning'],
  crm_nao_registrado:['Etapa CRM não registrada','warning'],
  finalizado_local:['Atendimento local finalizado','neutral'], ignorado:['Ignorado','neutral'],
  opt_out:['Contato pediu para parar','warning'],
  telefone_nao_resolvido:['Telefone pendente de identificação','warning'],
  recebido:['Recebido','neutral']
};
function showStatus(msg, error) {
  $('connectionLine').classList.toggle('error', Boolean(error));
  $('statusText').textContent=msg;
  $('updatedAt').textContent=state.lastSuccess ? 'Última leitura: '+state.lastSuccess+' (BRT)' : 'Fonte: registros locais do bot.';
}
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
function stepsHTML(steps, status) {
  if (!Array.isArray(steps) || !steps.length) {
    if (status === 'ignorado') return '<span class="plain-note">Nenhuma etapa necessária (mensagem ignorada)</span>';
    return '<span class="plain-note">Nenhuma etapa de API registrada</span>';
  }
  return '<details class="steps"><summary>Ver '+steps.length+' etapa(s)</summary><ol class="timeline">'+
    steps.map(t => '<li><strong>'+safe(t.step)+'</strong>'+
      '<div>'+safe(t.provider)+' · <span class="resp">'+safe(t.api_response)+'</span></div>'+
      (t.started_brt ? '<div class="sub">Início: '+safe(t.started_brt)+'</div>' : '')+
      (t.completed_brt ? '<div class="sub">Fim: '+safe(t.completed_brt)+'</div>' : '')+
      (t.external_id ? '<div class="sub">Recibo: '+safe(t.external_id)+'</div>' : '')+
      (t.reason ? '<div class="sub">Motivo: '+safe(formatReason(t.reason))+'</div>' : '')+'</li>').join('')+
      '</ol></details>';
}
function renderEvents() {
  const source=$('filterSource').value, filter=$('filterStatus').value;
  const term=$('searchEvents').value.trim().toLocaleLowerCase('pt-BR');
  const found=state.events.filter(e => {
    if (source!=='all' && source!==e.source) return false;
    if (filter==='attention' && !e.needs_attention) return false;
    if (filter==='ignorado' && e.status!=='ignorado' && e.status!=='opt_out') return false;
    if (filter!=='all' && filter!=='attention' && filter!=='ignorado' && filter!==e.status) return false;
    return !term || [e.name,e.phone,e.inbound_text,e.trigger_text,e.crm_receipt,e.decision_reason]
      .some(t => String(t||'').toLocaleLowerCase('pt-BR').includes(term));
  });
  $('resultCount').textContent='Exibindo '+found.length+' de '+state.events.length+' mensagens carregadas (filtros apenas nesta janela)';
  if (!found.length) {
    $('eventRows').innerHTML='<tr><td colspan="5" class="empty">Nenhum registro encontrado nesta janela. Ajuste os filtros ou carregue mais mensagens.</td></tr>';
    return;
  }
  $('eventRows').innerHTML=found.map(e => {
    const label=statusLabels[e.status] || ['Estado não identificado','neutral'];
    const sourceTag=e.source==='meta_ads' ? badge('Indício de anúncio','ads') : badge('Sem marcação de anúncio','normal');
    const nameSource=e.name_origin==='crm'?'Nome confirmado no CRM':
      (e.name_origin==='whatsapp'?'Nome informado pelo WhatsApp':'Nome não disponível');
    const message=safe(e.inbound_text || '(texto não registrado)');
    const trigger=e.trigger_matched ?
      '<div class="trigger">Gatilho reconhecido'+(e.trigger_text ? ': '+safe(e.trigger_text) : '')+'</div>' :
      '<div class="sub">Sem gatilho exato'+(e.has_ad_context ? ' · contexto Meta presente' : '')+'</div>';
    return '<tr>'+
      '<td data-label="Recebido"><div class="date">'+safe(e.received_at_brt || 'Data indisponível')+'</div></td>'+
      '<td data-label="Contato"><div class="cell-main">'+safe(e.name || 'Nome não informado')+'</div>'+
        '<div class="sub">'+safe(nameSource)+'</div>'+
        '<div class="sub">'+safe(e.phone || 'Telefone não identificado')+'</div><div style="margin-top:7px">'+sourceTag+'</div></td>'+
      '<td data-label="Mensagem / gatilho"><div class="message">'+message+'</div>'+
        '<div class="sub">Tipo: '+safe(e.inbound_type || 'Não informado')+'</div>'+trigger+'</td>'+
      '<td data-label="Resultado">'+badge(label[0],label[1])+
        (e.crm_receipt ? '<div class="sub">Despacho: '+safe(e.crm_receipt)+'</div>' : '')+
        (e.decision_reason ? '<div class="sub">'+safe(formatReason(e.decision_reason))+'</div>' : '')+'</td>'+
      '<td data-label="Histórico das APIs">'+stepsHTML(e.steps, e.status)+'</td></tr>';
  }).join('');
}
function renderContacts(){
  const term=$('searchContacts').value.trim().toLocaleLowerCase('pt-BR');
  const found=state.contacts.filter(c=>!term||[c.name,c.phone_raw,c.phone_formatted]
    .some(x=>String(x||'').toLocaleLowerCase('pt-BR').includes(term)));
  $('contactRows').innerHTML=found.length ? found.map(c=>
    '<tr><td>'+safe(c.last_interaction_brt)+'</td>'+
    '<td><strong>'+safe(c.name)+'</strong></td>'+
    '<td>'+safe(c.phone_formatted)+'</td>'+
    '<td>'+safe(c.channel)+'</td>'+
    '<td>'+safe(c.crm_status)+'</td></tr>').join('') :
    '<tr><td colspan="5" class="empty">Nenhum contato encontrado.</td></tr>';
}
async function loadContacts(){
  if (state.contactsLoaded) return;
  try {
    const res=await fetch(apiURL('/api/dashboard/contacts'),{credentials:'same-origin',cache:'no-store'});
    if(!res.ok) throw new Error('Falha HTTP '+res.status);
    const data=await res.json();
    state.contacts=Array.isArray(data.contacts)?data.contacts:[];
    state.contactsLoaded=true;
    renderContacts();
  } catch (_) {
    $('contactRows').innerHTML='<tr><td colspan="5" class="empty">Contatos indisponíveis. Tente fechar e abrir esta seção.</td></tr>';
  }
}
async function refreshData(){
  if(state.loading) return;
  state.loading=true;
  $('refreshBtn').disabled=true;
  showStatus('Consultando histórico do bot...',false);
  try {
    const res=await fetch(apiURL('/api/dashboard/events?limit='+encodeURIComponent($('pageLimit').value)),
      {credentials:'same-origin',cache:'no-store'});
    if(!res.ok) throw new Error('Falha HTTP '+res.status);
    const data=await res.json();
    if(!Array.isArray(data.events) || !data.stats) throw new Error('Formato inesperado');
    state.events=data.events;
    $('kpiReceived').textContent=data.stats.received||0;
    $('kpiAds').textContent=data.stats.meta_ads||0;
    $('kpiAccepted').textContent=data.stats.crm_accepted||0;
    $('kpiAttention').textContent=data.stats.attention == null
      ? (data.stats.crm_rejected||0)+(data.stats.uncertain||0)
      : data.stats.attention;
    state.lastSuccess=new Date().toLocaleString('pt-BR',{timeZone:'America/Sao_Paulo',hour12:false});
    showStatus('Histórico consultado com sucesso.',false);
    renderEvents();
    // Contacts are only read when the manager expands the optional section.
    if($('contactsDetails').open && state.contactsLoaded){
      state.contactsLoaded=false;
      await loadContacts();
    }
  } catch(_){
    showStatus('Não foi possível atualizar. Os dados anteriores continuam na tela.',true);
    if(!state.lastSuccess) $('eventRows').innerHTML='<tr><td colspan="5" class="empty">Histórico indisponível. Verifique o acesso ao bot.</td></tr>';
  } finally {
    state.loading=false;
    $('refreshBtn').disabled=false;
  }
}
function configureTimer(){
  if(state.timer) clearInterval(state.timer);
  const seconds=Number($('refreshInterval').value);
  if(seconds>0) state.timer=setInterval(()=>{
    if(!document.hidden) refreshData();
  },seconds*1000);
}
$('refreshBtn').addEventListener('click',refreshData);
$('refreshInterval').addEventListener('change',configureTimer);
$('pageLimit').addEventListener('change',refreshData);
['searchEvents','filterSource','filterStatus'].forEach(id=>{
  $(id).addEventListener(id==='searchEvents'?'input':'change',renderEvents);
});
$('contactsDetails').addEventListener('toggle',()=>{
  if($('contactsDetails').open) loadContacts();
});
$('searchContacts').addEventListener('input',renderContacts);
configureTimer();
refreshData();
</script>
</body>
</html>`
