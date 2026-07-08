/* ── ILovePDF — Frontend JS ──────────────────────────────── */

/* ── Theme & Color Management ───────────────────────────── */
(function initTheme() {
  const root = document.documentElement;
  const saved = localStorage.getItem('ilpdf-theme') || 'dark';
  const savedColor = localStorage.getItem('ilpdf-accent') || '#ff4d6d';

  // Apply saved theme
  if (saved === 'light') root.setAttribute('data-theme', 'light');
  applyAccent(savedColor);

  // Theme toggle
  const toggle = document.getElementById('themeToggle');
  if (toggle) {
    updateKnob(saved);
    toggle.addEventListener('click', () => {
      const current = root.getAttribute('data-theme');
      const next = current === 'light' ? 'dark' : 'light';
      if (next === 'light') {
        root.setAttribute('data-theme', 'light');
      } else {
        root.removeAttribute('data-theme');
      }
      localStorage.setItem('ilpdf-theme', next);
      updateKnob(next);
    });
  }

  function updateKnob(theme) {
    const knob = document.querySelector('.theme-toggle-knob');
    if (knob) knob.textContent = theme === 'light' ? '☀️' : '🌙';
  }

  // Color picker button — toggle presets panel
  const pickerBtn = document.getElementById('colorPickerBtn');
  const presetsPanel = document.getElementById('colorPresets');
  const pickerInput = document.getElementById('colorPickerInput');

  if (pickerBtn && presetsPanel) {
    pickerBtn.addEventListener('click', (e) => {
      e.stopPropagation();
      presetsPanel.classList.toggle('open');
    });

    // Close on outside click
    document.addEventListener('click', (e) => {
      if (!presetsPanel.contains(e.target) && e.target !== pickerBtn) {
        presetsPanel.classList.remove('open');
      }
    });

    // Preset color swatches
    presetsPanel.querySelectorAll('.color-preset').forEach(swatch => {
      swatch.addEventListener('click', () => {
        const color = swatch.dataset.color;
        applyAccent(color);
        localStorage.setItem('ilpdf-accent', color);
        pickerInput.value = color;

        presetsPanel.querySelectorAll('.color-preset').forEach(s => s.classList.remove('active'));
        swatch.classList.add('active');
      });
    });

    // Custom color input
    if (pickerInput) {
      pickerInput.value = savedColor;
      pickerInput.addEventListener('input', (e) => {
        applyAccent(e.target.value);
        localStorage.setItem('ilpdf-accent', e.target.value);
        presetsPanel.querySelectorAll('.color-preset').forEach(s => s.classList.remove('active'));
      });
    }
  }

  // Mark saved preset as active on load
  if (presetsPanel) {
    presetsPanel.querySelectorAll('.color-preset').forEach(s => {
      s.classList.toggle('active', s.dataset.color === savedColor);
    });
  }

  function applyAccent(color) {
    root.style.setProperty('--accent', color);

    // Update color picker button
    const btn = document.getElementById('colorPickerBtn');
    if (btn) btn.style.background = color;

    // Update hero gradient with the chosen accent
    const heroEm = document.querySelector('.hero h1 em');
    if (heroEm) {
      heroEm.style.background = `linear-gradient(135deg, ${color} 0%, #c77dff 45%, #48cae4 100%)`;
      heroEm.style.webkitBackgroundClip = 'text';
      heroEm.style.webkitTextFillColor = 'transparent';
    }

    // Update logo gradient
    const logo = document.querySelector('.logo');
    if (logo) {
      logo.style.background = `linear-gradient(135deg, ${color} 0%, #c77dff 50%, #48cae4 100%)`;
      logo.style.backgroundSize = '200% auto';
      logo.style.webkitBackgroundClip = 'text';
      logo.style.webkitTextFillColor = 'transparent';
    }

    // Update nav accent tag
    const tag = document.querySelector('.nav-tag.accent');
    if (tag) {
      tag.style.borderColor = color + '55';
      tag.style.color = color;
      tag.style.background = color + '12';
    }
  }
})();

