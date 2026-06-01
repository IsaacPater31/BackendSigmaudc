package utils

import (
	"net/http"
	"strings"
)

// GetIPAddress obtiene la dirección IP real del cliente
func GetIPAddress(r *http.Request) string {
	// Intentar obtener IP de headers de proxy
	ip := r.Header.Get("X-Forwarded-For")
	if ip != "" {
		// X-Forwarded-For puede contener múltiples IPs, tomar la primera
		ips := strings.Split(ip, ",")
		return strings.TrimSpace(ips[0])
	}

	ip = r.Header.Get("X-Real-Ip")
	if ip != "" {
		return ip
	}

	// Si no hay headers de proxy, usar RemoteAddr
	ip = r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}

	return ip
}

