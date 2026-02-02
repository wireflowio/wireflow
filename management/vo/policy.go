package vo

type PolicyVo struct {
	Name         string `json:"name"`
	Type         string `json:"type"`
	Description  string `json:"description"`
	PeerSelector string `json:"peerSelector"`
}
