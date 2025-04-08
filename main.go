package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"slices"
	"strconv"
)

var PORT string
var containerRegistryIp = "192.168.202.1"

var allowedOrigins = []string{"http://localhost:5045", "https://console.turbocloud.dev"}

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

func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		origin := r.Header.Get("Origin")
		if slices.Contains(allowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "*")
		}
		next.ServeHTTP(w, r)

	})
}

func main() {

	timestamp, _ := strconv.ParseInt("1744049650869546", 10, 64)
	fmt.Println(timestamp)

	loadInfoFromVPNCert()
	databaseInit()
	loadMachineInfo()

	registryEnv, isRegistryEnvExists := os.LookupEnv("TURBOCLOUD_CONTAINER_REGISTRY")
	if isRegistryEnvExists {
		containerRegistryIp = registryEnv
	}

	mux := http.NewServeMux()

	//Preflight requests requests
	mux.HandleFunc("OPTIONS /{pathname...}", handleOptions)

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
	mux.HandleFunc("GET /environment/{environmentId}/deployments", handleEnvironmentDeploymentsGet)

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

	//Logs
	mux.HandleFunc("GET /logs/environment/{environmentId}/{before_after}/{timestamp}", handleLogsEnvironmentGet)

	wrapped := use(mux, loggingMiddleware, CORSMiddleware)

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

	go handleDockerLogs()

	fmt.Println("Starting an agent on port " + PORT)
	log.Fatal(http.ListenAndServe(":"+PORT, wrapped))
}
