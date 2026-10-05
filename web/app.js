// Total Connect Web UI JavaScript

const state = {
  activePane: 'left',
  remotes: [],
  isEditingRemote: false,
  editingRemoteName: '',
  settings: {
    showHidden: false,
    defaultView: 'list' // 'list' or 'grid'
  },
  panes: {
    left: {
      remote: 'local',
      path: '.',
      items: [],
      filter: '',
      selected: new Set(),
      viewMode: 'list', // 'list' or 'grid'
      isEditingPath: false
    },
    right: {
      remote: 'local',
      path: '.',
      items: [],
      filter: '',
      selected: new Set(),
      viewMode: 'list',
      isEditingPath: false
    }
  }
};

// Initialize Application
document.addEventListener('DOMContentLoaded', () => {
  initApp();
});

async function initApp() {
  loadSettingsFromStorage();
  setupEventListeners();
  await loadRemotes();
  await loadPane('left');
  await loadPane('right');
  updateUI();
  initWizard();
}

// LocalStorage Settings Management
function loadSettingsFromStorage() {
  try {
    const saved = localStorage.getItem('tc_settings');
    if (saved) {
      const parsed = JSON.parse(saved);
      if (typeof parsed.showHidden === 'boolean') state.settings.showHidden = parsed.showHidden;
      if (parsed.defaultView === 'list' || parsed.defaultView === 'grid') {
        state.settings.defaultView = parsed.defaultView;
      }
    }
  } catch (err) {
    console.error('Failed to load settings:', err);
  }

  // Set initial pane view modes from settings
  state.panes.left.viewMode = state.settings.defaultView;
  state.panes.right.viewMode = state.settings.defaultView;
  updateViewToggleHeaderBtn();
}

function saveSettingsToStorage() {
  try {
    localStorage.setItem('tc_settings', JSON.stringify(state.settings));
  } catch (err) {
    console.error('Failed to save settings:', err);
  }
}

