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

function openModal(id) {
  const m = document.getElementById(id);
  m.classList.add('open');
  document.body.style.overflow = 'hidden';
}

function closeModal(id) {
  const m = document.getElementById(id);
  m.classList.remove('open');
  document.body.style.overflow = '';
}

document.addEventListener('click', e => {
  if (e.target.classList.contains('modal-overlay')) {
    closeModal(e.target.id);
  }
});

let mergeFiles = [];

function onMergeFilesChange(input) {
  const newFiles = Array.from(input._files || input.files);
  newFiles.forEach(f => {
    if (!mergeFiles.find(x => x.name === f.name && x.size === f.size)) {
      mergeFiles.push(f);
    }
  });
  input.value = '';
  input._files = null;
  const dz = document.getElementById('dz-merge');
  const c = dz.closest('.tool-card')?.style.getPropertyValue('--card-a');
  if (c && mergeFiles.length) dz.style.borderColor = c;
  openMergeModal();
}

function openMergeModal() {
  renderMergeModal();
  openModal('modal-merge');
}

function renderMergeModal() {
  const list = document.getElementById('merge-modal-list');
  list.innerHTML = '';
  mergeFiles.forEach((f, i) => {
    const item = document.createElement('div');
    item.className = 'mmodal-item';
    item.innerHTML = `
      <div class="mmodal-order">
        <button class="mmodal-arrow" onclick="moveMergeFile(${i}, -1)" ${i === 0 ? 'disabled' : ''}>▲</button>
        <span class="mmodal-num">${i + 1}</span>
        <button class="mmodal-arrow" onclick="moveMergeFile(${i}, 1)" ${i === mergeFiles.length - 1 ? 'disabled' : ''}>▼</button>
      </div>
      <div class="mmodal-info">
        <span class="mmodal-name" title="${f.name}">${f.name}</span>
        <span class="mmodal-size">${(f.size / 1024).toFixed(0)} KB</span>
      </div>
      <button class="mmodal-remove" onclick="removeMergeModal(${i})">✕</button>
    `;
    list.appendChild(item);
  });
}

function moveMergeFile(idx, dir) {
  const target = idx + dir;
  if (target < 0 || target >= mergeFiles.length) return;
  const tmp = mergeFiles[idx];
  mergeFiles[idx] = mergeFiles[target];
  mergeFiles[target] = tmp;
  renderMergeModal();
}

function removeMergeModal(idx) {
  mergeFiles.splice(idx, 1);
  if (mergeFiles.length === 0) {
    closeModal('modal-merge');
    return;
  }
  renderMergeModal();
}

async function submitMerge() {
  if (mergeFiles.length < 2) { toast('Add at least 2 PDFs to merge'); return; }
  const fd = new FormData();
  mergeFiles.forEach(f => fd.append('files', f));
  closeModal('modal-merge');
  setLoading('merge', true);
  try {
    const res  = await fetch('/api/merge', { method: 'POST', body: fd });
    const data = await res.json();
    showResult('rb-merge', res.ok, data);
    if (res.ok) mergeFiles = [];
  } catch (err) {
    showResult('rb-merge', false, { error: err.message });
  }
  setLoading('merge', false);
}

let splitFile = null;
let splitRanges = [{ start: '', end: '' }];

function onSplitFileChange(input) {
  splitFile = (input._files || input.files)[0] || null;
  input.value = '';
  input._files = null;
  if (!splitFile) return;
  const dz = document.getElementById('dz-split');
  const c = dz.closest('.tool-card')?.style.getPropertyValue('--card-a');
  if (c) dz.style.borderColor = c;
  const fl = document.getElementById('fl-split');
  if (fl) fl.textContent = splitFile.name;
  openSplitModal();
}

function openSplitModal() {
  splitRanges = [{ start: '', end: '' }];
  document.getElementById('split-modal-filename').textContent = splitFile ? splitFile.name : '';
  renderSplitRanges();
  openModal('modal-split');
}

function renderSplitRanges() {
  const container = document.getElementById('split-ranges-list');
  container.innerHTML = '';
  splitRanges.forEach((r, i) => {
    const row = document.createElement('div');
    row.className = 'smodal-row';
    row.innerHTML = `
      <span class="smodal-label">Part ${i + 1}</span>
      <input class="field-input smodal-input" type="number" min="1" placeholder="Start" value="${r.start}"
        oninput="splitRanges[${i}].start = this.value">
      <span class="smodal-sep">–</span>
      <input class="field-input smodal-input" type="number" min="1" placeholder="End" value="${r.end}"
        oninput="splitRanges[${i}].end = this.value">
      ${splitRanges.length > 1 ? `<button class="smodal-remove" onclick="removeSplitRange(${i})">✕</button>` : '<span class="smodal-remove-ph"></span>'}
    `;
    container.appendChild(row);
  });
}

function addSplitRange() {
  splitRanges.push({ start: '', end: '' });
  renderSplitRanges();
}

function removeSplitRange(idx) {
  splitRanges.splice(idx, 1);
  renderSplitRanges();
}

async function submitSplit() {
  if (!splitFile) { toast('Please select a PDF first'); return; }

  const validRanges = splitRanges.filter(r => r.start !== '' && r.end !== '');
  if (validRanges.length === 0) { toast('Enter at least one page range'); return; }

  for (const r of validRanges) {
    const s = parseInt(r.start), e = parseInt(r.end);
    if (isNaN(s) || isNaN(e) || s < 1 || e < s) {
      toast('Each range needs valid start ≤ end page numbers');
      return;
    }
  }

  const rawRanges = validRanges.map(r => `${r.start}-${r.end}`).join(',');

  const fd = new FormData();
  fd.append('file', splitFile);
  fd.append('ranges', rawRanges);

  closeModal('modal-split');
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