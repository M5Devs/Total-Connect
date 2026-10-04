// Total Connect Web UI JavaScript

const state = {
  activePane: 'left',
  remotes: [],
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
  const modalSelect = document.getElementById('modal-remote-select');

  const optionsHTML = ['<option value="local">Local Filesystem</option>']
    .concat(state.remotes.map(r => `<option value="${escapeHtml(r)}">${escapeHtml(r)}</option>`))
    .concat(['<option value="__add_connection__">➕ + Add Connection...</option>'])
    .join('');

  if (leftSelect) leftSelect.innerHTML = optionsHTML;
  if (rightSelect) rightSelect.innerHTML = optionsHTML;

  const modalOptionsHTML = ['<option value="local">Local Filesystem</option>']
    .concat(state.remotes.map(r => `<option value="${escapeHtml(r)}">${escapeHtml(r)}</option>`))
    .join('');

  if (modalSelect) modalSelect.innerHTML = modalOptionsHTML;

  if (leftSelect) leftSelect.value = state.panes.left.remote;
  if (rightSelect) rightSelect.value = state.panes.right.remote;
}

function onRemoteSelectChange(paneId, val) {
  if (val === '__add_connection__') {
    // Reset selection to current remote value in select box
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
    // Sort items: folders first (alphabetical), then files (alphabetical)
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

  // Can navigate up?
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

  // Mobile visibility
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

function confirmRemoteChange() {
  const select = document.getElementById('modal-remote-select');
  if (select) {
    onRemoteChange(state.activePane, select.value);
  }
  closeModal('remote-modal');
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
  const modal = document.getElementById('wizard-modal');
  if (!modal) return;

  const nameInput = document.getElementById('wizard-name');
  if (nameInput) nameInput.value = '';

  const typeSelect = document.getElementById('wizard-type');
  if (typeSelect) {
    typeSelect.value = 'ftp';
    onWizardTypeChange('ftp');
  }

  modal.classList.remove('hidden');
}

function onWizardTypeChange(type) {
  const container = document.getElementById('wizard-dynamic-fields');
  if (!container) return;

  let fieldsHtml = '';

  switch (type) {
    case 'ftp':
      fieldsHtml = `
        <div class="form-row">
          <div class="form-group flex-2">
            <label for="param-host">Host / Server</label>
            <input type="text" id="param-host" class="modal-input" placeholder="e.g. ftp.example.com" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-port">Port</label>
            <input type="text" id="param-port" class="modal-input" placeholder="21" value="21">
          </div>
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-user">Username</label>
            <input type="text" id="param-user" class="modal-input" placeholder="Username">
          </div>
          <div class="form-group flex-1">
            <label for="param-pass">Password</label>
            <input type="password" id="param-pass" class="modal-input" placeholder="Password">
          </div>
        </div>
        <div class="form-group checkbox-group">
          <label class="checkbox-label">
            <input type="checkbox" id="param-tls"> Enable Explicit TLS / Explicit FTP over TLS
          </label>
        </div>
      `;
      break;

    case 'sftp':
      fieldsHtml = `
        <div class="form-row">
          <div class="form-group flex-2">
            <label for="param-host">Host / Server</label>
            <input type="text" id="param-host" class="modal-input" placeholder="e.g. sftp.example.com or IP" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-port">Port</label>
            <input type="text" id="param-port" class="modal-input" placeholder="22" value="22">
          </div>
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-user">Username</label>
            <input type="text" id="param-user" class="modal-input" placeholder="Username" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-pass">Password</label>
            <input type="password" id="param-pass" class="modal-input" placeholder="Password (or leave blank for SSH key)">
          </div>
        </div>
        <div class="form-group">
          <label for="param-key_file">SSH Key Path (Optional)</label>
          <input type="text" id="param-key_file" class="modal-input" placeholder="e.g. ~/.ssh/id_rsa">
        </div>
      `;
      break;

    case 'webdav':
      fieldsHtml = `
        <div class="form-group">
          <label for="param-url">WebDAV Server URL</label>
          <input type="url" id="param-url" class="modal-input" placeholder="https://nextcloud.example.com/remote.php/dav/files/user/" required>
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-user">Username</label>
            <input type="text" id="param-user" class="modal-input" placeholder="Username">
          </div>
          <div class="form-group flex-1">
            <label for="param-pass">Password / App Token</label>
            <input type="password" id="param-pass" class="modal-input" placeholder="Password or token">
          </div>
        </div>
      `;
      break;

    case 's3':
      fieldsHtml = `
        <div class="form-group">
          <label for="param-endpoint">Endpoint (Optional for AWS S3, Required for MinIO)</label>
          <input type="text" id="param-endpoint" class="modal-input" placeholder="e.g. https://s3.amazonaws.com or http://minio:9000">
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-access_key_id">Access Key ID</label>
            <input type="text" id="param-access_key_id" class="modal-input" placeholder="Access Key" required>
          </div>
          <div class="form-group flex-1">
            <label for="param-secret_access_key">Secret Access Key</label>
            <input type="password" id="param-secret_access_key" class="modal-input" placeholder="Secret Key" required>
          </div>
        </div>
        <div class="form-row">
          <div class="form-group flex-1">
            <label for="param-region">Region</label>
            <input type="text" id="param-region" class="modal-input" placeholder="e.g. us-east-1">
          </div>
          <div class="form-group flex-1">
            <label for="param-provider">Provider</label>
            <select id="param-provider" class="modal-select">
              <option value="AWS">Amazon AWS S3</option>
              <option value="Minio">MinIO</option>
              <option value="Other">Other S3 Compatible</option>
            </select>
          </div>
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

  const name = nameInput ? nameInput.value.trim() : '';
  const type = typeSelect ? typeSelect.value : '';

  if (!name || !type) {
    showToast('Name and storage type are required', 'error');
    return;
  }

  const parameters = {};

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

  try {
    showToast('Creating connection...', 'info');
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
    showToast(`Connection "${name}" created successfully!`, 'success');

    // Refresh remote lists
    await loadRemotes();

    // Format target remote name with trailing colon if needed
    const createdRemote = name.endsWith(':') ? name : `${name}:`;

    // Immediately navigate active pane to new remote
    onRemoteChange(state.activePane, createdRemote);
  } catch (err) {
    showToast(`Failed to add connection: ${err.message}`, 'error');
  }
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
