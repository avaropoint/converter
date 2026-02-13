(function () {
  'use strict';

  // DOM elements
  const dropzone = document.getElementById('dropzone');
  const fileInput = document.getElementById('fileInput');
  const browseBtn = document.getElementById('browseBtn');
  const statusEl = document.getElementById('status');
  const resultsEl = document.getElementById('results');
  const fileListEl = document.getElementById('fileList');
  const fileCount = document.getElementById('fileCount');
  const downloadAll = document.getElementById('downloadAll');
  const resetBtn = document.getElementById('resetBtn');
  const versionLabel = document.getElementById('versionLabel');
  const successEl = document.getElementById('successState');
  const formatsBadge = document.querySelector('.formats-badge');

  // Fetch version from server and display in footer
  fetch('api/info')
    .then(function (r) { return r.json(); })
    .then(function (data) {
      if (data.version) {
        versionLabel.textContent = 'converter v' + data.version;
      }
    })
    .catch(function () {
      // Silently ignore — footer already says "converter"
    });

  // File selection
  browseBtn.addEventListener('click', function (e) {
    e.stopPropagation();
    fileInput.click();
  });

  dropzone.addEventListener('click', function () {
    fileInput.click();
  });

  // Drag and drop
  ['dragenter', 'dragover'].forEach(function (evt) {
    dropzone.addEventListener(evt, function (e) {
      e.preventDefault();
      dropzone.classList.add('active');
    });
  });

  ['dragleave', 'drop'].forEach(function (evt) {
    dropzone.addEventListener(evt, function (e) {
      e.preventDefault();
      dropzone.classList.remove('active');
    });
  });

  dropzone.addEventListener('drop', function (e) {
    if (e.dataTransfer.files.length > 0) {
      upload(e.dataTransfer.files[0]);
    }
  });

  fileInput.addEventListener('change', function () {
    if (fileInput.files.length > 0) {
      upload(fileInput.files[0]);
    }
  });

  // Reset
  resetBtn.addEventListener('click', function () {
    resultsEl.style.display = 'none';
    successEl.style.display = 'none';
    resetBtn.style.display = 'none';
    dropzone.style.display = '';
    formatsBadge.style.display = '';
    statusEl.textContent = '';
    fileInput.value = '';
  });

  /**
   * Upload a file to the conversion API.
   * @param {File} file
   */
  function upload(file) {
    statusEl.className = 'status';
    statusEl.innerHTML =
      '<span class="spinner"></span>Converting ' + escHtml(file.name) + '…';
    resultsEl.style.display = 'none';
    resetBtn.style.display = 'none';

    var form = new FormData();
    form.append('file', file);

    fetch('api/convert', { method: 'POST', body: form })
      .then(function (resp) {
        return resp.json().then(function (data) {
          return { ok: resp.ok, data: data };
        });
      })
      .then(function (result) {
        if (!result.ok) {
          statusEl.className = 'status error';
          statusEl.textContent = result.data.error || 'Conversion failed';
          return;
        }
        statusEl.textContent = '';
        dropzone.style.display = 'none';
        formatsBadge.style.display = 'none';
        showSuccess(result.data);
      })
      .catch(function () {
        statusEl.className = 'status error';
        statusEl.textContent = 'Connection error';
      });
  }

  /**
   * Show a brief success animation, then render the file list.
   * @param {Object} data - Response from /api/convert
   */
  function showSuccess(data) {
    successEl.style.display = 'flex';
    setTimeout(function () {
      successEl.style.display = 'none';
      showResults(data);
    }, 700);
  }

  /**
   * Render the extracted file list.
   * @param {Object} data - Response from /api/convert
   */
  function showResults(data) {
    var sid = data.sessionToken;
    var files = data.files;

    fileCount.textContent = files.length;
    downloadAll.href = 'api/zip/' + sid;
    fileListEl.innerHTML = '';

    files.forEach(function (f, i) {
      var li = document.createElement('li');
      li.style.animationDelay = (i * 50) + 'ms';
      var fileUrl = 'api/files/' + sid + '/' + encodeURIComponent(f.name);

      li.innerHTML =
        '<div class="file-icon ' + escAttr(f.type) + '">' +
          escHtml(iconLabel(f.type)) +
        '</div>' +
        '<div class="file-info">' +
          '<span class="file-name" title="' + escAttr(f.name) + '">' +
            escHtml(f.name) +
          '</span>' +
          '<span class="file-size">' + humanSize(f.size) + '</span>' +
        '</div>' +
        '<div class="file-actions">' +
          '<a href="' + fileUrl + '" target="_blank">' +
            '<svg viewBox="0 0 24 24">' +
              '<path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z"/>' +
              '<circle cx="12" cy="12" r="3"/>' +
            '</svg>' +
            'View' +
          '</a>' +
          '<a href="' + fileUrl + '" download="' + escAttr(f.name) + '">' +
            '<svg viewBox="0 0 24 24">' +
              '<path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>' +
              '<polyline points="7 10 12 15 17 10"/>' +
              '<line x1="12" y1="15" x2="12" y2="3"/>' +
            '</svg>' +
            'Save' +
          '</a>' +
        '</div>';

      fileListEl.appendChild(li);
    });

    resultsEl.style.display = 'block';
    resetBtn.style.display = 'block';
  }

  // --- Helpers ---

  var typeLabels = {
    html: 'HTML',
    text: 'TXT',
    rtf: 'RTF',
    image: 'IMG',
    pdf: 'PDF',
    document: 'DOC',
    spreadsheet: 'XLS',
    file: 'FILE'
  };

  function iconLabel(type) {
    return typeLabels[type] || 'FILE';
  }

  function humanSize(bytes) {
    if (bytes < 1024) return bytes + ' B';
    var units = ['KB', 'MB', 'GB'];
    var i = -1;
    var size = bytes;
    do {
      size /= 1024;
      i++;
    } while (size >= 1024 && i < units.length - 1);
    return size.toFixed(1) + ' ' + units[i];
  }

  function escHtml(s) {
    var d = document.createElement('div');
    d.textContent = s;
    return d.innerHTML;
  }

  function escAttr(s) {
    return s
      .replace(/&/g, '&amp;')
      .replace(/"/g, '&quot;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;');
  }
})();
