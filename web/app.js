// Total Connect Web UI JavaScript

const state = {
  activePane: 'left',
  remotes: [],
  isEditingRemote: false,
  editingRemoteName: '',
  panes: {
    left: {
      remote: 'local',
      path: '.',
      items: [],
      selected: new Set()
    },
    right: {
      remote: 'local',
      path: '.',
      items: [],
      selected: new Set()
    }
  }
};

// Initialize Application
document.addEventListener('DOMContentLoaded', () => {
  initApp();
});

async function initApp() {
  await loadRemotes();
  await loadPane('left');
  await loadPane('right');
  updateUI();
  initWizard();
}

// API Calls
async function apiCall(endpoint, options = {}) {
  try {
    const res = await fetch(endpoint, options);
    const contentType = res.headers.get('content-type');
    let data;
    if (contentType && contentType.includes('application/json')) {
      data = await res.json();
    } else {
      data = await res.text();
    }

    if (!res.ok) {
      const errMsg = (typeof data === 'object' && data.error) ? data.error : (data || res.statusText);
      throw new Error(errMsg);
    }
    return data;
  } catch (err) {
    showToast(err.message, 'error');
    throw err;
  }
}

async function loadRemotes() {
  try {
    const data = await apiCall('api/remotes');
    state.remotes = data.remotes || [];
    populateRemoteSelectors();
  } catch (err) {
    console.error('Failed to load remotes:', err);
  }
}

function populateRemoteSelectors() {
  const leftSelect = document.getElementById('remote-select-left');
  const rightSelect = document.getElementById('remote-select-right');

  const optionsHTML = ['<option value="local">Local Filesystem</option>']
    .concat(state.remotes.map(r => `<option value="${escapeHtml(r)}">${escapeHtml(r)}</option>`))
    .concat(['<option value="__add_connection__">➕ + Add Connection...</option>'])
    .join('');

  if (leftSelect) leftSelect.innerHTML = optionsHTML;
  if (rightSelect) rightSelect.innerHTML = optionsHTML;

  if (leftSelect) leftSelect.value = state.panes.left.remote;
  if (rightSelect) rightSelect.value = state.panes.right.remote;

  renderRemoteModalList();
}

function renderRemoteModalList() {
  const container = document.getElementById('remote-list-modal-body');
  if (!container) return;

  let html = `
    <div class="remote-item-row" style="display:flex; align-items:center; justify-content:space-between; padding:8px; border-bottom:1px solid rgba(255,255,255,0.1);">
      <span style="font-weight:bold; cursor:pointer;" onclick="selectRemoteFromModal('local')">📁 Local Filesystem</span>
      <button class="action-btn primary" style="padding:4px 8px; font-size:0.8rem;" onclick="selectRemoteFromModal('local')">Select</button>
    </div>
  `;

  if (state.remotes.length === 0) {
    html += `<div style="padding:12px; color: var(--text-muted); text-align:center;">No cloud remotes configured yet.</div>`;
  } else {
    state.remotes.forEach(remote => {
      const cleanName = remote.endsWith(':') ? remote.slice(0, -1) : remote;
      html += `
        <div class="remote-item-row" style="display:flex; align-items:center; justify-content:space-between; padding:8px; border-bottom:1px solid rgba(255,255,255,0.1);">
          <span style="font-weight:bold; cursor:pointer;" onclick="selectRemoteFromModal('${escapeJs(remote)}')">☁️ ${escapeHtml(cleanName)}</span>
          <div style="display:flex; gap:6px;">
            <button class="action-btn primary" style="padding:4px 8px; font-size:0.8rem;" onclick="selectRemoteFromModal('${escapeJs(remote)}')">Select</button>
            <button class="action-btn secondary" style="padding:4px 8px; font-size:0.8rem;" onclick="editRemote('${escapeJs(cleanName)}')">✏️ Edit</button>
            <button class="action-btn danger" style="padding:4px 8px; font-size:0.8rem;" onclick="deleteRemote('${escapeJs(cleanName)}')">🗑️ Delete</button>
          </div>
        </div>
      `;
    });
  }

  container.innerHTML = html;
}

