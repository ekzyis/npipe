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
  file: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
      <polyline points="14 2 14 8 20 8"/>
    </svg>`
};

const knownFiles = new Set();

function setupUpload() {
  document.getElementById("f").onchange = (e) => {
    const file = e.target.files[0];
    const reader = new FileReader();

    reader.onload = () => {
      const data = Array.from(new Uint8Array(reader.result));
      const label = document.querySelector("#upload label");
      const origHTML = label.innerHTML;

      label.innerHTML = `
        ${icons.spinner}
        <span>uploading</span>
      `;

      fetch("/file", {
        method: "POST",
        body: JSON.stringify({ name: file.name, data })
      }).then((res) => {
        if (res.ok) {
          pollFiles();
          document.getElementById("f").value = "";
          label.innerHTML = `
            <span class="success">${icons.check}</span>
            <span class="success">uploaded</span>
          `;
          setTimeout(() => {
            label.innerHTML = origHTML;
          }, 1000);
        }
      });
    };

    reader.readAsArrayBuffer(file);
  };
}

function addFile(id, name) {
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
}

function downloadFile(id, name) {
  fetch("/file/" + id)
    .then((res) => res.blob())
    .then((blob) => {
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = name;
      a.click();
    });
}

function pollFiles() {
  fetch("/files")
    .then((res) => res.json())
    .then((files) => {
      files.forEach((f) => {
        if (!knownFiles.has(f.id)) {
          knownFiles.add(f.id);
          addFile(f.id, f.name);
        }
      });
    });
}

document.getElementById("files").onclick = (e) => {
  const card = e.target.closest(".file");
  if (card) {
    e.preventDefault();
    downloadFile(card.dataset.id, card.dataset.name);
  }
};

setupUpload();
pollFiles();
setInterval(pollFiles, 2000);
