package nmap

import _ "embed"

//go:embed nmap-service-probes
var NmapServiceProbes string

//go:embed nmap-custom-probes
var NmapCustomizeProbes string

//go:embed nmap-os-db
var NmapOSDB string