function selectRemoteFromModal(remoteVal) {
  onRemoteChange(state.activePane, remoteVal);
  closeModal('remote-modal');
}

async function deleteRemote(remoteName) {
  const cleanName = remoteName.endsWith(':') ? remoteName.slice(0, -1) : remoteName;
  if (!confirm(`Are you sure you want to delete remote [${cleanName}]?`)) {
    return;
  }

  try {
    await apiCall('api/remotes/delete', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ name: cleanName })
    });

    showToast(`Remote [${cleanName}] deleted`, 'success');

    // Reset panes using this remote back to Local Filesystem
    ['left', 'right'].forEach(paneId => {
      const current = state.panes[paneId].remote;
      const currentClean = current.endsWith(':') ? current.slice(0, -1) : current;
      if (currentClean === cleanName) {
        onRemoteChange(paneId, 'local');
      }
    });

    await loadRemotes();
  } catch (err) {
    showToast(`Failed to delete remote: ${err.message}`, 'error');
  }
}

async function editRemote(remoteName) {
  const cleanName = remoteName.endsWith(':') ? remoteName.slice(0, -1) : remoteName;
  try {
    const data = await apiCall(`api/remotes/config?name=${encodeURIComponent(cleanName)}`);
    closeModal('remote-modal');
    openWizardModalForEdit(cleanName, data.type, data.parameters || {});
  } catch (err) {
    showToast(`Failed to fetch config for ${cleanName}: ${err.message}`, 'error');
  }
}

function onRemoteSelectChange(paneId, val) {
  if (val === '__add_connection__') {
    populateRemoteSelectors();
    openWizardModal();
    return;
  }
  onRemoteChange(paneId, val);
}

async function loadPane(paneId) {
  const pane = state.panes[paneId];
  pane.selected.clear();
  updateSelectAllCheckbox(paneId);

  const query = new URLSearchParams({
    remote: pane.remote,
    path: pane.path
  });

  try {
    const items = await apiCall(`api/entries?${query.toString()}`);
    pane.items = sortEntries(items || []);
    renderPaneList(paneId);
  } catch (err) {
    pane.items = [];
    renderPaneList(paneId);
  }
}

function sortEntries(items) {
  const dirs = items.filter(i => i.is_dir && i.name !== '..');
  const files = items.filter(i => !i.is_dir);

  dirs.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }));
  files.sort((a, b) => a.name.localeCompare(b.name, undefined, { sensitivity: 'base' }));

  return [...dirs, ...files];
}

// Rendering
function renderPaneList(paneId) {
  const pane = state.panes[paneId];
  const container = document.getElementById(`file-list-${paneId}`);
  const pathInput = document.getElementById(`path-input-${paneId}`);
  const footer = document.getElementById(`pane-footer-${paneId}`);

  if (pathInput) {
    pathInput.value = pane.path;
  }

  if (!container) return;
  container.innerHTML = '';

  let html = '';

  if (canNavigateUp(pane.path)) {
    html += `
      <div class="file-row" onclick="onRowClick('${paneId}', '..', true, event)">
        <span class="col-check"></span>
        <span class="col-name">
          <span class="item-icon">📁</span>
          <span class="item-name is-parent">..</span>
        </span>
        <span class="col-size">&lt;UP&gt;</span>
        <span class="col-mtime"></span>
      </div>
    `;
  }

  if (pane.items.length === 0) {
    html += `<div class="file-row" style="color: var(--text-muted); justify-content: center;">(Empty Directory)</div>`;
  } else {
    pane.items.forEach((item, index) => {
      const isSelected = pane.selected.has(item.name);
      const icon = item.is_dir ? '📁' : '📄';
      const sizeStr = item.is_dir ? '&lt;DIR&gt;' : formatSize(item.size);
      const mtimeStr = item.mod_time ? formatDate(item.mod_time) : '';

      html += `
        <div class="file-row ${isSelected ? 'selected' : ''}" onclick="onRowClick('${paneId}', '${escapeJs(item.name)}', ${item.is_dir}, event)">
          <span class="col-check" onclick="event.stopPropagation()">
            <input type="checkbox" ${isSelected ? 'checked' : ''} onchange="toggleItemSelect('${paneId}', '${escapeJs(item.name)}', this.checked)">
          </span>
          <span class="col-name">
            <span class="item-icon">${icon}</span>
            <span class="item-name ${item.is_dir ? 'is-dir' : ''}">${escapeHtml(item.name)}</span>
          </span>
          <span class="col-size">${sizeStr}</span>
          <span class="col-mtime">${mtimeStr}</span>
        </div>
      `;
    });
  }

  container.innerHTML = html;

  if (footer) {
    const selCount = pane.selected.size;
    footer.innerText = `${pane.items.length} items ${selCount > 0 ? `(${selCount} selected)` : ''}`;
  }
}