/* Initialize cards — set accent color + mouse-tracking glow */
document.querySelectorAll('.tool-card[data-accent]').forEach((card, i) => {
  card.style.setProperty('--card-a', card.dataset.accent);

  /* Staggered entrance animation */
  card.style.opacity = '0';
  card.style.transform = 'translateY(16px)';
  card.style.transition = 'opacity .5s ease, transform .5s ease, border-color .35s, box-shadow .35s';
  setTimeout(() => {
    card.style.opacity = '1';
    card.style.transform = 'translateY(0)';
  }, 60 + i * 40);

  /* Mouse-tracking inner glow */
  card.addEventListener('mousemove', e => {
    const rect = card.getBoundingClientRect();
    const x = ((e.clientX - rect.left) / rect.width * 100).toFixed(0);
    const y = ((e.clientY - rect.top)  / rect.height * 100).toFixed(0);
    card.style.setProperty('--mx', x + '%');
    card.style.setProperty('--my', y + '%');
  });
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

/* ── Premium file size gate (50 MB) ──────────────────────── */
const MAX_FREE_SIZE = 50 * 1024 * 1024; // 50 MB

function checkFileSizeLimit(files) {
  if (!files) return true;
  for (const f of files) {
    if (f.size > MAX_FREE_SIZE) {
      showPremiumModal(f.name, f.size);
      return false;
    }
  }
  return true;
}

function showPremiumModal(filename, size) {
  const mb = (size / (1024 * 1024)).toFixed(1);
  const modal = document.getElementById('premiumModal');
  if (modal) {
    document.getElementById('premium-filename').textContent = filename;
    document.getElementById('premium-size').textContent = mb + ' MB';
    modal.classList.add('open');
  } else {
    toast(`"${filename}" is ${mb} MB — files over 50 MB require a premium account`);
  }
}

function closePremiumModal() {
  document.getElementById('premiumModal')?.classList.remove('open');
}

/* ── Submit helpers — now handle binary blob responses ───── */

async function submitOne(endpoint, inputId, extras, resultId, key) {
  const files = getFiles(inputId);
  if (!files || files.length === 0) { toast('Please select a file first'); return; }
  if (!checkFileSizeLimit(files)) return;
  const fd = new FormData();
  fd.append('file', files[0]);
  for (const [k, v] of Object.entries(extras)) fd.append(k, String(v));
  setLoading(key, true);
  try {
    const res = await fetch('/api/' + endpoint, { method: 'POST', body: fd });
    if (!res.ok) {
      const data = await res.json();
      showResult(resultId, false, data);
    } else {
      const blob = await res.blob();
      const disposition = res.headers.get('Content-Disposition') || '';
      const filenameMatch = disposition.match(/filename="?([^";\n]+)"?/);
      const filename = filenameMatch ? filenameMatch[1] : 'download';
      const url = URL.createObjectURL(blob);

      const meta = {};
      const origSize = res.headers.get('X-Original-Size');
      const compSize = res.headers.get('X-Compressed-Size');
      const savings  = res.headers.get('X-Savings-Percent');
      if (origSize) meta.originalSize = parseInt(origSize);
      if (compSize) meta.newSize = parseInt(compSize);
      if (savings)  meta.savings = parseInt(savings);

      showBlobResult(resultId, url, filename, meta);
    }
  } catch (err) {
    showResult(resultId, false, { error: err.message });
  }
  setLoading(key, false);
}

async function submitMany(endpoint, inputId, extras, resultId, key) {
  const files = getFiles(inputId);
  if (!files || files.length === 0) { toast('Please select files first'); return; }
  if (!checkFileSizeLimit(files)) return;
  const fd = new FormData();
  for (const f of files) fd.append('files', f);
  for (const [k, v] of Object.entries(extras)) fd.append(k, String(v));
  setLoading(key, true);
  try {
    const res = await fetch('/api/' + endpoint, { method: 'POST', body: fd });
    if (!res.ok) {
      const data = await res.json();
      showResult(resultId, false, data);
    } else {
      const blob = await res.blob();
      const disposition = res.headers.get('Content-Disposition') || '';
      const filenameMatch = disposition.match(/filename="?([^";\n]+)"?/);
      const filename = filenameMatch ? filenameMatch[1] : 'download';
      const url = URL.createObjectURL(blob);
      showBlobResult(resultId, url, filename, {});
    }
  } catch (err) {
    showResult(resultId, false, { error: err.message });
  }
  setLoading(key, false);
}

