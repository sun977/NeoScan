package adapter

// WebEndpointAttributes 对应 web_endpoint 契约
type WebEndpointAttributes struct {
	Endpoints []WebEndpointInfo `json:"endpoints"`
}

type WebEndpointInfo struct {
	URL           string            `json:"url"`
	IP            string            `json:"ip"`
	Port          int               `json:"port"`
	Title         string            `json:"title,omitempty"`
	StatusCode    int               `json:"status_code"`
	ContentLength int64             `json:"content_length,omitempty"`
	Headers       map[string]string `json:"headers,omitempty"`
	TechStack     []string          `json:"tech_stack,omitempty"`
	Screenshot    string            `json:"screenshot,omitempty"` // Base64
	Favicon       string            `json:"favicon,omitempty"`    // Base64
}

// DirectoryScanAttributes 对应 directory_scan 契约
type DirectoryScanAttributes struct {
	Paths []PathInfo `json:"paths"`
}

type PathInfo struct {
	URL       string `json:"url"`
	Status    int    `json:"status"`
	Length    int64  `json:"length,omitempty"`
	Sensitive bool   `json:"sensitive,omitempty"`
}

// ApiDiscoveryAttributes 对应 api_discovery 契约
type ApiDiscoveryAttributes struct {
	APIs []ApiInfo `json:"apis"`
	Spec *ApiSpec  `json:"spec,omitempty"`
}

type ApiInfo struct {
	Method       string `json:"method"`
	Path         string `json:"path"`
	Status       int    `json:"status"`
	AuthRequired bool   `json:"auth_required,omitempty"`
}

type ApiSpec struct {
	Format  string `json:"format,omitempty"` // OpenAPI, Swagger
	Version string `json:"version,omitempty"`
}