// UI Interaction Handlers
function setActivePane(paneId) {
  state.activePane = paneId;
  updateUI();
}

function updateUI() {
  const leftPane = document.getElementById('pane-left');
  const rightPane = document.getElementById('pane-right');
  const leftTab = document.getElementById('tab-left');
  const rightTab = document.getElementById('tab-right');

  if (leftPane && rightPane) {
    if (state.activePane === 'left') {
      leftPane.classList.add('active');
      rightPane.classList.remove('active');
    } else {
      rightPane.classList.add('active');
      leftPane.classList.remove('active');
    }
  }

  if (leftPane && rightPane) {
    if (state.activePane === 'left') {
      leftPane.classList.add('mobile-visible');
      rightPane.classList.remove('mobile-visible');
    } else {
      rightPane.classList.add('mobile-visible');
      leftPane.classList.remove('mobile-visible');
    }
  }

  if (leftTab && rightTab) {
    if (state.activePane === 'left') {
      leftTab.classList.add('active');
      rightTab.classList.remove('active');
    } else {
      rightTab.classList.add('active');
      leftTab.classList.remove('active');
    }
  }
}

function switchMobileTab(paneId) {
  setActivePane(paneId);
}

function onRemoteChange(paneId, remoteVal) {
  state.panes[paneId].remote = remoteVal;
  state.panes[paneId].path = '.';
  loadPane(paneId);
}

function navigateParent(paneId) {
  const pane = state.panes[paneId];
  if (canNavigateUp(pane.path)) {
    pane.path = getParentPath(pane.path);
    loadPane(paneId);
  }
}

function onRowClick(paneId, name, isDir, event) {
  setActivePane(paneId);

  if (name === '..') {
    navigateParent(paneId);
    return;
  }

  if (isDir) {
    const pane = state.panes[paneId];
    pane.path = joinPath(pane.path, name);
    loadPane(paneId);
  } else {
    toggleItemSelect(paneId, name);
  }
}

function toggleItemSelect(paneId, name, forcedState) {
  const pane = state.panes[paneId];
  if (forcedState !== undefined) {
    if (forcedState) pane.selected.add(name);
    else pane.selected.delete(name);
  } else {
    if (pane.selected.has(name)) pane.selected.delete(name);
    else pane.selected.add(name);
  }
  updateSelectAllCheckbox(paneId);
  renderPaneList(paneId);
}

function toggleSelectAll(paneId, checked) {
  const pane = state.panes[paneId];
  pane.selected.clear();
  if (checked) {
    pane.items.forEach(i => pane.selected.add(i.name));
  }
  renderPaneList(paneId);
}

function updateSelectAllCheckbox(paneId) {
  const pane = state.panes[paneId];
  const chk = document.getElementById(`select-all-${paneId}`);
  if (chk) {
    chk.checked = pane.items.length > 0 && pane.selected.size === pane.items.length;
  }
}

// Action Toolbar Handlers
async function handleCopy() {
  const srcPaneId = state.activePane;
  const dstPaneId = srcPaneId === 'left' ? 'right' : 'left';
  const srcPane = state.panes[srcPaneId];
  const dstPane = state.panes[dstPaneId];

  const selectedItems = Array.from(srcPane.selected);
  if (selectedItems.length === 0) {
    showToast('Select items to copy', 'info');
    return;
  }

  showToast(`Copying ${selectedItems.length} item(s)...`, 'info');

  for (const name of selectedItems) {
    const srcPath = joinPath(srcPane.path, name);
    const dstPath = joinPath(dstPane.path, name);

    try {
      await apiCall('api/copy', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          srcRemote: srcPane.remote,
          srcPath: srcPath,
          dstRemote: dstPane.remote,
          dstPath: dstPath
        })
      });
    } catch (err) {
      showToast(`Failed copying ${name}: ${err.message}`, 'error');
      return;
    }
  }

  showToast('Copy complete', 'success');
  loadPane(dstPaneId);
}