/* ── Submit helpers for JSON-response endpoints ──────────── */

async function submitJson(endpoint, inputId, extras, resultId, key) {
  const files = getFiles(inputId);
  if (!files || files.length === 0) { toast('Please select a file first'); return; }
  if (!checkFileSizeLimit(files)) return;
  const fd = new FormData();
  fd.append('file', files[0]);
  for (const [k, v] of Object.entries(extras)) fd.append(k, String(v));
  setLoading(key, true);
  try {
    const res = await fetch('/api/' + endpoint, { method: 'POST', body: fd });
    const data = await res.json();
    if (!res.ok) {
      showResult(resultId, false, data);
    } else {
      showJsonResult(resultId, data);
    }
  } catch (err) {
    showResult(resultId, false, { error: err.message });
  }
  setLoading(key, false);
}

async function submitJsonMany(endpoint, inputId, extras, resultId, key) {
  const files = getFiles(inputId);
  if (!files || files.length < 2) { toast('Please select at least 2 files'); return; }
  const fd = new FormData();
  for (const f of files) fd.append('files', f);
  for (const [k, v] of Object.entries(extras)) fd.append(k, String(v));
  setLoading(key, true);
  try {
    const res = await fetch('/api/' + endpoint, { method: 'POST', body: fd });
    const data = await res.json();
    if (!res.ok) {
      showResult(resultId, false, data);
    } else {
      showJsonResult(resultId, data);
    }
  } catch (err) {
    showResult(resultId, false, { error: err.message });
  }
  setLoading(key, false);
}

/* ── Result rendering ────────────────────────────────────── */

