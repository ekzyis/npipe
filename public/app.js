const icons = {
  upload: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
      <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
      <polyline points="17 8 12 3 7 8"/>
      <line x1="12" y1="3" x2="12" y2="15"/>
    </svg>`,
  spinner: `
    <svg class="spinner" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="12" cy="12" r="10" stroke-opacity="0.25"/>
      <path d="M12 2a10 10 0 0 1 10 10"/>
    </svg>`,
  check: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <polyline points="20 6 9 17 4 12"/>
    </svg>`,
  error: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <line x1="18" y1="6" x2="6" y2="18"/>
      <line x1="6" y1="6" x2="18" y2="18"/>
    </svg>`,
  file: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
      <polyline points="14 2 14 8 20 8"/>
    </svg>`
};

const knownFiles = new Map();

const statusText = {
  400: "Bad Request",
  401: "Unauthorized",
  403: "Forbidden",
  404: "Not Found",
  413: "Content Too Large",
  500: "Internal Server Error",
  502: "Bad Gateway",
  503: "Service Unavailable"
};

function setupUpload() {
  document.getElementById("f").onchange = (e) => {
    const file = e.target.files[0];
    const reader = new FileReader();

    reader.onload = () => {
      const data = Array.from(new Uint8Array(reader.result));
      const upload = document.getElementById("upload");
      const label = upload.querySelector("label");
      const input = document.getElementById("f");
      const origHTML = label.innerHTML;

      input.disabled = true;
      upload.classList.add("uploading");
      upload.style.setProperty("--progress", 0);
      label.innerHTML = `
        ${icons.spinner}
        <span>uploading</span>
      `;

      const xhr = new XMLHttpRequest();
      xhr.open("POST", "/file");

      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) {
          const progress = (e.loaded / e.total) * 100;
          console.log("uploading:", progress.toFixed(1) + "%");
          upload.style.setProperty("--progress", progress);
        }
      };

      xhr.onload = () => {
        input.value = "";
        upload.classList.remove("uploading");
        if (xhr.status >= 200 && xhr.status < 300) {
          pollFiles();
          upload.classList.add("success");
          label.innerHTML = `
            <span class="success">${icons.check}</span>
            <span class="success">uploaded</span>
          `;
          setTimeout(() => {
            upload.classList.remove("success");
            label.innerHTML = origHTML;
            input.disabled = false;
          }, 1000);
        } else {
          label.innerHTML = `
            <span class="error">${icons.error}</span>
            <span class="error">${statusText[xhr.status] || xhr.status}</span>
          `;
          setTimeout(() => {
            label.innerHTML = origHTML;
            input.disabled = false;
          }, 3000);
        }
      };

      xhr.onerror = () => {
        input.value = "";
        upload.classList.remove("uploading");
        label.innerHTML = `
          <span class="error">${icons.error}</span>
          <span class="error">network error</span>
        `;
        setTimeout(() => {
          label.innerHTML = origHTML;
          input.disabled = false;
        }, 3000);
      };

      xhr.send(JSON.stringify({ name: file.name, data }));
    };

    reader.readAsArrayBuffer(file);
  };
}

function addFile(id, name, createdAt, expiresAt) {
  const div = document.createElement("div");
  div.className = "card file";
  div.dataset.id = id;
  div.dataset.name = name;
  div.innerHTML = `
    <a href="#">
      ${icons.file}
      <span title="${name}">${name}</span>
    </a>
  `;
  document.getElementById("files").appendChild(div);
  const created = new Date(createdAt).getTime();
  const expires = new Date(expiresAt).getTime();
  knownFiles.set(id, { created, expires });
  setProgress(div, created, expires);
}

function setProgress(card, createdAt, expiresAt) {
  const ttl = expiresAt - createdAt;
  const remaining = Math.max(0, expiresAt - Date.now());
  const progress = (remaining / ttl) * 100;
  card.style.setProperty('--progress', progress);
}

function removeFile(id) {
  const card = document.querySelector(`.file[data-id="${id}"]`);
  if (card) card.remove();
  knownFiles.delete(id);
}

function downloadFile(card, id, name) {
  card.classList.add("downloading");
  card.style.setProperty("--progress", 0);

  const xhr = new XMLHttpRequest();
  xhr.open("GET", "/file/" + id);
  xhr.responseType = "blob";

  xhr.onprogress = (e) => {
    if (e.lengthComputable) {
      const progress = (e.loaded / e.total) * 100;
      console.log("downloading:", id, progress.toFixed(1) + "%");
      card.style.setProperty("--progress", progress);
    }
  };

  xhr.onload = () => {
    card.classList.remove("downloading");
    if (xhr.status >= 200 && xhr.status < 300) {
      const url = URL.createObjectURL(xhr.response);
      const a = document.createElement("a");
      a.href = url;
      a.download = name;
      a.click();
    }
  };

  xhr.onerror = () => {
    card.classList.remove("downloading");
  };

  xhr.send();
}

function pollFiles() {
  fetch("/files")
    .then((res) => res.json())
    .then((files) => {
      const serverIds = new Set(files.map((f) => f.id));

      // remove expired files
      for (const id of knownFiles.keys()) {
        if (!serverIds.has(id)) {
          removeFile(id);
        }
      }

      // add new files and update progress
      files.forEach((f) => {
        if (!knownFiles.has(f.id)) {
          addFile(f.id, f.name, f.createdAt, f.expiresAt);
        } else {
          const card = document.querySelector(`.file[data-id="${f.id}"]`);
          const { created, expires } = knownFiles.get(f.id);
          if (card && !card.classList.contains("downloading")) setProgress(card, created, expires);
        }
      });
    });
}

document.getElementById("files").onclick = (e) => {
  const card = e.target.closest(".file");
  if (card && !card.classList.contains("downloading")) {
    e.preventDefault();
    downloadFile(card, card.dataset.id, card.dataset.name);
  }
};

setupUpload();
pollFiles();
setInterval(pollFiles, 1000);