async function handleMove() {
  const srcPaneId = state.activePane;
  const dstPaneId = srcPaneId === 'left' ? 'right' : 'left';
  const srcPane = state.panes[srcPaneId];
  const dstPane = state.panes[dstPaneId];

  const selectedItems = Array.from(srcPane.selected);
  if (selectedItems.length === 0) {
    showToast('Select items to move', 'info');
    return;
  }

  showToast(`Moving ${selectedItems.length} item(s)...`, 'info');

  for (const name of selectedItems) {
    const srcPath = joinPath(srcPane.path, name);
    const dstPath = joinPath(dstPane.path, name);

    try {
      await apiCall('api/move', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          srcRemote: srcPane.remote,
          srcPath: srcPath,
          dstRemote: dstPane.remote,
          dstPath: dstPath
        })
      });
    } catch (err) {
      showToast(`Failed moving ${name}: ${err.message}`, 'error');
      return;
    }
  }

  showToast('Move complete', 'success');
  loadPane(srcPaneId);
  loadPane(dstPaneId);
}

async function handleDelete() {
  const paneId = state.activePane;
  const pane = state.panes[paneId];

  const selectedItems = Array.from(pane.selected);
  if (selectedItems.length === 0) {
    showToast('Select items to delete', 'info');
    return;
  }

  if (!confirm(`Are you sure you want to delete ${selectedItems.length} item(s)?`)) {
    return;
  }

  showToast(`Deleting ${selectedItems.length} item(s)...`, 'info');

  for (const name of selectedItems) {
    const itemPath = joinPath(pane.path, name);

    try {
      await apiCall('api/delete', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          remote: pane.remote,
          path: itemPath
        })
      });
    } catch (err) {
      showToast(`Failed deleting ${name}: ${err.message}`, 'error');
      return;
    }
  }

  showToast('Delete complete', 'success');
  loadPane(paneId);
}

function refreshActivePane() {
  loadPane(state.activePane);
  showToast('Refreshed', 'info');
}

// Modal Handlers
function openMkdirModal() {
  const pane = state.panes[state.activePane];
  const info = document.getElementById('mkdir-target-info');
  const input = document.getElementById('mkdir-input');
  if (info) info.innerText = `In (${pane.remote}): ${pane.path}`;
  if (input) input.value = '';
  document.getElementById('mkdir-modal').classList.remove('hidden');
}

async function confirmMkdir() {
  const input = document.getElementById('mkdir-input');
  const dirName = input ? input.value.trim() : '';
  if (!dirName) return;

  const pane = state.panes[state.activePane];
  const targetPath = joinPath(pane.path, dirName);

  try {
    await apiCall('api/mkdir', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        remote: pane.remote,
        path: targetPath
      })
    });
    showToast(`Directory created: ${dirName}`, 'success');
    closeModal('mkdir-modal');
    loadPane(state.activePane);
  } catch (err) {
    showToast(`Failed creating directory: ${err.message}`, 'error');
  }
}

function openChangeRemoteModal() {
  populateRemoteSelectors();
  document.getElementById('remote-modal').classList.remove('hidden');
}

function closeModal(modalId) {
  const modal = document.getElementById(modalId);
  if (modal) modal.classList.add('hidden');
}

// Connection Wizard Logic
function initWizard() {
  const typeSelect = document.getElementById('wizard-type');
  if (typeSelect) {
    onWizardTypeChange(typeSelect.value);
  }
}