function showBlobResult(id, blobUrl, filename, meta) {
  const box = document.getElementById(id);
  let metaHtml = '';
  if (meta.originalSize && meta.newSize) {
    const orig = (meta.originalSize / 1024).toFixed(1);
    const next = (meta.newSize / 1024).toFixed(1);
    metaHtml = `<div class="result-meta">${orig} KB → ${next} KB &nbsp;·&nbsp; saved ~${meta.savings || 0}%</div>`;
  }

  // PDF preview — show inline iframe for PDF files
  const isPdf = filename.toLowerCase().endsWith('.pdf');
  const previewHtml = isPdf
    ? `<div class="result-preview-wrap">
         <div class="result-preview-bar">
           <span>📄 Preview</span>
           <button class="result-preview-toggle" onclick="this.closest('.result-preview-wrap').classList.toggle('collapsed')">▼</button>
         </div>
         <iframe class="result-preview" src="${blobUrl}" title="PDF Preview"></iframe>
       </div>`
    : '';

  box.innerHTML = `<div class="result-inner ok">
    <div class="result-msg">✓ Processed successfully</div>
    ${metaHtml}
    <div class="result-actions">
      <a class="dl-btn" href="${blobUrl}" download="${filename}">↓ Download ${filename}</a>
      ${isPdf ? `<a class="dl-btn preview-btn" href="${blobUrl}" target="_blank">🔍 Open in New Tab</a>` : ''}
    </div>
    ${previewHtml}
  </div>`;
  box.classList.add('show');
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

function showJsonResult(id, data) {
  const box = document.getElementById(id);
  let html = '<div class="result-inner ok">';
  html += '<div class="result-msg">✓ Results</div>';

  // Extract text results
  if (data.pages && Array.isArray(data.pages)) {
    html += `<div class="result-meta">${data.count || data.pages.length} pages extracted</div>`;
    html += '<div class="json-results">';
    for (const p of data.pages) {
      html += `<div class="json-page"><strong>Page ${p.page}</strong><pre>${escapeHtml(p.text).slice(0, 500)}${p.text.length > 500 ? '…' : ''}</pre></div>`;
    }
    html += '</div>';
  }

  // Form fields results
  if (data.fields && Array.isArray(data.fields)) {
    html += `<div class="result-meta">${data.count || data.fields.length} fields found</div>`;
    html += '<div class="json-results"><table class="json-table"><tr><th>Name</th><th>Type</th><th>Value</th><th>Locked</th></tr>';
    for (const f of data.fields) {
      html += `<tr><td>${escapeHtml(f.name)}</td><td>${escapeHtml(f.type)}</td><td>${escapeHtml(f.value || '—')}</td><td>${f.locked ? '🔒' : '—'}</td></tr>`;
    }
    html += '</table></div>';
  }

  // Compare results
  if (data.summary !== undefined) {
    html += `<div class="result-meta">${escapeHtml(data.summary)}</div>`;
    html += `<div class="json-results">
      <div class="compare-row"><strong>File 1:</strong> ${data.file1_pages} pages ${data.file1_title ? '— ' + escapeHtml(data.file1_title) : ''}</div>
      <div class="compare-row"><strong>File 2:</strong> ${data.file2_pages} pages ${data.file2_title ? '— ' + escapeHtml(data.file2_title) : ''}</div>`;
    if (data.page_diffs && data.page_diffs.length > 0) {
      html += '<div class="compare-diffs"><strong>Differences:</strong>';
      for (const d of data.page_diffs) {
        html += `<div>Page ${d.page}: ${escapeHtml(d.difference)}</div>`;
      }
      html += '</div>';
    }
    html += '</div>';
  }

  // Fallback: raw JSON
  if (!data.pages && !data.fields && data.summary === undefined) {
    html += `<pre class="json-raw">${escapeHtml(JSON.stringify(data, null, 2))}</pre>`;
  }

  html += '</div>';
  box.innerHTML = html;
  box.classList.add('show');
}

function escapeHtml(str) {
  if (!str) return '';
  return String(str).replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');
}

/* ── Modals ──────────────────────────────────────────────── */

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

/* ── Merge modal ─────────────────────────────────────────── */

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
    const res = await fetch('/api/merge', { method: 'POST', body: fd });
    if (!res.ok) {
      const data = await res.json();
      showResult('rb-merge', false, data);
    } else {
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      showBlobResult('rb-merge', url, 'merged.pdf', {});
      mergeFiles = [];
    }
  } catch (err) {
    showResult('rb-merge', false, { error: err.message });
  }
  setLoading('merge', false);
}

/* ── Split modal ─────────────────────────────────────────── */

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
    const res = await fetch('/api/split', { method: 'POST', body: fd });
    if (!res.ok) {
      const data = await res.json();
      showResult('rb-split', false, data);
    } else {
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      showBlobResult('rb-split', url, 'split_pages.zip', {});
    }
  } catch (err) {
    showResult('rb-split', false, { error: err.message });
  }
  setLoading('split', false);
}

