/* ── Card accent colours from data-accent ───────────────────── */
document.querySelectorAll('.tool-card[data-accent]').forEach(card => {
  card.style.setProperty('--card-a', card.dataset.accent);
});

function onDragOver(e, el) {
  e.preventDefault();
  el.classList.add('drag-over');
}
function onDragLeave(el) {
  el.classList.remove('drag-over');
}
function onDrop(e, el, inputId) {
  e.preventDefault();
  el.classList.remove('drag-over');
  const input = document.getElementById(inputId);
  input._files = e.dataTransfer.files;
  renderFileList(input._files, el);
}
function onFileChange(input, dzId) {
  input._files = null;
  renderFileList(input.files, document.getElementById(dzId));
}

function renderFileList(files, dzEl) {
  if (!files || files.length === 0) return;
  const suffix = dzEl.id.replace('dz-', '');
  const fl = document.getElementById('fl-' + suffix);
  if (!fl) return;
  const names = Array.from(files).map(f => f.name);
  fl.textContent = files.length > 1
    ? `${files.length} files: ${names.join(', ')}`.slice(0, 72) + (names.join(', ').length > 68 ? '…' : '')
    : names[0];
  const c = dzEl.closest('.tool-card')?.style.getPropertyValue('--card-a');
  if (c) dzEl.style.borderColor = c;
}

function getFiles(inputId) {
  const el = document.getElementById(inputId);
  return el._files || el.files;
}

function setLoading(key, on) {
  const sp  = document.getElementById('sp-' + key);
  const btn = sp?.closest('button');
  if (sp)  sp.classList.toggle('visible', on);
  if (btn) btn.disabled = on;
}

async function submitOne(endpoint, inputId, extras, resultId, key) {
  const files = getFiles(inputId);
  if (!files || files.length === 0) { toast('Please select a file first'); return; }
  const fd = new FormData();
  fd.append('file', files[0]);
  for (const [k, v] of Object.entries(extras)) fd.append(k, String(v));
  setLoading(key, true);
  try {
    const res  = await fetch('/api/' + endpoint, { method: 'POST', body: fd });
    const data = await res.json();
    showResult(resultId, res.ok, data);
  } catch (err) {
    showResult(resultId, false, { error: err.message });
  }
  setLoading(key, false);
}

async function submitMany(endpoint, inputId, extras, resultId, key) {
  const files = getFiles(inputId);
  if (!files || files.length === 0) { toast('Please select files first'); return; }
  const fd = new FormData();
  for (const f of files) fd.append('files', f);
  for (const [k, v] of Object.entries(extras)) fd.append(k, String(v));
  setLoading(key, true);
  try {
    const res  = await fetch('/api/' + endpoint, { method: 'POST', body: fd });
    const data = await res.json();
    showResult(resultId, res.ok, data);
  } catch (err) {
    showResult(resultId, false, { error: err.message });
  }
  setLoading(key, false);
}

function showResult(id, success, data) {
  const box = document.getElementById(id);
  const cls = success ? 'ok' : 'err';
  const icon = success ? '✓' : '✗';
  const msg  = data.message || data.error || 'Unknown response';
  let meta = '';
  if (data.originalSize && data.newSize) {
    const orig = (data.originalSize / 1024).toFixed(1);
    const next = (data.newSize / 1024).toFixed(1);
    meta = `<div class="result-meta">${orig} KB → ${next} KB &nbsp;·&nbsp; saved ~${data.savings}%</div>`;
  }
  if (data.pages) meta += `<div class="result-meta">${data.pages} parts</div>`;
  const dl = (success && data.download)
    ? `<a class="dl-btn" href="${data.download}" download="${data.filename}">↓ Download ${data.filename}</a>`
    : '';
  box.innerHTML = `<div class="result-inner ${cls}"><div class="result-msg">${icon} ${msg}</div>${meta}${dl}</div>`;
  box.classList.add('show');
}

function toast(msg) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.classList.add('show');
  setTimeout(() => el.classList.remove('show'), 3000);
}

/* ── Merge: reorderable file list ───────────────────────────── */

let mergeFiles = [];
let dragSrcIdx = null;