// Event Delegation & Event Listeners (XSS Hardened)
function setupEventListeners() {
  // Global Header Actions
  document.getElementById('btn-view-toggle')?.addEventListener('click', toggleHeaderViewMode);
  document.getElementById('btn-settings')?.addEventListener('click', openSettingsModal);
  document.getElementById('btn-add-conn')?.addEventListener('click', openWizardModal);

  // Settings Modal Buttons
  document.getElementById('btn-close-settings')?.addEventListener('click', () => closeModal('settings-modal'));
  document.getElementById('btn-cancel-settings')?.addEventListener('click', () => closeModal('settings-modal'));
  document.getElementById('btn-save-settings')?.addEventListener('click', saveSettingsFromModal);

  // Mobile Tabs
  document.getElementById('mobile-tabs')?.addEventListener('click', (e) => {
    const btn = e.target.closest('[data-tab]');
    if (btn) {
      const tab = btn.getAttribute('data-tab');
      if (tab) switchMobileTab(tab);
    }
  });

  // Pane Active Focus Listeners
  ['left', 'right'].forEach(paneId => {
    const paneEl = document.getElementById(`pane-${paneId}`);
    paneEl?.addEventListener('click', (e) => {
      setActivePane(paneId);
    });

    // Remote selector change
    const remoteSelect = document.getElementById(`remote-select-${paneId}`);
    remoteSelect?.addEventListener('change', (e) => {
      onRemoteSelectChange(paneId, e.target.value);
    });

    // Up parent directory button
    document.getElementById(`btn-up-${paneId}`)?.addEventListener('click', (e) => {
      e.stopPropagation();
      navigateParent(paneId);
    });

    // Address bar click / Edit path button
    document.getElementById(`breadcrumbs-${paneId}`)?.addEventListener('click', (e) => {
      e.stopPropagation();
      const breadcrumb = e.target.closest('[data-path]');
      if (breadcrumb) {
        const targetPath = breadcrumb.getAttribute('data-path');
        navigateToPath(paneId, state.panes[paneId].remote, targetPath);
      } else {
        // Clicked empty space in address bar -> enter path edit mode
        enablePathEditMode(paneId);
      }
    });

    document.getElementById(`btn-edit-path-${paneId}`)?.addEventListener('click', (e) => {
      e.stopPropagation();
      enablePathEditMode(paneId);
    });

    // Path edit text input enter / escape / blur
    const pathInput = document.getElementById(`path-input-${paneId}`);
    pathInput?.addEventListener('keydown', (e) => {
      if (e.key === 'Enter') {
        e.preventDefault();
        submitPathEdit(paneId, pathInput.value);
      } else if (e.key === 'Escape') {
        e.preventDefault();
        disablePathEditMode(paneId);
      }
    });
    pathInput?.addEventListener('blur', () => {
      disablePathEditMode(paneId);
    });

    // Filter input
    const filterInput = document.getElementById(`filter-input-${paneId}`);
    filterInput?.addEventListener('input', (e) => {
      state.panes[paneId].filter = e.target.value.toLowerCase().trim();
      renderPaneList(paneId);
    });

    // Select all checkbox
    const selectAllChk = document.getElementById(`select-all-${paneId}`);
    selectAllChk?.addEventListener('change', (e) => {
      toggleSelectAll(paneId, e.target.checked);
    });

    // File list event delegation for rows/cards
    const listContainer = document.getElementById(`file-list-${paneId}`);
    listContainer?.addEventListener('click', (e) => {
      const row = e.target.closest('[data-item-name]');
      if (!row) return;

      const itemName = row.getAttribute('data-item-name');
      const isDir = row.getAttribute('data-is-dir') === 'true';

      // Checkbox click check
      if (e.target.matches('input[type="checkbox"]')) {
        e.stopPropagation();
        toggleItemSelect(paneId, itemName, e.target.checked);
        return;
      }

      onRowClick(paneId, itemName, isDir, e);
    });
  });

  // Action Toolbar Buttons
  document.getElementById('btn-action-copy')?.addEventListener('click', handleCopy);
  document.getElementById('btn-action-move')?.addEventListener('click', handleMove);
  document.getElementById('btn-action-mkdir')?.addEventListener('click', openMkdirModal);
  document.getElementById('btn-action-delete')?.addEventListener('click', handleDelete);
  document.getElementById('btn-action-refresh')?.addEventListener('click', refreshActivePane);
  document.getElementById('btn-action-remotes')?.addEventListener('click', openChangeRemoteModal);

  // Mkdir Modal
  document.getElementById('btn-confirm-mkdir')?.addEventListener('click', confirmMkdir);
  document.getElementById('btn-cancel-mkdir')?.addEventListener('click', () => closeModal('mkdir-modal'));

  // Remote Management Modal Event Delegation
  document.getElementById('btn-close-remote-modal')?.addEventListener('click', () => closeModal('remote-modal'));
  document.getElementById('btn-remote-modal-close')?.addEventListener('click', () => closeModal('remote-modal'));
  document.getElementById('btn-remote-modal-add')?.addEventListener('click', () => {
    closeModal('remote-modal');
    openWizardModal();
  });

  document.getElementById('remote-list-modal-body')?.addEventListener('click', (e) => {
    const btn = e.target.closest('[data-remote-action]');
    if (!btn) return;

    const action = btn.getAttribute('data-remote-action');
    const remote = btn.getAttribute('data-remote');

    if (action === 'select') {
      selectRemoteFromModal(remote);
    } else if (action === 'edit') {
      editRemote(remote);
    } else if (action === 'delete') {
      deleteRemote(remote);
    }
  });

  // Wizard Modal
  document.getElementById('btn-close-wizard')?.addEventListener('click', () => closeModal('wizard-modal'));
  document.getElementById('btn-cancel-wizard')?.addEventListener('click', () => closeModal('wizard-modal'));
  document.getElementById('wizard-type')?.addEventListener('change', (e) => {
    onWizardTypeChange(e.target.value);
  });
  document.getElementById('wizard-form')?.addEventListener('submit', handleWizardSubmit);
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
      <span style="font-weight:bold; cursor:pointer;" data-remote-action="select" data-remote="local">📁 Local Filesystem</span>
      <button class="action-btn primary" style="padding:4px 8px; font-size:0.8rem;" data-remote-action="select" data-remote="local">Select</button>
    </div>
  `;

  if (state.remotes.length === 0) {
    html += `<div style="padding:12px; color: var(--text-muted); text-align:center;">No cloud remotes configured yet.</div>`;
  } else {
    state.remotes.forEach(remote => {
      const cleanName = remote.endsWith(':') ? remote.slice(0, -1) : remote;
      html += `
        <div class="remote-item-row" style="display:flex; align-items:center; justify-content:space-between; padding:8px; border-bottom:1px solid rgba(255,255,255,0.1);">
          <span style="font-weight:bold; cursor:pointer;" data-remote-action="select" data-remote="${escapeHtml(remote)}">☁️ ${escapeHtml(cleanName)}</span>
          <div style="display:flex; gap:6px;">
            <button class="action-btn primary" style="padding:4px 8px; font-size:0.8rem;" data-remote-action="select" data-remote="${escapeHtml(remote)}">Select</button>
            <button class="action-btn secondary" style="padding:4px 8px; font-size:0.8rem;" data-remote-action="edit" data-remote="${escapeHtml(cleanName)}">✏️ Edit</button>
            <button class="action-btn danger" style="padding:4px 8px; font-size:0.8rem;" data-remote-action="delete" data-remote="${escapeHtml(cleanName)}">🗑️ Delete</button>
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
    path: pane.path,
    hidden: state.settings.showHidden ? 'true' : 'false'
  });

  try {
    const items = await apiCall(`api/entries?${query.toString()}`);
    pane.items = sortEntries(items || []);
    renderAddressBar(paneId);
    renderPaneList(paneId);
  } catch (err) {
    pane.items = [];
    renderAddressBar(paneId);
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

// Dual-Mode Address Bar Logic
function renderAddressBar(paneId) {
  const pane = state.panes[paneId];
  const breadcrumbsEl = document.getElementById(`breadcrumbs-${paneId}`);
  if (!breadcrumbsEl) return;

  const displayRemote = pane.remote === 'local' ? 'Local' : pane.remote;
  let html = `<span class="breadcrumb-item" data-path=".">${escapeHtml(displayRemote)}</span>`;

  if (pane.path && pane.path !== '.' && pane.path !== '/') {
    const segments = pane.path.split('/').filter(Boolean);
    let accum = '';
    segments.forEach((seg, idx) => {
      accum = accum ? `${accum}/${seg}` : seg;
      html += `<span class="breadcrumb-separator">/</span>`;
      html += `<span class="breadcrumb-item" data-path="${escapeHtml(accum)}">${escapeHtml(seg)}</span>`;
    });
  }

  breadcrumbsEl.innerHTML = html;
}

function enablePathEditMode(paneId) {
  const pane = state.panes[paneId];
  pane.isEditingPath = true;

  const breadcrumbsEl = document.getElementById(`breadcrumbs-${paneId}`);
  const inputEl = document.getElementById(`path-input-${paneId}`);

  if (breadcrumbsEl) breadcrumbsEl.classList.add('hidden');
  if (inputEl) {
    inputEl.classList.remove('hidden');

    let fullPathStr = pane.path;
    if (pane.remote && pane.remote !== 'local') {
      const cleanRemote = pane.remote.endsWith(':') ? pane.remote : `${pane.remote}:`;
      fullPathStr = pane.path === '.' ? cleanRemote : `${cleanRemote}${pane.path}`;
    }
    inputEl.value = fullPathStr;
    inputEl.focus();
    inputEl.select();
  }
}

function disablePathEditMode(paneId) {
  const pane = state.panes[paneId];
  pane.isEditingPath = false;

  const breadcrumbsEl = document.getElementById(`breadcrumbs-${paneId}`);
  const inputEl = document.getElementById(`path-input-${paneId}`);

  if (breadcrumbsEl) breadcrumbsEl.classList.remove('hidden');
  if (inputEl) inputEl.classList.add('hidden');
}

function submitPathEdit(paneId, rawInputValue) {
  const trimmed = rawInputValue.trim();
  disablePathEditMode(paneId);
  if (!trimmed) return;

  let targetRemote = 'local';
  let targetPath = trimmed;

  const colonIdx = trimmed.indexOf(':');
  if (colonIdx !== -1) {
    const potentialRemote = trimmed.slice(0, colonIdx + 1); // e.g. "gdrive:"
    const cleanRemoteName = potentialRemote.slice(0, -1);

    // Check if potentialRemote is a known remote or matches clean remote
    const match = state.remotes.find(r => r === potentialRemote || r.slice(0, -1) === cleanRemoteName);
    if (match) {
      targetRemote = match;
      targetPath = trimmed.slice(colonIdx + 1);
    }
  }

  navigateToPath(paneId, targetRemote, targetPath);
}

function navigateToPath(paneId, remote, targetPath) {
  const pane = state.panes[paneId];
  pane.remote = remote;
  pane.path = targetPath || '.';

  // Update remote dropdown UI
  const remoteSelect = document.getElementById(`remote-select-${paneId}`);
  if (remoteSelect) remoteSelect.value = pane.remote;

  loadPane(paneId);
}

// File Extension Icon Helper (Microsoft Files / Fluent Icons)
function getFileTypeIcon(item) {
  if (item.is_dir) return '📁';

  const ext = item.name.split('.').pop().toLowerCase();
  switch (ext) {
    case 'zip': case 'tar': case 'gz': case 'bz2': case '7z': case 'rar': case 'xz':
      return '📦';
    case 'mp4': case 'mkv': case 'avi': case 'mov': case 'wmv': case 'flv': case 'webm':
      return '🎬';
    case 'mp3': case 'wav': case 'flac': case 'aac': case 'ogg': case 'm4a':
      return '🎵';
    case 'go': case 'js': case 'ts': case 'py': case 'html': case 'css': case 'json': case 'sh': case 'c': case 'cpp':
      return '💻';
    case 'png': case 'jpg': case 'jpeg': case 'gif': case 'svg': case 'webp': case 'bmp': case 'ico':
      return '🖼️';
    case 'pdf': case 'txt': case 'md': case 'doc': case 'docx': case 'xls': case 'xlsx': case 'ppt': case 'pptx':
      return '📄';
    default:
      return '📄';
  }
}

// Rendering List vs Grid View
function renderPaneList(paneId) {
  const pane = state.panes[paneId];
  const container = document.getElementById(`file-list-${paneId}`);
  const header = document.getElementById(`table-header-${paneId}`);
  const footer = document.getElementById(`pane-footer-${paneId}`);

  if (!container) return;

  // Toggle list vs grid CSS classes
  if (pane.viewMode === 'grid') {
    container.className = 'file-list grid-view';
    if (header) header.classList.add('hidden');
  } else {
    container.className = 'file-list list-view';
    if (header) header.classList.remove('hidden');
  }

  container.innerHTML = '';

  // Apply instant filter
  let visibleItems = pane.items;
  if (pane.filter) {
    visibleItems = pane.items.filter(i => i.name.toLowerCase().includes(pane.filter));
  }

  let html = '';

  if (canNavigateUp(pane.path)) {
    if (pane.viewMode === 'grid') {
      html += `
        <div class="file-card" data-item-name=".." data-is-dir="true">
          <div class="card-icon">📁</div>
          <div class="card-name is-dir">.. (Parent)</div>
          <div class="card-meta">&lt;UP&gt;</div>
        </div>
      `;
    } else {
      html += `
        <div class="file-row" data-item-name=".." data-is-dir="true">
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
  }

  if (visibleItems.length === 0) {
    html += `<div style="grid-column: 1/-1; padding: 20px; color: var(--text-muted); text-align: center;">(No items found)</div>`;
  } else {
    visibleItems.forEach((item) => {
      const isSelected = pane.selected.has(item.name);
      const icon = getFileTypeIcon(item);
      const sizeStr = item.is_dir ? '&lt;DIR&gt;' : formatSize(item.size);
      const mtimeStr = item.mod_time ? formatDate(item.mod_time) : '';

      if (pane.viewMode === 'grid') {
        html += `
          <div class="file-card ${isSelected ? 'selected' : ''}" data-item-name="${escapeHtml(item.name)}" data-is-dir="${item.is_dir}">
            <div class="card-check">
              <input type="checkbox" ${isSelected ? 'checked' : ''}>
            </div>
            <div class="card-icon">${icon}</div>
            <div class="card-name ${item.is_dir ? 'is-dir' : ''}">${escapeHtml(item.name)}</div>
            <div class="card-meta">${sizeStr}</div>
          </div>
        `;
      } else {
        html += `
          <div class="file-row ${isSelected ? 'selected' : ''}" data-item-name="${escapeHtml(item.name)}" data-is-dir="${item.is_dir}">
            <span class="col-check">
              <input type="checkbox" ${isSelected ? 'checked' : ''}>
            </span>
            <span class="col-name">
              <span class="item-icon">${icon}</span>
              <span class="item-name ${item.is_dir ? 'is-dir' : ''}">${escapeHtml(item.name)}</span>
            </span>
            <span class="col-size">${sizeStr}</span>
            <span class="col-mtime">${mtimeStr}</span>
          </div>
        `;
      }
    });
  }

  container.innerHTML = html;

  if (footer) {
    const selCount = pane.selected.size;
    footer.innerText = `${visibleItems.length} items ${selCount > 0 ? `(${selCount} selected)` : ''}`;
  }
}

// Settings Modal Logic
function openSettingsModal() {
  const chkHidden = document.getElementById('setting-show-hidden');
  const selDefaultView = document.getElementById('setting-default-view');

  if (chkHidden) chkHidden.checked = state.settings.showHidden;
  if (selDefaultView) selDefaultView.value = state.settings.defaultView;

  document.getElementById('settings-modal')?.classList.remove('hidden');
}

function saveSettingsFromModal() {
  const chkHidden = document.getElementById('setting-show-hidden');
  const selDefaultView = document.getElementById('setting-default-view');

  if (chkHidden) state.settings.showHidden = chkHidden.checked;
  if (selDefaultView) state.settings.defaultView = selDefaultView.value;

  saveSettingsToStorage();
  closeModal('settings-modal');

  // Reload panes to reflect hidden files toggle
  loadPane('left');
  loadPane('right');
  showToast('Preferences saved', 'success');
}

function toggleHeaderViewMode() {
  const activePaneObj = state.panes[state.activePane];
  activePaneObj.viewMode = activePaneObj.viewMode === 'list' ? 'grid' : 'list';
  updateViewToggleHeaderBtn();
  renderPaneList(state.activePane);
}

function updateViewToggleHeaderBtn() {
  const iconEl = document.getElementById('view-toggle-icon');
  const labelEl = document.getElementById('view-toggle-label');
  const activeMode = state.panes[state.activePane].viewMode;

  if (iconEl && labelEl) {
    if (activeMode === 'list') {
      iconEl.innerText = '🔲';
      labelEl.innerText = 'Grid';
    } else {
      iconEl.innerText = '📄';
      labelEl.innerText = 'List';
    }
  }
}

// UI Interaction Handlers
function setActivePane(paneId) {
  state.activePane = paneId;
  updateUI();
  updateViewToggleHeaderBtn();
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
  document.getElementById('mkdir-modal')?.classList.remove('hidden');
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
  document.getElementById('remote-modal')?.classList.remove('hidden');
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

// Fixed getParentPath: Handles remote paths with colons correctly
// e.g., remote:folder/subfolder -> remote:folder
// e.g., remote:folder -> remote:
// e.g., folder/subfolder -> folder
// e.g., folder -> .
function getParentPath(currentPath) {
  if (!currentPath || currentPath === '.' || currentPath === '/') return '.';

  let remotePrefix = '';
  let pathPart = currentPath;

  const colonIdx = currentPath.indexOf(':');
  if (colonIdx !== -1) {
    remotePrefix = currentPath.slice(0, colonIdx + 1); // "remote:"
    pathPart = currentPath.slice(colonIdx + 1);       // "folder/subfolder"
  }

  // Trim leading/trailing slashes on pathPart
  pathPart = pathPart.replace(/^\/+|\/+$/g, '');

  if (!pathPart) {
    return remotePrefix || '.';
  }

  const parts = pathPart.split('/').filter(Boolean);
  if (parts.length <= 1) {
    return remotePrefix ? remotePrefix : '.';
  }

  parts.pop();
  const parentSubPath = parts.join('/');
  return remotePrefix ? `${remotePrefix}${parentSubPath}` : parentSubPath;
}

function joinPath(base, child) {
  if (!base || base === '.') return child;

  // Handle trailing colon/slash
  if (base.endsWith(':')) {
    return `${base}${child}`;
  }
  if (base.endsWith('/')) {
    return `${base}${child}`;
  }
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