/* ── Search Functionality ────────────────────────────────── */
(function initSearch() {
  const input = document.getElementById('searchInput');
  const clearBtn = document.getElementById('searchClear');
  const countEl = document.getElementById('searchCount');
  if (!input) return;

  // Build search index from all tool cards automatically.
  // This indexes title + description so any future card is searchable.
  const cards = Array.from(document.querySelectorAll('.tool-card'));
  const sections = []; // { label, grid, cards[] }

  // Group cards by their parent grid + preceding section label
  document.querySelectorAll('.tool-grid').forEach(grid => {
    const label = grid.previousElementSibling;
    const gridCards = Array.from(grid.querySelectorAll('.tool-card'));
    sections.push({ label, grid, cards: gridCards });
  });

  // Build index: each card gets a searchable text blob + keyword aliases
  const keywordMap = {
    'merge': 'combine join concatenate',
    'split': 'extract separate divide',
    'compress': 'reduce shrink optimize smaller size',
    'rotate': 'turn flip orientation',
    'watermark': 'stamp overlay text',
    'jpg': 'image picture photo jpeg png convert',
    'pdf': 'document',
    'delete': 'remove erase',
    'reorder': 'rearrange sort move order',
    'insert': 'add blank empty page',
    'crop': 'trim cut margin',
    'page': 'number count',
    'extract': 'pull get text image',
    'encrypt': 'password protect lock security',
    'decrypt': 'unlock remove password',
    'redact': 'black censor hide sensitive',
    'sign': 'signature stamp approve',
    'form': 'field fill flatten data input',
    'compare': 'diff difference check',
    'repair': 'fix broken corrupted recover',
    'protect': 'password lock zip archive',
  };

  const cardIndex = cards.map(card => {
    const title = (card.querySelector('.card-title')?.textContent || '').toLowerCase();
    const desc  = (card.querySelector('.card-desc')?.textContent || '').toLowerCase();

    // Build keyword string from aliases
    let keywords = '';
    for (const [key, aliases] of Object.entries(keywordMap)) {
      if (title.includes(key) || desc.includes(key)) {
        keywords += ' ' + aliases;
      }
    }

    return {
      el: card,
      title,
      desc,
      text: title + ' ' + desc + ' ' + keywords,
    };
  });

  let debounceTimer;

  input.addEventListener('input', () => {
    clearTimeout(debounceTimer);
    debounceTimer = setTimeout(runSearch, 80);

    // Show/hide clear button
    clearBtn.classList.toggle('visible', input.value.length > 0);
  });

  clearBtn.addEventListener('click', () => {
    input.value = '';
    clearBtn.classList.remove('visible');
    runSearch();
    input.focus();
  });

  // Keyboard shortcut: Ctrl+K or / to focus search
  document.addEventListener('keydown', e => {
    if ((e.ctrlKey && e.key === 'k') || (e.key === '/' && !['INPUT','TEXTAREA','SELECT'].includes(document.activeElement.tagName))) {
      e.preventDefault();
      input.focus();
      input.select();
    }
    if (e.key === 'Escape' && document.activeElement === input) {
      input.value = '';
      clearBtn.classList.remove('visible');
      runSearch();
      input.blur();
    }
  });

  function runSearch() {
    const raw = input.value.trim().toLowerCase();

    if (!raw) {
      // Show all
      cards.forEach(c => c.classList.remove('search-hidden'));
      sections.forEach(s => {
        if (s.label) s.label.classList.remove('search-hidden');
        s.grid.classList.remove('search-hidden');
      });
      countEl.textContent = '';
      return;
    }

    // Split into search terms for multi-word matching
    const terms = raw.split(/\s+/).filter(t => t.length > 0);

    let matchCount = 0;

    // Score each card
    cardIndex.forEach(item => {
      const matches = terms.every(term => item.text.includes(term));
      item.el.classList.toggle('search-hidden', !matches);
      if (matches) matchCount++;
    });

    // Hide empty sections
    sections.forEach(s => {
      const visibleCards = s.cards.filter(c => !c.classList.contains('search-hidden'));
      const isEmpty = visibleCards.length === 0;
      if (s.label) s.label.classList.toggle('search-hidden', isEmpty);
      s.grid.classList.toggle('search-hidden', isEmpty);
    });

    // Update count
    if (matchCount === 0) {
      countEl.textContent = `No tools match "${raw}"`;
    } else {
      countEl.textContent = `${matchCount} tool${matchCount > 1 ? 's' : ''} found`;
    }
  }
})();

/* ── File Manager Modal ──────────────────────────────────── */
let fmFiles = [];     // array of { file, id }
let fmNextId = 0;
let fmCallback = null;
let fmAccept = '';

function openFileManager(accept, callback) {
  fmFiles = [];
  fmNextId = 0;
  fmCallback = callback;
  fmAccept = accept;
  const modal = document.getElementById('fileManagerModal');
  if (!modal) return;
  modal.classList.add('open');
  renderFmList();
  // Auto-open file picker
  fmPickFiles();
}

function closeFileManager() {
  document.getElementById('fileManagerModal')?.classList.remove('open');
}

function fmPickFiles() {
  const input = document.createElement('input');
  input.type = 'file';
  input.accept = fmAccept;
  input.multiple = true;
  input.onchange = () => {
    for (const f of input.files) {
      fmFiles.push({ file: f, id: fmNextId++ });
    }
    renderFmList();
  };
  input.click();
}

