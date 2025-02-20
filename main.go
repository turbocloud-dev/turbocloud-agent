package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
)

var PORT string
var containerRegistryIp = "192.168.202.1"

func use(r *http.ServeMux, middlewares ...func(next http.Handler) http.Handler) http.Handler {
	var s http.Handler
	s = r

	for _, mw := range middlewares {
		s = mw(s)
	}

	return s
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("New request: %s %s", r.Method, r.URL.String())
		next.ServeHTTP(w, r)
	})
}

func acceptHeaderMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Accept: %v", r.Header.Get("Accept"))
		next.ServeHTTP(w, r)
	})
}

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		origin := r.Header.Get("Origin")
		allowedOrigins := []string{"http://localhost:5045", "https://console.turbocloud.dev"}

		if slices.Contains(allowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		next.ServeHTTP(w, r)

	})
}

func main() {

	loadInfoFromVPNCert()
	databaseInit()
	loadMachineInfo()

	registryEnv, isRegistryEnvExists := os.LookupEnv("TURBOCLOUD_CONTAINER_REGISTRY")
	if isRegistryEnvExists {
		containerRegistryIp = registryEnv
	}

	mux := http.NewServeMux()

	//Proxy routes
	mux.HandleFunc("GET /hey", handleHeyGot)
	mux.HandleFunc("POST /proxy", handleProxyPost)
	mux.HandleFunc("GET /proxy", handleProxyGet)
	mux.HandleFunc("DELETE /proxy/{id}", handleProxyDelete)

	//Service routes
	mux.HandleFunc("POST /service", handleServicePost)
	mux.HandleFunc("GET /service", handleServiceGet)
	mux.HandleFunc("DELETE /service/{id}", handleServiceDelete)

	//Environment routes
	mux.HandleFunc("POST /environment", handleEnvironmentPost)
	mux.HandleFunc("GET /service/{serviceId}/environment", handleEnvironmentByServiceIdGet)
	mux.HandleFunc("PUT /environment", handleEnvironmentByServiceIdPut)
	mux.HandleFunc("DELETE /environment/{id}", handleEnvironmentDelete)

	//Deployment routes
	mux.HandleFunc("GET /deploy/environment/{environmentId}", handleEnvironmentDeploymentGet)
	mux.HandleFunc("POST /deploy/environment/{environmentId}", handleEnvironmentDeploymentPost)
	mux.HandleFunc("POST /deploy/{serviceId}", handleServiceDeploymentPost)

	//Machine routes
	mux.HandleFunc("POST /machine", handleMachinePost)
	mux.HandleFunc("GET /machine", handleMachineGet)
	mux.HandleFunc("GET /join/{machineId}/{secret}", handleJoinGet)
	mux.HandleFunc("GET /public-ssh-keys", handlePublicSSHKeysGet)
	mux.HandleFunc("GET /machine/stats", handleMachineStatsGet)
	mux.HandleFunc("DELETE /machine/{id}", handleMachineDelete)

	wrapped := use(mux, loggingMiddleware, CORSMiddleware, acceptHeaderMiddleware)

	port_env, is_port_env_exists := os.LookupEnv("TURBOCLOUD_AGENT_PORT")
	if is_port_env_exists {
		PORT = port_env
	} else {
		PORT = "5445"
	}

	go startDeploymentCheckerWorker()

	reloadProxyServer()
	go startProxyCheckerWorker()

	go loadMachineStats()
	go pingMachines()

	go startContainerJobsCheckerWorker()
	go startImageJobsCheckerWorker()

	fmt.Println("Starting an agent on port " + PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, wrapped))
}
