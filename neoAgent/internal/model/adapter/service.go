package adapter

// ServiceFingerprintAttributes 对应 service_fingerprint 契约
type ServiceFingerprintAttributes struct {
	Services []ServiceInfo `json:"services"`
}

type ServiceInfo struct {
	IP         string `json:"ip"`
	Port       int    `json:"port"`
	Proto      string `json:"proto"`
	Name       string `json:"name"`
	Version    string `json:"version,omitempty"`
	Product    string `json:"product,omitempty"`
	OSType     string `json:"os_type,omitempty"`
	CPE        string `json:"cpe,omitempty"`
	Confidence int    `json:"confidence,omitempty"`
}