function fmRemoveFile(id) {
  fmFiles = fmFiles.filter(f => f.id !== id);
  renderFmList();
}

function renderFmList() {
  const list = document.getElementById('fmFileList');
  const count = document.getElementById('fmCount');
  const submitBtn = document.getElementById('fmSubmitBtn');
  if (!list) return;

  count.textContent = `${fmFiles.length} file${fmFiles.length !== 1 ? 's' : ''} selected`;
  submitBtn.disabled = fmFiles.length === 0;

  if (fmFiles.length === 0) {
    list.innerHTML = '<div class="fm-empty">No files yet — click "Add Files" to start</div>';
    return;
  }

  list.innerHTML = fmFiles.map((item, idx) => {
    const f = item.file;
    const sizeKB = (f.size / 1024).toFixed(1);
    const isImg = f.type.startsWith('image/');
    return `<div class="fm-item" draggable="true" data-idx="${idx}" data-id="${item.id}">
      <span class="fm-drag">⠿</span>
      <span class="fm-num">${idx + 1}</span>
      ${isImg ? `<img class="fm-thumb" src="" data-file-id="${item.id}">` : '<span class="fm-thumb-icon">📄</span>'}
      <span class="fm-name">${f.name}</span>
      <span class="fm-size">${sizeKB} KB</span>
      <button class="fm-remove" onclick="fmRemoveFile(${item.id})">✕</button>
    </div>`;
  }).join('');

  // Load image thumbnails
  list.querySelectorAll('img.fm-thumb').forEach(img => {
    const item = fmFiles.find(f => f.id === parseInt(img.dataset.fileId));
    if (item) {
      const reader = new FileReader();
      reader.onload = e => img.src = e.target.result;
      reader.readAsDataURL(item.file);
    }
  });

  // Wire up drag-to-reorder
  list.querySelectorAll('.fm-item').forEach(el => {
    el.addEventListener('dragstart', e => {
      e.dataTransfer.setData('text/plain', el.dataset.idx);
      el.classList.add('fm-dragging');
    });
    el.addEventListener('dragend', () => el.classList.remove('fm-dragging'));
    el.addEventListener('dragover', e => { e.preventDefault(); el.classList.add('fm-dragover'); });
    el.addEventListener('dragleave', () => el.classList.remove('fm-dragover'));
    el.addEventListener('drop', e => {
      e.preventDefault();
      el.classList.remove('fm-dragover');
      const fromIdx = parseInt(e.dataTransfer.getData('text/plain'));
      const toIdx = parseInt(el.dataset.idx);
      if (fromIdx !== toIdx) {
        const [moved] = fmFiles.splice(fromIdx, 1);
        fmFiles.splice(toIdx, 0, moved);
        renderFmList();
      }
    });
  });
}

function fmSubmit() {
  if (fmCallback && fmFiles.length > 0) {
    // Create a fake file list from managed files
    const dt = new DataTransfer();
    fmFiles.forEach(item => dt.items.add(item.file));
    fmCallback(dt.files);
  }
  closeFileManager();
}

/* ── Interactive Page Editing Helpers ─────────────────────── */

async function fetchPageCount(file) {
  const fd = new FormData();
  fd.append('file', file);
  try {
    const res = await fetch('/api/page-count', { method: 'POST', body: fd });
    const data = await res.json();
    return data.pages || 0;
  } catch {
    return 0;
  }
}

// Delete Pages — interactive page grid
async function initDeletePages(inputId) {
  const files = getFiles(inputId);
  if (!files || files.length === 0) { toast('Please select a PDF first'); return; }
  const container = document.getElementById('delpg-grid');
  container.innerHTML = '<div class="page-grid-loading">Analyzing PDF...</div>';
  container.style.display = 'block';
  const count = await fetchPageCount(files[0]);
  if (count === 0) { container.innerHTML = '<div class="page-grid-loading">Could not read pages</div>'; return; }

  let html = '<div class="page-grid">';
  for (let i = 1; i <= count; i++) {
    html += `<label class="page-chip">
      <input type="checkbox" value="${i}" onchange="updateDeletePagesValue()">
      <span>${i}</span>
    </label>`;
  }
  html += '</div><div class="page-grid-hint">Click pages to select them for deletion</div>';
  container.innerHTML = html;
}