function openWizardModal() {
  state.isEditingRemote = false;
  state.editingRemoteName = '';

  const modal = document.getElementById('wizard-modal');
  if (!modal) return;

  const title = document.getElementById('wizard-title');
  if (title) title.innerText = '⚡ Add New Connection';

  const submitBtn = document.getElementById('wizard-submit-btn');
  if (submitBtn) submitBtn.innerText = 'Save & Connect';

  const nameInput = document.getElementById('wizard-name');
  if (nameInput) {
    nameInput.value = '';
    nameInput.disabled = false;
  }

  const typeSelect = document.getElementById('wizard-type');
  if (typeSelect) {
    typeSelect.value = 'ftp';
    typeSelect.disabled = false;
    onWizardTypeChange('ftp');
  }

  modal.classList.remove('hidden');
}

function openWizardModalForEdit(remoteName, remoteType, params) {
  state.isEditingRemote = true;
  state.editingRemoteName = remoteName;

  const modal = document.getElementById('wizard-modal');
  if (!modal) return;

  const title = document.getElementById('wizard-title');
  if (title) title.innerText = `✏️ Edit Connection [${remoteName}]`;

  const submitBtn = document.getElementById('wizard-submit-btn');
  if (submitBtn) submitBtn.innerText = 'Update Connection';

  const nameInput = document.getElementById('wizard-name');
  if (nameInput) {
    nameInput.value = remoteName;
    nameInput.disabled = true;
  }

  const knownTypes = ['ftp', 'sftp', 'webdav', 's3', 'b2', 'drive', 'mega', 'dropbox', 'onedrive'];
  const targetType = knownTypes.includes(remoteType) ? remoteType : 'custom';

  const typeSelect = document.getElementById('wizard-type');
  if (typeSelect) {
    typeSelect.value = targetType;
    typeSelect.disabled = true;
    onWizardTypeChange(targetType, params, remoteType);
  }

  modal.classList.remove('hidden');
}