function onMergeFilesChange(input) {
  const newFiles = Array.from(input._files || input.files);
  newFiles.forEach(f => {
    if (!mergeFiles.find(x => x.name === f.name && x.size === f.size)) {
      mergeFiles.push(f);
    }
  });
  input.value = '';
  input._files = null;
  renderMergeList();
  const dz = document.getElementById('dz-merge');
  const c = dz.closest('.tool-card')?.style.getPropertyValue('--card-a');
  if (c && mergeFiles.length) dz.style.borderColor = c;
}

function renderMergeList() {
  const ul = document.getElementById('merge-list');
  ul.innerHTML = '';
  mergeFiles.forEach((f, i) => {
    const li = document.createElement('li');
    li.className = 'merge-item';
    li.draggable = true;
    li.dataset.idx = i;

    li.innerHTML = `
      <span class="merge-item-handle">⠿</span>
      <span class="merge-item-name" title="${f.name}">${f.name}</span>
      <span class="merge-item-size">${(f.size / 1024).toFixed(0)} KB</span>
      <button class="merge-item-remove" onclick="removeMergeFile(${i})" title="Remove">✕</button>
    `;

    li.addEventListener('dragstart', e => {
      dragSrcIdx = i;
      li.classList.add('drag-source');
      e.dataTransfer.effectAllowed = 'move';
    });
    li.addEventListener('dragend', () => {
      li.classList.remove('drag-source');
      document.querySelectorAll('.merge-item').forEach(el => el.classList.remove('drag-over-item'));
    });
    li.addEventListener('dragover', e => {
      e.preventDefault();
      e.dataTransfer.dropEffect = 'move';
      document.querySelectorAll('.merge-item').forEach(el => el.classList.remove('drag-over-item'));
      li.classList.add('drag-over-item');
    });
    li.addEventListener('drop', e => {
      e.preventDefault();
      e.stopPropagation();
      const targetIdx = parseInt(li.dataset.idx);
      if (dragSrcIdx === null || dragSrcIdx === targetIdx) return;
      const moved = mergeFiles.splice(dragSrcIdx, 1)[0];
      mergeFiles.splice(targetIdx, 0, moved);
      dragSrcIdx = null;
      renderMergeList();
    });

    ul.appendChild(li);
  });
}

function removeMergeFile(idx) {
  mergeFiles.splice(idx, 1);
  renderMergeList();
}

async function submitMerge() {
  if (mergeFiles.length < 2) { toast('Add at least 2 PDFs to merge'); return; }
  const fd = new FormData();
  mergeFiles.forEach(f => fd.append('files', f));
  setLoading('merge', true);
  try {
    const res  = await fetch('/api/merge', { method: 'POST', body: fd });
    const data = await res.json();
    showResult('rb-merge', res.ok, data);
    if (res.ok) mergeFiles = [], renderMergeList();
  } catch (err) {
    showResult('rb-merge', false, { error: err.message });
  }
  setLoading('merge', false);
}

/* ── Split: mode toggle + custom submit ─────────────────────── */

function onSplitModeChange() {
  const mode = document.getElementById('split-mode').value;
  document.getElementById('split-chunk-row').style.display  = mode === 'custom' ? 'flex' : 'none';
  document.getElementById('split-ranges-row').style.display = mode === 'ranges' ? 'flex' : 'none';
}

async function submitSplit() {
  const files = getFiles('fi-split');
  if (!files || files.length === 0) { toast('Please select a PDF first'); return; }

  const mode = document.getElementById('split-mode').value;
  const fd = new FormData();
  fd.append('file', files[0]);

  if (mode === '1') {
    fd.append('span', '1');
  } else if (mode === 'custom') {
    const span = parseInt(document.getElementById('split-span').value) || 1;
    fd.append('span', String(span));
  } else if (mode === 'ranges') {
    const raw = document.getElementById('split-ranges').value.trim();
    if (!raw) { toast('Enter page ranges, e.g. 1-3, 5, 7-9'); return; }
    fd.append('span', '1');
    fd.append('ranges', raw);
  }

  setLoading('split', true);
  try {
    const res  = await fetch('/api/split', { method: 'POST', body: fd });
    const data = await res.json();
    showResult('rb-split', res.ok, data);
  } catch (err) {
    showResult('rb-split', false, { error: err.message });
  }
  setLoading('split', false);
}