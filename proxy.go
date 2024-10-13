package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
)

type Proxy struct {
	Id              string
	ContainerId     string
	ServerPrivateIP string
	Port            string
	Domain          string
}

func handleProxyPost(w http.ResponseWriter, r *http.Request) {
	var proxy Proxy
	err := decodeJSONBody(w, r, &proxy)

	if err != nil {
		var mr *malformedRequest
		if errors.As(err, &mr) {
			http.Error(w, mr.msg, mr.status)
		} else {
			log.Print(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}

	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for Proxy:", err)
		return
	}

	proxy.Id = id
	addProxy(&proxy)

	jsonBytes, err := json.Marshal(proxy)
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))

}

func handleProxyDelete(w http.ResponseWriter, r *http.Request) {
	proxyId := r.PathValue("id")

	if !deleteProxy(proxyId) {
		fmt.Println("Cannot delete Proxy from Proxy table")
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	fmt.Fprint(w, "")

}

func handleProxyGet(w http.ResponseWriter, r *http.Request) {

	jsonBytes, err := json.Marshal(getAllProxies())
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))
}

func reloadProxyServer() {

	const caddyfilePath = `/etc/caddy/Caddyfile`

	f, err := os.Create(caddyfilePath)
	if err != nil {
		fmt.Println("Cannot open/create Caddyfile:", err)
	}

	caddyfileTemplate := createTemplate("caddyfile", `
{
	order coraza_waf first
}

{{.TURBOCLOUD_AGENT_DOMAIN}} {

    coraza_waf {
        load_owasp_crs
        directives `+"`"+`
            Include @coraza.conf-recommended
            Include @crs-setup.conf.example
            Include @owasp_crs/*.conf
            SecRuleEngine On
        `+"`"+`
        }


    reverse_proxy /hey localhost:{{.TURBOCLOUD_AGENT_PORT}}
    reverse_proxy /deploy/* localhost:{{.TURBOCLOUD_AGENT_PORT}}
    reverse_proxy /join_vpn/* localhost:{{.TURBOCLOUD_AGENT_PORT}}
    reverse_proxy * abort
}

192.168.202.1.localcloud.dev {


    coraza_waf {
        load_owasp_crs
        directives `+"`"+`
            Include @coraza.conf-recommended
            Include @crs-setup.conf.example
            Include @owasp_crs/*.conf
            SecRuleEngine On
		`+"`"+`
        }

    reverse_proxy * localhost:5445
    tls /etc/ssl/vpn_fullchain.pem /etc/ssl/vpn_private.key
}
`)

	var templateBytes bytes.Buffer
	templateData := map[string]string{
		"TURBOCLOUD_AGENT_DOMAIN": os.Getenv("TURBOCLOUD_AGENT_DOMAIN"),
		"TURBOCLOUD_AGENT_PORT":   PORT,
	}

	if err := caddyfileTemplate.Execute(&templateBytes, templateData); err != nil {
		fmt.Println("Cannot execute template for Caddyfile:", err)
	}

	var templateDomainBytes bytes.Buffer

	proxies := getAllProxies()
	caddyfileDomainTemplate := createTemplate("caddyfile", `
	{{ range . }}

{{.Domain}} {

    coraza_waf {
        load_owasp_crs
        directives `+"`"+`
            Include @coraza.conf-recommended
            Include @crs-setup.conf.example
            Include @owasp_crs/*.conf
            SecRuleEngine On
		`+"`"+`
        }

    reverse_proxy * {{.ServerPrivateIP}}:{{.Port}}
}

{{ end }}

`)

	if err := caddyfileDomainTemplate.Execute(&templateDomainBytes, proxies); err != nil {
		fmt.Println("Cannot execute template for Caddyfile:", err)
	}
	// A `WriteString` is also available.
	_, err = f.WriteString(templateBytes.String() + templateDomainBytes.String())

	if err != nil {
		fmt.Println("Cannot save Caddyfile:", err)
	}

	f.Sync()

	_, err = exec.Command("caddy", "reload", "-c", caddyfilePath).Output()
	if err != nil {
		switch e := err.(type) {
		case *exec.Error:
			fmt.Println("Failed executing 'caddy reload' :", err)
		case *exec.ExitError:
			fmt.Println("'caddy reload' exited with error code =", e.ExitCode())
		}
	} else {
		fmt.Println("Caddy has been reloaded")
	}

}