function onWizardTypeChange(type, initialParams = {}, actualType = '') {
  const container = document.getElementById('wizard-dynamic-fields');
  if (!container) return;

  let fieldsHtml = '';

  switch (type) {
    case 'ftp':
      fieldsHtml = `
        <div class="form-row">
          <div class="form-group flex-2">
            <label for="param-host">Host / Server</label>
            <input type="text" id="param-host" class="modal-input" placeholder="e.g. ftp.example.com" value="${escapeHtml(initialParams.host || '')}" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-port">Port</label>
            <input type="text" id="param-port" class="modal-input" placeholder="21" value="${escapeHtml(initialParams.port || '21')}">
          </div>
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-user">Username</label>
            <input type="text" id="param-user" class="modal-input" placeholder="Username" value="${escapeHtml(initialParams.user || '')}">
          </div>
          <div class="form-group flex-1">
            <label for="param-pass">Password</label>
            <input type="password" id="param-pass" class="modal-input" placeholder="${state.isEditingRemote ? '(Unchanged unless entered)' : 'Password'}" value="">
          </div>
        </div>
        <div class="form-group checkbox-group">
          <label class="checkbox-label">
            <input type="checkbox" id="param-tls" ${initialParams.tls === 'true' ? 'checked' : ''}> Enable Explicit TLS / Explicit FTP over TLS
          </label>
        </div>
      `;
      break;

    case 'sftp':
      fieldsHtml = `
        <div class="form-row">
          <div class="form-group flex-2">
            <label for="param-host">Host / Server</label>
            <input type="text" id="param-host" class="modal-input" placeholder="e.g. sftp.example.com or IP" value="${escapeHtml(initialParams.host || '')}" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-port">Port</label>
            <input type="text" id="param-port" class="modal-input" placeholder="22" value="${escapeHtml(initialParams.port || '22')}">
          </div>
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-user">Username</label>
            <input type="text" id="param-user" class="modal-input" placeholder="Username" value="${escapeHtml(initialParams.user || '')}" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-pass">Password</label>
            <input type="password" id="param-pass" class="modal-input" placeholder="${state.isEditingRemote ? '(Unchanged unless entered)' : 'Password (or leave blank for SSH key)'}" value="">
          </div>
        </div>
        <div class="form-group">
          <label for="param-key_file">SSH Key Path (Optional)</label>
          <input type="text" id="param-key_file" class="modal-input" placeholder="e.g. ~/.ssh/id_rsa" value="${escapeHtml(initialParams.key_file || '')}">
        </div>
        <div class="form-group checkbox-group">
          <label class="checkbox-label">
            <input type="checkbox" id="param-key_use_agent" ${initialParams.key_use_agent === 'true' ? 'checked' : ''}> Use SSH Agent (Requires local ssh-agent socket)
          </label>
        </div>
      `;
      break;

    case 'webdav':
      fieldsHtml = `
        <div class="form-group">
          <label for="param-url">WebDAV Server URL</label>
          <input type="url" id="param-url" class="modal-input" placeholder="https://nextcloud.example.com/remote.php/dav/files/user/" value="${escapeHtml(initialParams.url || '')}" required>
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-user">Username</label>
            <input type="text" id="param-user" class="modal-input" placeholder="Username" value="${escapeHtml(initialParams.user || '')}">
          </div>
          <div class="form-group flex-1">
            <label for="param-pass">Password / App Token</label>
            <input type="password" id="param-pass" class="modal-input" placeholder="${state.isEditingRemote ? '(Unchanged unless entered)' : 'Password or token'}" value="">
          </div>
        </div>
      `;
      break;

    case 's3':
      fieldsHtml = `
        <div class="form-group">
          <label for="param-endpoint">Endpoint (Optional for AWS S3, Required for MinIO / Cloudflare R2)</label>
          <input type="text" id="param-endpoint" class="modal-input" placeholder="e.g. https://<account_id>.r2.cloudflarestorage.com or http://minio:9000" value="${escapeHtml(initialParams.endpoint || '')}">
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-access_key_id">Access Key ID</label>
            <input type="text" id="param-access_key_id" class="modal-input" placeholder="Access Key" value="${escapeHtml(initialParams.access_key_id || '')}" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-secret_access_key">Secret Access Key</label>
            <input type="password" id="param-secret_access_key" class="modal-input" placeholder="${state.isEditingRemote ? '(Unchanged unless entered)' : 'Secret Key'}" value="">
          </div>
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-region">Region</label>
            <input type="text" id="param-region" class="modal-input" placeholder="e.g. us-east-1 or auto" value="${escapeHtml(initialParams.region || '')}">
          </div>
          <div class="form-group flex-1">
            <label for="param-provider">Provider</label>
            <select id="param-provider" class="modal-select">
              <option value="AWS" ${initialParams.provider === 'AWS' ? 'selected' : ''}>Amazon AWS S3</option>
              <option value="Cloudflare" ${initialParams.provider === 'Cloudflare' ? 'selected' : ''}>Cloudflare R2</option>
              <option value="Minio" ${initialParams.provider === 'Minio' ? 'selected' : ''}>MinIO</option>
              <option value="Other" ${initialParams.provider === 'Other' ? 'selected' : ''}>Other S3 Compatible</option>
            </select>
          </div>
        </div>
      `;
      break;

    case 'b2':
      fieldsHtml = `
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-account">Application Key ID / Account</label>
            <input type="text" id="param-account" class="modal-input" placeholder="Key ID" value="${escapeHtml(initialParams.account || '')}" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-key">Application Key</label>
            <input type="password" id="param-key" class="modal-input" placeholder="${state.isEditingRemote ? '(Unchanged unless entered)' : 'Application Key'}" value="">
          </div>
        </div>
      `;
      break;

    case 'drive':
      fieldsHtml = `
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-client_id">Client ID (Optional)</label>
            <input type="text" id="param-client_id" class="modal-input" placeholder="OAuth Client ID" value="${escapeHtml(initialParams.client_id || '')}">
          </div>
          <div class="form-group flex-1">
            <label for="param-client_secret">Client Secret (Optional)</label>
            <input type="password" id="param-client_secret" class="modal-input" placeholder="OAuth Client Secret" value="">
          </div>
        </div>
        <div class="form-group">
          <label for="param-scope">Scope (Optional)</label>
          <input type="text" id="param-scope" class="modal-input" placeholder="e.g. drive or drive.file" value="${escapeHtml(initialParams.scope || '')}">
        </div>
        <div class="form-group">
          <label for="param-root_folder_id">Root Folder ID (Optional)</label>
          <input type="text" id="param-root_folder_id" class="modal-input" placeholder="Google Drive Root Folder ID" value="${escapeHtml(initialParams.root_folder_id || '')}">
        </div>
      `;
      break;

    case 'mega':
      fieldsHtml = `
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-user">MEGA Email / User</label>
            <input type="email" id="param-user" class="modal-input" placeholder="user@example.com" value="${escapeHtml(initialParams.user || '')}" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-pass">MEGA Password</label>
            <input type="password" id="param-pass" class="modal-input" placeholder="${state.isEditingRemote ? '(Unchanged unless entered)' : 'Password'}" value="">
          </div>
        </div>
      `;
      break;

    case 'dropbox':
      fieldsHtml = `
        <div class="form-group">
          <label for="param-token">Access Token / Token (Required unless App Key provided)</label>
          <input type="password" id="param-token" class="modal-input" placeholder="${state.isEditingRemote ? '(Unchanged unless entered)' : 'Dropbox Token'}" value="">
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-app_key">App Key (Optional)</label>
            <input type="text" id="param-app_key" class="modal-input" placeholder="App Key" value="${escapeHtml(initialParams.app_key || '')}">
          </div>
          <div class="form-group flex-1">
            <label for="param-app_secret">App Secret (Optional)</label>
            <input type="password" id="param-app_secret" class="modal-input" placeholder="App Secret" value="">
          </div>
        </div>
      `;
      break;

    case 'onedrive':
      fieldsHtml = `
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-client_id">Client ID (Optional)</label>
            <input type="text" id="param-client_id" class="modal-input" placeholder="Client ID" value="${escapeHtml(initialParams.client_id || '')}">
          </div>
          <div class="form-group flex-1">
            <label for="param-client_secret">Client Secret (Optional)</label>
            <input type="password" id="param-client_secret" class="modal-input" placeholder="Client Secret" value="">
          </div>
        </div>
        <div class="form-group">
          <label for="param-drive_id">Drive ID (Optional)</label>
          <input type="text" id="param-drive_id" class="modal-input" placeholder="Drive ID" value="${escapeHtml(initialParams.drive_id || '')}">
        </div>
      `;
      break;

    case 'custom':
      let rawConfigText = '';
      if (state.isEditingRemote && actualType) {
        rawConfigText = `type = ${actualType}\n`;
        for (const [k, v] of Object.entries(initialParams)) {
          rawConfigText += `${k} = ${v}\n`;
        }
      }

      fieldsHtml = `
        <div class="form-group">
          <label for="param-custom_type">Storage Provider Type (rclone type)</label>
          <input type="text" id="param-custom_type" class="modal-input" placeholder="e.g. drive, mega, pcloud, qstor, alias, etc." value="${escapeHtml(actualType || initialParams.type || '')}">
        </div>
        <div class="form-group">
          <label for="raw-config-text">⚡ Raw Rclone Config / Key-Value Block</label>
          <textarea id="raw-config-text" class="modal-input" style="height:120px; font-family:monospace; font-size:0.85rem;" placeholder="Paste config block or options, e.g.:&#10;[myremote]&#10;type = drive&#10;scope = drive.readonly&#10;&#10;OR simply key=value pairs:&#10;user = alice&#10;pass = secret">${escapeHtml(rawConfigText)}</textarea>
          <p style="font-size:0.8rem; color: var(--text-muted); margin-top:4px;">
            Supports pasting full standard <code>[remote]\ntype=...\n...</code> blocks OR key=value pairs for any of rclone's 50+ backends.
          </p>
        </div>
      `;
      break;

    default:
      fieldsHtml = '';
  }

  container.innerHTML = fieldsHtml;
}

