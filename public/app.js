const icons = {
  file: `
    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5">
      <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/>
      <polyline points="14 2 14 8 20 8"/>
    </svg>`
};

function addFileCard(id, name, size, isOwn = false) {
  if (document.querySelector(`.file[data-id="${id}"]`)) return;

  const div = document.createElement("div");
  div.className = "card file" + (isOwn ? " own" : "");
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

function removeFileCard(id) {
  const card = document.querySelector(`.file[data-id="${id}"]`);
  if (card) card.remove();
}

function downloadBlob(name, blob) {
  const url = URL.createObjectURL(blob);
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}

// WebRTC callbacks
rtc.onFileAvailable((id, name, size) => {
  addFileCard(id, name, size);
});

rtc.onFileUnavailable((id) => {
  removeFileCard(id);
});

rtc.onDownloadProgress((id, progress) => {
  const card = document.querySelector(`.file[data-id="${id}"]`);
  if (card) {
    card.style.setProperty("--progress", progress);
    console.log("downloading:", progress.toFixed(1) + "%");
  }
});

rtc.onDownloadComplete((id, name, blob) => {
  console.log("download complete:", name, "size:", blob.size);
  const card = document.querySelector(`.file[data-id="${id}"]`);
  if (card) card.classList.remove("downloading");
  downloadBlob(name, blob);
});

// UI events
document.getElementById("f").onchange = (e) => {
  const file = e.target.files[0];
  if (file) {
    rtc.share(file, (id, name, size) => {
      addFileCard(id, name, size, true);
    });
    e.target.value = "";
  }
};

document.getElementById("files").onclick = (e) => {
  const card = e.target.closest(".file");
  if (card && !card.classList.contains("downloading") && !card.classList.contains("own")) {
    e.preventDefault();
    card.classList.add("downloading");
    card.style.setProperty("--progress", 0);
    rtc.request(card.dataset.id);
  }
};

rtc.connect();