function updateDeletePagesValue() {
  const checked = Array.from(document.querySelectorAll('#delpg-grid input:checked')).map(cb => cb.value);
  document.getElementById('delpg-pages').value = checked.join(',');
}

// Reorder Pages — interactive drag pills
async function initReorderPages(inputId) {
  const files = getFiles(inputId);
  if (!files || files.length === 0) { toast('Please select a PDF first'); return; }
  const container = document.getElementById('reorder-grid');
  container.innerHTML = '<div class="page-grid-loading">Analyzing PDF...</div>';
  container.style.display = 'block';
  const count = await fetchPageCount(files[0]);
  if (count === 0) { container.innerHTML = '<div class="page-grid-loading">Could not read pages</div>'; return; }

  const pages = Array.from({ length: count }, (_, i) => i + 1);
  renderReorderGrid(container, pages);
}

function renderReorderGrid(container, pages) {
  let html = '<div class="page-grid reorder-grid">';
  pages.forEach((p, idx) => {
    html += `<div class="page-pill" draggable="true" data-idx="${idx}" data-page="${p}">
      <span class="page-pill-drag">⠿</span> Page ${p}
    </div>`;
  });
  html += '</div><div class="page-grid-hint">Drag pages to reorder them</div>';
  container.innerHTML = html;

  // Wire drag
  container.querySelectorAll('.page-pill').forEach(el => {
    el.addEventListener('dragstart', e => {
      e.dataTransfer.setData('text/plain', el.dataset.idx);
      el.classList.add('fm-dragging');
    });
    el.addEventListener('dragend', () => el.classList.remove('fm-dragging'));
    el.addEventListener('dragover', e => { e.preventDefault(); el.classList.add('fm-dragover'); });
    el.addEventListener('dragleave', () => el.classList.remove('fm-dragover'));
    el.addEventListener('drop', e => {
      e.preventDefault();
      el.classList.remove('fm-dragover');
      const fromIdx = parseInt(e.dataTransfer.getData('text/plain'));
      const toIdx = parseInt(el.dataset.idx);
      if (fromIdx !== toIdx) {
        const [moved] = pages.splice(fromIdx, 1);
        pages.splice(toIdx, 0, moved);
        renderReorderGrid(container, pages);
        document.getElementById('reorder-order').value = pages.join(',');
      }
    });
  });

  document.getElementById('reorder-order').value = pages.join(',');
}

// Insert Blank — visual page list with + buttons
async function initInsertBlank(inputId) {
  const files = getFiles(inputId);
  if (!files || files.length === 0) { toast('Please select a PDF first'); return; }
  const container = document.getElementById('insertblank-grid');
  container.innerHTML = '<div class="page-grid-loading">Analyzing PDF...</div>';
  container.style.display = 'block';
  const count = await fetchPageCount(files[0]);
  if (count === 0) { container.innerHTML = '<div class="page-grid-loading">Could not read pages</div>'; return; }

  let insertAfter = new Set();
  let html = '<div class="insert-grid">';
  for (let i = 1; i <= count; i++) {
    html += `<div class="insert-page">Page ${i}</div>`;
    html += `<button class="insert-btn" data-after="${i}" onclick="toggleInsertAfter(this, ${i})">+ Insert blank</button>`;
  }
  html += '</div><div class="page-grid-hint">Click "+" buttons where you want blank pages inserted</div>';
  container.innerHTML = html;
  window._insertAfterSet = new Set();
}

function toggleInsertAfter(btn, pageNum) {
  if (window._insertAfterSet.has(pageNum)) {
    window._insertAfterSet.delete(pageNum);
    btn.classList.remove('active');
  } else {
    window._insertAfterSet.add(pageNum);
    btn.classList.add('active');
  }
  document.getElementById('insertblank-after').value = Array.from(window._insertAfterSet).sort((a,b)=>a-b).join(',');
}

