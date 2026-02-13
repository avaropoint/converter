package main

// indexHTML is the embedded web interface served at /.
const indexHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>Converter</title>
<style>
  *, *::before, *::after { box-sizing: border-box; margin: 0; padding: 0; }

  :root {
    --bg: #fafbfc;
    --surface: #ffffff;
    --surface-hover: #f8f9fb;
    --border: #e8ecf0;
    --border-light: #f0f2f5;
    --text: #1a2332;
    --text-secondary: #6b7a8d;
    --text-muted: #9ba8b7;
    --accent: #3b82f6;
    --accent-hover: #2563eb;
    --accent-light: #eff6ff;
    --accent-glow: rgba(59, 130, 246, 0.12);
    --success: #10b981;
    --error: #ef4444;
    --radius: 14px;
    --radius-sm: 8px;
    --shadow-sm: 0 1px 2px rgba(0,0,0,0.04);
    --shadow: 0 2px 8px rgba(0,0,0,0.06), 0 0 1px rgba(0,0,0,0.08);
    --shadow-lg: 0 8px 24px rgba(0,0,0,0.08), 0 0 1px rgba(0,0,0,0.08);
    --transition: 0.2s cubic-bezier(0.4, 0, 0.2, 1);
  }

  body {
    font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Inter, Roboto, Helvetica, Arial, sans-serif;
    background: var(--bg);
    color: var(--text);
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 3rem 1.25rem 2rem;
    -webkit-font-smoothing: antialiased;
  }

  .brand {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin-bottom: 0.375rem;
  }

  .brand-icon {
    width: 32px;
    height: 32px;
    background: var(--accent);
    border-radius: 9px;
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .brand-icon svg {
    width: 18px;
    height: 18px;
    fill: none;
    stroke: #fff;
    stroke-width: 2;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  h1 {
    font-size: 1.375rem;
    font-weight: 700;
    letter-spacing: -0.02em;
    color: var(--text);
  }

  .subtitle {
    color: var(--text-muted);
    font-size: 0.875rem;
    margin-bottom: 2.25rem;
    text-align: center;
  }

  .container {
    width: 100%;
    max-width: 580px;
  }

  /* Drop zone */
  .dropzone {
    position: relative;
    border: 2px dashed var(--border);
    border-radius: var(--radius);
    padding: 2.75rem 2rem;
    text-align: center;
    cursor: pointer;
    transition: all var(--transition);
    background: var(--surface);
    box-shadow: var(--shadow-sm);
  }

  .dropzone:hover {
    border-color: #c8d3e0;
    box-shadow: var(--shadow);
  }

  .dropzone.active {
    border-color: var(--accent);
    border-style: solid;
    background: var(--accent-light);
    box-shadow: 0 0 0 4px var(--accent-glow), var(--shadow);
  }

  .dropzone-graphic {
    width: 56px;
    height: 56px;
    margin: 0 auto 1rem;
    background: var(--accent-light);
    border-radius: 14px;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: transform var(--transition);
  }

  .dropzone:hover .dropzone-graphic,
  .dropzone.active .dropzone-graphic {
    transform: scale(1.05);
  }

  .dropzone-graphic svg {
    width: 26px;
    height: 26px;
    stroke: var(--accent);
    fill: none;
    stroke-width: 1.75;
    stroke-linecap: round;
    stroke-linejoin: round;
  }

  .dropzone-text {
    font-size: 0.9375rem;
    font-weight: 500;
    color: var(--text-secondary);
    margin-bottom: 0.25rem;
  }

  .dropzone-hint {
    font-size: 0.8125rem;
    color: var(--text-muted);
  }

  .dropzone-hint .browse {
    color: var(--accent);
    font-weight: 500;
    cursor: pointer;
    transition: color var(--transition);
  }

  .dropzone-hint .browse:hover {
    color: var(--accent-hover);
  }

  #fileInput { display: none; }

  /* Status */
  .status {
    margin-top: 1.25rem;
    text-align: center;
    font-size: 0.875rem;
    color: var(--text-secondary);
    min-height: 1.5rem;
  }

  .status.error {
    color: var(--error);
    font-weight: 500;
  }

  .spinner {
    display: inline-block;
    width: 14px;
    height: 14px;
    border: 2px solid var(--border);
    border-top-color: var(--accent);
    border-radius: 50%;
    animation: spin 0.65s linear infinite;
    vertical-align: -1px;
    margin-right: 0.4rem;
  }

  @keyframes spin { to { transform: rotate(360deg); } }

  /* Results  */
  .results {
    margin-top: 1.25rem;
    background: var(--surface);
    border-radius: var(--radius);
    border: 1px solid var(--border);
    box-shadow: var(--shadow);
    overflow: hidden;
    animation: slideIn 0.3s ease-out;
  }

  @keyframes slideIn {
    from { opacity: 0; transform: translateY(8px); }
    to   { opacity: 1; transform: translateY(0); }
  }

  .results-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.875rem 1.125rem;
    border-bottom: 1px solid var(--border-light);
  }

  .results-header h2 {
    font-size: 0.875rem;
    font-weight: 600;
    color: var(--text);
  }

  .results-header .file-count {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    background: var(--accent);
    color: #fff;
    font-size: 0.6875rem;
    font-weight: 700;
    width: 20px;
    height: 20px;
    border-radius: 6px;
    margin-left: 0.4rem;
    vertical-align: middle;
  }

  .download-all {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    padding: 0.4rem 0.75rem;
    font-size: 0.8125rem;
    font-weight: 500;
    background: var(--accent);
    color: #fff;
    border: none;
    border-radius: var(--radius-sm);
    cursor: pointer;
    text-decoration: none;
    transition: all var(--transition);
    box-shadow: 0 1px 3px rgba(59, 130, 246, 0.3);
  }

  .download-all:hover {
    background: var(--accent-hover);
    box-shadow: 0 2px 6px rgba(59, 130, 246, 0.35);
    transform: translateY(-0.5px);
  }

  .download-all svg {
    width: 13px;
    height: 13px;
    stroke: currentColor;
    fill: none;
    stroke-width: 2;
    stroke-linecap: round;
  }

  /* File list */
  .file-list { list-style: none; }

  .file-list li {
    display: flex;
    align-items: center;
    padding: 0.75rem 1.125rem;
    border-bottom: 1px solid var(--border-light);
    transition: background var(--transition);
  }

  .file-list li:last-child { border-bottom: none; }
  .file-list li:hover { background: var(--surface-hover); }

  .file-icon {
    width: 32px;
    height: 32px;
    border-radius: var(--radius-sm);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.625rem;
    font-weight: 700;
    letter-spacing: 0.03em;
    color: #fff;
    margin-right: 0.75rem;
    flex-shrink: 0;
    text-transform: uppercase;
  }

  .file-icon.html  { background: linear-gradient(135deg, #f97316, #ea580c); }
  .file-icon.text  { background: linear-gradient(135deg, #94a3b8, #64748b); }
  .file-icon.rtf   { background: linear-gradient(135deg, #a855f7, #7c3aed); }
  .file-icon.image { background: linear-gradient(135deg, #10b981, #059669); }
  .file-icon.pdf   { background: linear-gradient(135deg, #ef4444, #dc2626); }
  .file-icon.file  { background: linear-gradient(135deg, #94a3b8, #64748b); }
  .file-icon.document    { background: linear-gradient(135deg, #3b82f6, #2563eb); }
  .file-icon.spreadsheet { background: linear-gradient(135deg, #10b981, #059669); }

  .file-info { flex: 1; min-width: 0; }

  .file-name {
    display: block;
    font-size: 0.875rem;
    font-weight: 500;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--text);
  }

  .file-size {
    font-size: 0.75rem;
    color: var(--text-muted);
    margin-top: 1px;
  }

  .file-actions {
    display: flex;
    gap: 0.375rem;
    flex-shrink: 0;
    margin-left: 0.5rem;
    opacity: 0.5;
    transition: opacity var(--transition);
  }

  .file-list li:hover .file-actions { opacity: 1; }

  .file-actions a {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
    padding: 0.3rem 0.5rem;
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--text-secondary);
    text-decoration: none;
    border: 1px solid var(--border);
    border-radius: 6px;
    transition: all var(--transition);
    white-space: nowrap;
  }

  .file-actions a:hover {
    color: var(--accent);
    border-color: var(--accent);
    background: var(--accent-light);
  }

  .file-actions a svg {
    width: 12px;
    height: 12px;
    stroke: currentColor;
    fill: none;
    stroke-width: 2;
    stroke-linecap: round;
  }

  /* Footer */
  footer {
    margin-top: 2.5rem;
    font-size: 0.75rem;
    color: var(--text-muted);
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }

  footer .dot {
    width: 3px;
    height: 3px;
    border-radius: 50%;
    background: var(--text-muted);
    opacity: 0.5;
  }

  /* Supported formats badge */
  .formats-badge {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    margin-top: 1.25rem;
    padding: 0.35rem 0.75rem;
    font-size: 0.75rem;
    color: var(--text-muted);
    background: var(--surface);
    border: 1px solid var(--border-light);
    border-radius: 100px;
  }

  .formats-badge .tag {
    font-weight: 600;
    font-size: 0.6875rem;
    color: var(--accent);
    background: var(--accent-light);
    padding: 0.125rem 0.4rem;
    border-radius: 4px;
  }

  /* Reset button */
  .reset-btn {
    display: none;
    margin-top: 0.75rem;
    padding: 0.4rem 0.85rem;
    font-size: 0.8125rem;
    font-weight: 500;
    color: var(--text-secondary);
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    cursor: pointer;
    transition: all var(--transition);
    text-align: center;
  }

  .reset-btn:hover {
    color: var(--text);
    border-color: #c8d3e0;
    box-shadow: var(--shadow-sm);
  }

  @media (max-width: 480px) {
    body { padding: 1.5rem 1rem; }
    .dropzone { padding: 2rem 1.25rem; }
    .file-actions { opacity: 1; }
  }
</style>
</head>
<body>

<div class="brand">
  <div class="brand-icon">
    <svg viewBox="0 0 24 24"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8Z"/><polyline points="14 2 14 8 20 8"/><path d="m9 15 2 2 4-4"/></svg>
  </div>
  <h1>Converter</h1>
</div>
<p class="subtitle">Drop a file to convert and extract its contents</p>

<div class="container">
  <div class="dropzone" id="dropzone">
    <div class="dropzone-graphic">
      <svg viewBox="0 0 24 24"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="17 8 12 3 7 8"/><line x1="12" y1="3" x2="12" y2="15"/></svg>
    </div>
    <p class="dropzone-text">Drop your file here</p>
    <p class="dropzone-hint">or <span class="browse" id="browseBtn">browse to select</span></p>
    <input type="file" id="fileInput">
  </div>

  <div style="text-align:center">
    <div class="formats-badge">
      Supported <span class="tag">winmail.dat</span> <span class="tag">TNEF</span>
    </div>
  </div>

  <div class="status" id="status"></div>

  <div class="results" id="results" style="display:none">
    <div class="results-header">
      <h2>Extracted <span class="file-count" id="fileCount">0</span></h2>
      <a class="download-all" id="downloadAll" href="#">
        <svg viewBox="0 0 24 24"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>
        Download All
      </a>
    </div>
    <ul class="file-list" id="fileList"></ul>
  </div>

  <button class="reset-btn" id="resetBtn">Convert another file</button>
</div>

<footer>
  <span>converter v` + version + `</span>
  <span class="dot"></span>
  <span>Files are processed in memory and auto-deleted after 10 min</span>
</footer>

<script>
(function() {
  const dropzone = document.getElementById('dropzone');
  const fileInput = document.getElementById('fileInput');
  const browseBtn = document.getElementById('browseBtn');
  const statusEl = document.getElementById('status');
  const resultsEl = document.getElementById('results');
  const fileListEl = document.getElementById('fileList');
  const fileCount = document.getElementById('fileCount');
  const downloadAll = document.getElementById('downloadAll');
  const resetBtn = document.getElementById('resetBtn');

  browseBtn.addEventListener('click', e => { e.stopPropagation(); fileInput.click(); });
  dropzone.addEventListener('click', () => fileInput.click());

  ['dragenter', 'dragover'].forEach(e => {
    dropzone.addEventListener(e, ev => { ev.preventDefault(); dropzone.classList.add('active'); });
  });
  ['dragleave', 'drop'].forEach(e => {
    dropzone.addEventListener(e, ev => { ev.preventDefault(); dropzone.classList.remove('active'); });
  });

  dropzone.addEventListener('drop', ev => {
    if (ev.dataTransfer.files.length > 0) upload(ev.dataTransfer.files[0]);
  });

  fileInput.addEventListener('change', () => {
    if (fileInput.files.length > 0) upload(fileInput.files[0]);
  });

  resetBtn.addEventListener('click', () => {
    resultsEl.style.display = 'none';
    resetBtn.style.display = 'none';
    dropzone.style.display = '';
    document.querySelector('.formats-badge').style.display = '';
    statusEl.textContent = '';
    fileInput.value = '';
  });

  function upload(file) {
    statusEl.className = 'status';
    statusEl.innerHTML = '<span class="spinner"></span>Converting ' + escHtml(file.name) + '...';
    resultsEl.style.display = 'none';
    resetBtn.style.display = 'none';

    const form = new FormData();
    form.append('file', file);

    fetch('/api/convert', { method: 'POST', body: form })
      .then(resp => resp.json().then(data => ({ ok: resp.ok, data })))
      .then(({ ok, data }) => {
        if (!ok) {
          statusEl.className = 'status error';
          statusEl.textContent = data.error || 'Conversion failed';
          return;
        }
        statusEl.textContent = '';
        dropzone.style.display = 'none';
        document.querySelector('.formats-badge').style.display = 'none';
        showResults(data);
      })
      .catch(() => {
        statusEl.className = 'status error';
        statusEl.textContent = 'Connection error';
      });
  }

  function showResults(data) {
    const sid = data.sessionId;
    const files = data.files;

    fileCount.textContent = files.length;
    downloadAll.href = '/api/zip/' + sid;
    fileListEl.innerHTML = '';

    files.forEach(f => {
      const li = document.createElement('li');
      const fileUrl = '/api/files/' + sid + '/' + encodeURIComponent(f.name);

      li.innerHTML =
        '<div class="file-icon ' + escAttr(f.type) + '">' + escHtml(iconLabel(f.type)) + '</div>' +
        '<div class="file-info">' +
          '<span class="file-name" title="' + escAttr(f.name) + '">' + escHtml(f.name) + '</span>' +
          '<span class="file-size">' + humanSize(f.size) + '</span>' +
        '</div>' +
        '<div class="file-actions">' +
          '<a href="' + fileUrl + '" target="_blank">' +
            '<svg viewBox="0 0 24 24"><path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/><circle cx="12" cy="12" r="3"/></svg>' +
            'View' +
          '</a>' +
          '<a href="' + fileUrl + '" download="' + escAttr(f.name) + '">' +
            '<svg viewBox="0 0 24 24"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>' +
            'Save' +
          '</a>' +
        '</div>';

      fileListEl.appendChild(li);
    });

    resultsEl.style.display = 'block';
    resetBtn.style.display = 'block';
  }

  function iconLabel(type) {
    return { html:'HTML', text:'TXT', rtf:'RTF', image:'IMG', pdf:'PDF', document:'DOC', spreadsheet:'XLS', file:'FILE' }[type] || 'FILE';
  }

  function humanSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    const u = ['KB','MB','GB'];
    let i = -1, s = bytes;
    do { s /= 1024; i++; } while (s >= 1024 && i < u.length - 1);
    return s.toFixed(1) + ' ' + u[i];
  }

  function escHtml(s) { const d = document.createElement('div'); d.textContent = s; return d.innerHTML; }
  function escAttr(s) { return s.replace(/&/g,'&amp;').replace(/"/g,'&quot;').replace(/</g,'&lt;').replace(/>/g,'&gt;'); }
})();
</script>
</body>
</html>`