async function handleWizardSubmit(event) {
  event.preventDefault();

  const nameInput = document.getElementById('wizard-name');
  const typeSelect = document.getElementById('wizard-type');

  let name = nameInput ? nameInput.value.trim() : '';
  let type = typeSelect ? typeSelect.value : '';

  if (state.isEditingRemote) {
    name = state.editingRemoteName;
  }

  if (!name) {
    showToast('Connection name is required', 'error');
    return;
  }

  let parameters = {};

  if (type === 'custom') {
    const customTypeInput = document.getElementById('param-custom_type');
    const rawConfigTextArea = document.getElementById('raw-config-text');

    const customTypeVal = customTypeInput ? customTypeInput.value.trim() : '';
    const rawText = rawConfigTextArea ? rawConfigTextArea.value.trim() : '';

    // Parse raw rclone config text
    const parsed = parseRawRcloneConfig(rawText);

    if (parsed.remoteName && !state.isEditingRemote && !name) {
      name = parsed.remoteName;
    }

    if (parsed.type) {
      type = parsed.type;
    } else if (customTypeVal) {
      type = customTypeVal;
    }

    parameters = parsed.parameters;
  } else {
    // Gather parameters from dynamic input elements
    const container = document.getElementById('wizard-dynamic-fields');
    if (container) {
      const inputs = container.querySelectorAll('input, select');
      inputs.forEach(input => {
        const fieldName = input.id.replace('param-', '');
        if (input.type === 'checkbox') {
          parameters[fieldName] = input.checked ? 'true' : 'false';
        } else {
          const val = input.value.trim();
          if (val !== '') {
            parameters[fieldName] = val;
          }
        }
      });
    }
  }

  if (!type) {
    showToast('Storage type is required', 'error');
    return;
  }

  try {
    showToast(state.isEditingRemote ? 'Updating connection...' : 'Creating connection...', 'info');
    await apiCall('api/remotes/create', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: name,
        type: type,
        parameters: parameters
      })
    });

    closeModal('wizard-modal');
    showToast(`Connection "${name}" ${state.isEditingRemote ? 'updated' : 'created'} successfully!`, 'success');

    // Refresh remote lists
    await loadRemotes();

    // Format target remote name with trailing colon
    const targetRemote = name.endsWith(':') ? name : `${name}:`;

    // Immediately navigate active pane to remote
    onRemoteChange(state.activePane, targetRemote);
  } catch (err) {
    showToast(`Failed to save connection: ${err.message}`, 'error');
  }
}

