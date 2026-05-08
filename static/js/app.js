document.querySelectorAll('.tool-card[data-accent]').forEach(card => {
  const c = card.dataset.accent;
  card.style.setProperty('--card-a', c);
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
  // Attach dropped files to the input element as a custom property
  input._files = e.dataTransfer.files;
  renderFileList(input._files, el);
}
function onFileChange(input, dzId) {
  input._files = null; // use native .files
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
  // Highlight drop zone border
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
    const next = (data.newSize     / 1024).toFixed(1);
    meta = `<div class="result-meta">${orig} KB → ${next} KB &nbsp;·&nbsp; saved ~${data.savings}%</div>`;
  }
  if (data.pages) {
    meta += `<div class="result-meta">${data.pages} pages</div>`;
  }

  const dl = (success && data.download)
    ? `<a class="dl-btn" href="${data.download}" download="${data.filename}">↓ Download ${data.filename}</a>`
    : '';

  box.innerHTML = `
    <div class="result-inner ${cls}">
      <div class="result-msg">${icon} ${msg}</div>
      ${meta}${dl}
    </div>`;
  box.classList.add('show');
}

function toast(msg) {
  const el = document.getElementById('toast');
  el.textContent = msg;
  el.classList.add('show');
  setTimeout(() => el.classList.remove('show'), 3000);
}