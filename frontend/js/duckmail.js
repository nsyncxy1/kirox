// ============================================================
// duckmail.js  —  DuckMail 配置管理（参照 moemail.js 模式）
// ============================================================

// 当前已保存的 DuckMail 配置列表（内存缓存）
window.duckMailConfigs = [];

// ---- 初始化 ----
async function initDuckMail() {
  try {
    const configs = await window.go.main.App.GetDuckMailConfigs();
    window.duckMailConfigs = configs || [];
    renderDuckMailList();
    updateDuckMailSummary();
    renderDuckMailConfigSelect();
  } catch (e) {
    console.error('[DuckMail] 初始化失败', e);
  }
}

// ---- 渲染邮箱池页面的配置列表 ----
function renderDuckMailList() {
  const el = document.getElementById('duckmail-inline-list');
  if (!el) return;
  const cfgs = window.duckMailConfigs || [];
  if (cfgs.length === 0) {
    el.innerHTML = '<div style="text-align:center;color:var(--text-muted);font-size:12px;padding:12px 0;">暂无配置，请添加</div>';
    return;
  }
  el.innerHTML = cfgs.map((c, i) => `
    <div class="moemail-config-item" style="display:flex;align-items:center;gap:8px;padding:8px 0;border-bottom:1px solid var(--border);">
      <span style="flex:0 0 24px;font-size:12px;color:var(--text-muted);text-align:center;">${i + 1}</span>
      <span style="flex:1;font-size:13px;font-weight:600;color:var(--text);">${escHtml(c.name || '未命名')}</span>
      <span style="flex:2;font-size:12px;color:var(--text-secondary);word-break:break-all;">${escHtml(c.apiUrl || '')}</span>
      <span style="flex:1.5;font-size:12px;color:var(--text-secondary);">${escHtml(c.domain || '')}</span>
      <button onclick="deleteDuckMailConfig(${i})" class="btn btn-danger btn-sm" style="flex-shrink:0;padding:3px 10px;font-size:11px;">删除</button>
    </div>
  `).join('');
}

// ---- 渲染注册页面的配置勾选列表 ----
function renderDuckMailConfigSelect() {
  const el = document.getElementById('cfg-duckmail-configs-list');
  if (!el) return;
  const cfgs = window.duckMailConfigs || [];
  if (cfgs.length === 0) {
    el.innerHTML = '<div style="text-align:center;color:var(--text-muted);font-size:12px;padding:12px;">暂无 DuckMail 配置，请先在邮箱池页面添加</div>';
    return;
  }
  el.innerHTML = cfgs.map((c, i) => `
    <label class="moemail-domain-tag" style="cursor:pointer;display:inline-flex;align-items:center;gap:6px;">
      <input type="checkbox" class="duckmail-config-checkbox" data-index="${i}" checked style="accent-color:#f59e0b;">
      <span style="font-size:12px;font-weight:600;">${escHtml(c.name || '未命名')}</span>
      <span style="font-size:11px;color:var(--text-muted);">${escHtml(c.domain || '')}</span>
    </label>
  `).join('');
}

// ---- 全选 ----
function selectAllDuckMailConfigs() {
  document.querySelectorAll('.duckmail-config-checkbox').forEach(cb => cb.checked = true);
}

// ---- 摘要标签 ----
function updateDuckMailSummary() {
  const el = document.getElementById('settings-duckmail-summary');
  if (!el) return;
  const n = (window.duckMailConfigs || []).length;
  el.textContent = n > 0 ? `${n} 个配置` : '未配置';
}

// ---- 添加配置（内联表单） ----
async function inlineAddDuckMail() {
  const name   = (document.getElementById('duckmail-inline-name')?.value   || '').trim();
  const url    = (document.getElementById('duckmail-inline-url')?.value    || '').trim();
  const apiKey = (document.getElementById('duckmail-inline-apikey')?.value || '').trim();
  const domain = (document.getElementById('duckmail-inline-domain')?.value || '').trim();

  if (!url)    { setDuckMailStatus('请填写 API URL', 'error');  return; }
  if (!domain) { setDuckMailStatus('请填写域名',     'error');  return; }

  const cfg = {
    name:   name || `配置${(window.duckMailConfigs.length + 1)}`,
    apiUrl: url,
    apiKey: apiKey,
    domain: domain,
  };

  window.duckMailConfigs.push(cfg);
  await saveDuckMailConfigs();

  // 清空表单
  ['duckmail-inline-name','duckmail-inline-url','duckmail-inline-apikey','duckmail-inline-domain']
    .forEach(id => { const el = document.getElementById(id); if (el) el.value = ''; });

  setDuckMailStatus('✓ 添加成功', 'success');
  setTimeout(() => setDuckMailStatus('', ''), 2000);
}

// ---- 测试连接 ----
async function inlineTestDuckMail() {
  const url    = (document.getElementById('duckmail-inline-url')?.value    || '').trim();
  const apiKey = (document.getElementById('duckmail-inline-apikey')?.value || '').trim();
  const domain = (document.getElementById('duckmail-inline-domain')?.value || '').trim();
  const name   = (document.getElementById('duckmail-inline-name')?.value   || '').trim() || '测试';

  if (!url)    { setDuckMailStatus('请填写 API URL', 'error');  return; }
  if (!domain) { setDuckMailStatus('请填写域名',     'error');  return; }

  const btn = document.getElementById('duckmail-inline-test-btn');
  if (btn) { btn.disabled = true; btn.textContent = '测试中...'; }
  setDuckMailStatus('', '');

  try {
    const cfg = { name, apiUrl: url, apiKey, domain };
    const result = await window.go.main.App.TestDuckMailConnection(JSON.stringify(cfg));
    if (result.error) {
      setDuckMailStatus('✗ ' + result.error, 'error');
    } else {
      setDuckMailStatus(`✓ 连接成功，域名: ${(result.domains || [domain]).join(', ')}`, 'success');
    }
  } catch (e) {
    setDuckMailStatus('✗ ' + String(e), 'error');
  } finally {
    if (btn) { btn.disabled = false; btn.textContent = '测试连接'; }
  }
}

// ---- 删除单条配置 ----
async function deleteDuckMailConfig(index) {
  window.duckMailConfigs.splice(index, 1);
  await saveDuckMailConfigs();
}

// ---- 持久化 ----
async function saveDuckMailConfigs() {
  try {
    const result = await window.go.main.App.SaveDuckMailConfigs(JSON.stringify(window.duckMailConfigs));
    if (result.error) {
      console.error('[DuckMail] 保存失败:', result.error);
    }
  } catch (e) {
    console.error('[DuckMail] 保存异常:', e);
  }
  renderDuckMailList();
  updateDuckMailSummary();
  renderDuckMailConfigSelect();
}

// ---- 状态文字 ----
function setDuckMailStatus(msg, type) {
  const el = document.getElementById('duckmail-inline-status');
  if (!el) return;
  el.textContent = msg;
  el.style.color = type === 'error' ? 'var(--danger)' : type === 'success' ? '#10b981' : 'var(--text-muted)';
}

// ---- 获取选中的 DuckMail 配置列表（供 task.js 使用） ----
function getSelectedDuckMailConfigs() {
  const cfgs = window.duckMailConfigs || [];
  const checked = [];
  document.querySelectorAll('.duckmail-config-checkbox:checked').forEach(cb => {
    const idx = parseInt(cb.dataset.index, 10);
    if (!isNaN(idx) && cfgs[idx]) {
      checked.push(cfgs[idx]);
    }
  });
  return checked;
}

// ---- 工具 ----
function escHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}
