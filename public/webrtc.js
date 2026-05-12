const peerId = crypto.randomUUID();
const sharedFiles = new Map(); // fileId -> { name, data }
const remoteFiles = new Map(); // fileId -> { name, size, ownerId }
const peerConnections = new Map(); // peerId -> RTCPeerConnection

const rtcConfig = {
  iceServers: [
    { urls: "stun:stun.l.google.com:19302" },
    { urls: "stun:stun1.l.google.com:19302" },
  ]
};

let ws;
let onFileAvailable = () => {};
let onFileUnavailable = () => {};
let onDownloadProgress = () => {};
let onDownloadComplete = () => {};

function connectWebSocket() {
  const protocol = location.protocol === "https:" ? "wss:" : "ws:";
  ws = new WebSocket(`${protocol}//${location.host}/ws?id=${peerId}`);

  ws.onopen = () => console.log("signaling connected");
  ws.onclose = () => setTimeout(connectWebSocket, 1000);
  ws.onmessage = (e) => handleSignal(JSON.parse(e.data));
}

function send(msg) {
  console.log("signal sending:", msg.type, msg);
  if (ws.readyState === WebSocket.OPEN) {
    ws.send(JSON.stringify(msg));
  }
}

function handleSignal(msg) {
  console.log("signal received:", msg.type, msg);
  switch (msg.type) {
    case "file-available":
      remoteFiles.set(msg.fileId, { name: msg.fileName, size: msg.fileSize, ownerId: msg.from });
      onFileAvailable(msg.fileId, msg.fileName, msg.fileSize);
      break;

    case "file-unavailable":
      remoteFiles.delete(msg.fileId);
      onFileUnavailable(msg.fileId);
      break;

    case "peer-left":
      for (const [id, f] of remoteFiles) {
        if (f.ownerId === msg.from) {
          remoteFiles.delete(id);
          onFileUnavailable(id);
        }
      }
      break;

    case "request":
      handleDownloadRequest(msg.from, msg.fileId);
      break;

    case "offer":
      handleOffer(msg.from, msg.fileId, msg.payload);
      break;

    case "answer":
      handleAnswer(msg.from, msg.payload);
      break;

    case "ice-candidate":
      handleIceCandidate(msg.from, msg.payload);
      break;
  }
}

function generateId() {
  return Array.from(crypto.getRandomValues(new Uint8Array(8)))
    .map(b => b.toString(16).padStart(2, "0")).join("");
}

function shareFile(file, callback) {
  const reader = new FileReader();
  reader.onload = () => {
    const id = generateId();
    const data = new Uint8Array(reader.result);
    sharedFiles.set(id, { name: file.name, data });
    send({ type: "share", fileId: id, fileName: file.name, fileSize: data.length });
    console.log("sharing:", id, file.name);
    if (callback) callback(id, file.name, data.length);
  };
  reader.readAsArrayBuffer(file);
}

function requestFile(fileId) {
  const file = remoteFiles.get(fileId);
  console.log("requesting file:", fileId, file);
  if (!file) return;
  send({ type: "request", fileId, to: file.ownerId });
}

async function handleOffer(from, fileId, offer) {
  console.log("handling offer from:", from, "for file:", fileId);
  const pc = new RTCPeerConnection(rtcConfig);
  peerConnections.set(from, pc);

  pc.onconnectionstatechange = () => console.log("connection state:", pc.connectionState);
  pc.oniceconnectionstatechange = () => console.log("ice state:", pc.iceConnectionState);

  const chunks = [];
  let received = 0;
  const file = remoteFiles.get(fileId);

  pc.ondatachannel = (e) => {
    const dc = e.channel;
    console.log("datachannel received, state:", dc.readyState);
    dc.binaryType = "arraybuffer";
    dc.onopen = () => console.log("datachannel open");
    dc.onclose = () => console.log("datachannel closed");
    dc.onerror = (err) => console.log("datachannel error:", err);
    dc.onmessage = (e) => {
      chunks.push(e.data);
      received += e.data.byteLength;
      if (file) {
        const progress = (received / file.size) * 100;
        console.log("download progress:", progress.toFixed(1) + "%");
        onDownloadProgress(fileId, progress);
      }
    };
    dc.onclose = () => {
      console.log("transfer complete, blob size:", chunks.reduce((a, c) => a + c.byteLength, 0));
      const blob = new Blob(chunks);
      onDownloadComplete(fileId, file?.name || "download", blob);
      pc.close();
      peerConnections.delete(from);
      remoteFiles.delete(fileId);
      onFileUnavailable(fileId);
    };
  };

  pc.onicecandidate = (e) => {
    if (e.candidate) {
      send({ type: "ice-candidate", to: from, payload: e.candidate });
    }
  };

  await pc.setRemoteDescription(offer);
  const answer = await pc.createAnswer();
  await pc.setLocalDescription(answer);
  send({ type: "answer", to: from, payload: answer });
}

async function handleDownloadRequest(from, fileId) {
  console.log("handling download request from:", from, "for file:", fileId);
  const file = sharedFiles.get(fileId);
  console.log("file found:", file ? file.name : "NOT FOUND");
  if (!file) return;

  const pc = new RTCPeerConnection(rtcConfig);
  peerConnections.set(from, pc);

  pc.onconnectionstatechange = () => console.log("connection state:", pc.connectionState);
  pc.oniceconnectionstatechange = () => console.log("ice state:", pc.iceConnectionState);

  const dc = pc.createDataChannel("file");
  dc.binaryType = "arraybuffer";

  dc.onerror = (err) => console.log("datachannel error:", err);
  dc.onclose = () => console.log("datachannel closed (sender)");
  dc.onopen = () => {
    console.log("datachannel open (sender), starting transfer");
    const chunkSize = 16384;
    let offset = 0;
    const sendChunk = () => {
      if (offset >= file.data.length) {
        dc.close();
        pc.close();
        peerConnections.delete(from);
        sharedFiles.delete(fileId);
        send({ type: "unshare", fileId });
        onFileUnavailable(fileId);
        return;
      }
      const chunk = file.data.slice(offset, offset + chunkSize);
      dc.send(chunk);
      offset += chunkSize;
      setTimeout(sendChunk, 0);
    };
    sendChunk();
  };

  pc.onicecandidate = (e) => {
    if (e.candidate) {
      send({ type: "ice-candidate", to: from, payload: e.candidate });
    }
  };

  const offer = await pc.createOffer();
  await pc.setLocalDescription(offer);
  send({ type: "offer", to: from, fileId, payload: offer });
}

function handleAnswer(from, answer) {
  const pc = peerConnections.get(from);
  if (pc) pc.setRemoteDescription(answer);
}

function handleIceCandidate(from, candidate) {
  const pc = peerConnections.get(from);
  if (pc) pc.addIceCandidate(candidate);
}

// Public API
window.rtc = {
  connect: connectWebSocket,
  share: shareFile,
  request: requestFile,
  onFileAvailable: (fn) => { onFileAvailable = fn; },
  onFileUnavailable: (fn) => { onFileUnavailable = fn; },
  onDownloadProgress: (fn) => { onDownloadProgress = fn; },
  onDownloadComplete: (fn) => { onDownloadComplete = fn; },
};