// Crop — percentage sliders
function initCropSliders() {
  const container = document.getElementById('crop-sliders');
  container.style.display = 'block';
  container.innerHTML = `
    <div class="crop-slider-row">
      <label>Top margin</label>
      <input type="range" min="0" max="50" value="0" oninput="updateCropBox()" id="crop-top">
      <span class="range-val" id="crop-top-val">0%</span>
    </div>
    <div class="crop-slider-row">
      <label>Bottom margin</label>
      <input type="range" min="0" max="50" value="0" oninput="updateCropBox()" id="crop-bottom">
      <span class="range-val" id="crop-bottom-val">0%</span>
    </div>
    <div class="crop-slider-row">
      <label>Left margin</label>
      <input type="range" min="0" max="50" value="0" oninput="updateCropBox()" id="crop-left">
      <span class="range-val" id="crop-left-val">0%</span>
    </div>
    <div class="crop-slider-row">
      <label>Right margin</label>
      <input type="range" min="0" max="50" value="0" oninput="updateCropBox()" id="crop-right">
      <span class="range-val" id="crop-right-val">0%</span>
    </div>
    <div class="crop-preview-box" id="crop-preview">
      <div class="crop-preview-inner" id="crop-preview-inner"></div>
    </div>
  `;
  updateCropBox();
}

function updateCropBox() {
  const top = parseInt(document.getElementById('crop-top').value);
  const bottom = parseInt(document.getElementById('crop-bottom').value);
  const left = parseInt(document.getElementById('crop-left').value);
  const right = parseInt(document.getElementById('crop-right').value);

  document.getElementById('crop-top-val').textContent = top + '%';
  document.getElementById('crop-bottom-val').textContent = bottom + '%';
  document.getElementById('crop-left-val').textContent = left + '%';
  document.getElementById('crop-right-val').textContent = right + '%';

  // Update visual preview
  const inner = document.getElementById('crop-preview-inner');
  if (inner) {
    inner.style.top = top + '%';
    inner.style.bottom = bottom + '%';
    inner.style.left = left + '%';
    inner.style.right = right + '%';
  }

  // Convert to approximate points (A4 = 595 x 842)
  const w = 595, h = 842;
  const x1 = Math.round(w * left / 100);
  const y1 = Math.round(h * bottom / 100);
  const x2 = Math.round(w * (100 - right) / 100);
  const y2 = Math.round(h * (100 - top) / 100);
  document.getElementById('crop-box').value = `[${x1} ${y1} ${x2} ${y2}]`;
}

// Fill Form — auto-detect fields
async function initFillForm(inputId) {
  const files = getFiles(inputId);
  if (!files || files.length === 0) { toast('Please select a PDF first'); return; }
  const container = document.getElementById('fillform-fields-grid');
  container.innerHTML = '<div class="page-grid-loading">Detecting form fields...</div>';
  container.style.display = 'block';

  const fd = new FormData();
  fd.append('file', files[0]);
  try {
    const res = await fetch('/api/form-fields', { method: 'POST', body: fd });
    const data = await res.json();
    if (!data.fields || data.fields.length === 0) {
      container.innerHTML = '<div class="page-grid-hint">No fillable form fields found in this PDF</div>';
      return;
    }
    let html = '<div class="form-fields-auto">';
    data.fields.forEach(field => {
      const name = field.name || field.Name || 'unknown';
      const val  = field.value || field.Value || '';
      html += `<div class="form-field-row">
        <label class="form-field-label">${name}</label>
        <input type="text" class="field-input form-field-input" data-field-name="${name}" value="${val}" placeholder="Enter value...">
      </div>`;
    });
    html += '</div>';
    container.innerHTML = html;
  } catch (err) {
    container.innerHTML = `<div class="page-grid-hint">Error: ${err.message}</div>`;
  }
}

function collectFormFields() {
  const inputs = document.querySelectorAll('#fillform-fields-grid .form-field-input');
  const pairs = [];
  inputs.forEach(inp => {
    if (inp.value.trim()) {
      pairs.push(`${inp.dataset.fieldName}=${inp.value.trim()}`);
    }
  });
  document.getElementById('fillform-fields').value = pairs.join('; ');
}