function parseRawRcloneConfig(text) {
  let remoteName = '';
  let type = '';
  const parameters = {};

  if (!text) return { remoteName, type, parameters };

  const lines = text.split('\n');
  for (let line of lines) {
    line = line.trim();
    if (!line || line.startsWith('#') || line.startsWith(';')) continue;

    // Check header [remote_name]
    if (line.startsWith('[') && line.endsWith(']')) {
      remoteName = line.slice(1, -1).trim();
      continue;
    }

    const eqIdx = line.indexOf('=');
    if (eqIdx !== -1) {
      const key = line.slice(0, eqIdx).trim();
      const val = line.slice(eqIdx + 1).trim();

      if (key.toLowerCase() === 'type') {
        type = val;
      } else {
        parameters[key] = val;
      }
    }
  }

  return { remoteName, type, parameters };
}

// Helpers
function showToast(message, type = 'info') {
  const container = document.getElementById('toast-container');
  if (!container) return;

  const toast = document.createElement('div');
  toast.className = `toast ${type}`;
  toast.innerText = message;

  container.appendChild(toast);

  setTimeout(() => {
    toast.style.opacity = '0';
    toast.style.transition = 'opacity 0.3s';
    setTimeout(() => toast.remove(), 300);
  }, 3000);
}

function canNavigateUp(currentPath) {
  if (!currentPath || currentPath === '.' || currentPath === '/') return false;
  return true;
}

function getParentPath(currentPath) {
  if (!currentPath || currentPath === '.') return '.';
  const parts = currentPath.split('/').filter(Boolean);
  if (parts.length <= 1) return '.';
  parts.pop();
  return parts.join('/');
}

function joinPath(base, child) {
  if (!base || base === '.') return child;
  if (base.endsWith('/')) return base + child;
  return `${base}/${child}`;
}

function formatSize(bytes) {
  if (bytes === 0) return '0 B';
  const k = 1024;
  const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
  const i = Math.floor(Math.log(bytes) / Math.log(k));
  return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
}

function formatDate(isoStr) {
  const date = new Date(isoStr);
  if (isNaN(date.getTime())) return '';
  return date.toISOString().slice(0, 16).replace('T', ' ');
}

function escapeHtml(str) {
  return String(str)
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

function escapeJs(str) {
  return String(str).replace(/\\/g, '\\\\').replace(/'/g, "\\'");
}
