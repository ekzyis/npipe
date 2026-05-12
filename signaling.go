package main

import (
	"encoding/json"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Peer struct {
	ID   string
	IP   string
	Conn *websocket.Conn
}

type FileInfo struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Size    int    `json:"size"`
	OwnerID string `json:"ownerId"`
}

type SignalMessage struct {
	Type     string          `json:"type"`
	From     string          `json:"from,omitempty"`
	To       string          `json:"to,omitempty"`
	FileID   string          `json:"fileId,omitempty"`
	FileName string          `json:"fileName,omitempty"`
	FileSize int             `json:"fileSize,omitempty"`
	Payload  json.RawMessage `json:"payload,omitempty"`
}

type SignalingHub struct {
	mu    sync.RWMutex
	peers map[string]*Peer          // peerID -> Peer
	files map[string]*FileInfo      // fileID -> FileInfo
	ipMap map[string]map[string]bool // ip -> set of peerIDs
}

func NewSignalingHub() *SignalingHub {
	return &SignalingHub{
		peers: make(map[string]*Peer),
		files: make(map[string]*FileInfo),
		ipMap: make(map[string]map[string]bool),
	}
}

var hub = NewSignalingHub()

func (h *SignalingHub) AddPeer(peer *Peer) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.peers[peer.ID] = peer
	if h.ipMap[peer.IP] == nil {
		h.ipMap[peer.IP] = make(map[string]bool)
	}
	h.ipMap[peer.IP][peer.ID] = true
}

func (h *SignalingHub) RemovePeer(peerID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if peer, ok := h.peers[peerID]; ok {
		delete(h.ipMap[peer.IP], peerID)
		if len(h.ipMap[peer.IP]) == 0 {
			delete(h.ipMap, peer.IP)
		}
	}
	delete(h.peers, peerID)
	// Remove files owned by this peer
	for id, f := range h.files {
		if f.OwnerID == peerID {
			delete(h.files, id)
		}
	}
}

func (h *SignalingHub) AddFile(f *FileInfo) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.files[f.ID] = f
}

func (h *SignalingHub) RemoveFile(fileID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.files, fileID)
}

func (h *SignalingHub) GetFilesForIP(ip string) []*FileInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	var files []*FileInfo
	peerIDs := h.ipMap[ip]
	for _, f := range h.files {
		if peerIDs[f.OwnerID] {
			files = append(files, f)
		}
	}
	return files
}

func (h *SignalingHub) GetPeer(peerID string) *Peer {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.peers[peerID]
}

func (h *SignalingHub) GetFile(fileID string) *FileInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.files[fileID]
}

func (h *SignalingHub) BroadcastToIP(ip string, msg SignalMessage, excludePeerID string) {
	h.mu.RLock()
	peerIDs := h.ipMap[ip]
	var peers []*Peer
	for pid := range peerIDs {
		if pid != excludePeerID {
			if p := h.peers[pid]; p != nil {
				peers = append(peers, p)
			}
		}
	}
	h.mu.RUnlock()

	data, _ := json.Marshal(msg)
	for _, p := range peers {
		p.Conn.WriteMessage(websocket.TextMessage, data)
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("websocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	ip := getClientIP(r)
	peerID := r.URL.Query().Get("id")
	if peerID == "" {
		return
	}

	peer := &Peer{ID: peerID, IP: ip, Conn: conn}
	hub.AddPeer(peer)
	log.Printf("%s connected (peer %s)", ip, peerID)

	defer func() {
		hub.RemovePeer(peerID)
		log.Printf("%s disconnected (peer %s)", ip, peerID)
		// Notify others that files are gone
		hub.BroadcastToIP(ip, SignalMessage{Type: "peer-left", From: peerID}, peerID)
	}()

	// Send current files to new peer
	files := hub.GetFilesForIP(ip)
	for _, f := range files {
		msg := SignalMessage{Type: "file-available", FileID: f.ID, FileName: f.Name, FileSize: f.Size, From: f.OwnerID}
		data, _ := json.Marshal(msg)
		conn.WriteMessage(websocket.TextMessage, data)
	}

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var msg SignalMessage
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		msg.From = peerID

		switch msg.Type {
		case "share":
			// Peer is sharing a file
			f := &FileInfo{
				ID:      msg.FileID,
				Name:    msg.FileName,
				Size:    msg.FileSize,
				OwnerID: peerID,
			}
			hub.AddFile(f)
			log.Printf("%s sharing %s (%d bytes)", ip, f.ID, f.Size)
			// Broadcast to other peers on same IP
			hub.BroadcastToIP(ip, SignalMessage{
				Type:     "file-available",
				FileID:   f.ID,
				FileName: f.Name,
				FileSize: f.Size,
				From:     peerID,
			}, peerID)

		case "unshare":
			hub.RemoveFile(msg.FileID)
			hub.BroadcastToIP(ip, SignalMessage{Type: "file-unavailable", FileID: msg.FileID}, peerID)

		case "request":
			// Peer wants to download a file, relay to owner
			f := hub.GetFile(msg.FileID)
			if f == nil {
				continue
			}
			owner := hub.GetPeer(f.OwnerID)
			if owner == nil {
				continue
			}
			msg.To = f.OwnerID
			data, _ := json.Marshal(msg)
			owner.Conn.WriteMessage(websocket.TextMessage, data)

		case "offer", "answer", "ice-candidate":
			// Relay WebRTC signaling to target peer
			target := hub.GetPeer(msg.To)
			if target == nil {
				continue
			}
			data, _ := json.Marshal(msg)
			target.Conn.WriteMessage(websocket.TextMessage, data)
		}
	}
}
