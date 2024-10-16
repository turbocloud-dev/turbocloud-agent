package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
)

const StatusToDeploy = "ToDeploy"
const StatusInProgress = "InProgress"
const StatusDeployed = "Deployed"

type Deployment struct {
	Id            string
	Status        string
	MachineIds    []string
	EnvironmentId string
	ImageId       string
}

func handleDeploymentPost(w http.ResponseWriter, r *http.Request) {
	var deployment Deployment

	environmentId := r.PathValue("environmentId")
	deployment.EnvironmentId = environmentId

	err := decodeJSONBody(w, r, &deployment)

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

	addDeployment(&deployment)

	jsonBytes, err := json.Marshal(deployment)
	if err != nil {
		fmt.Println("Cannot convert Proxy object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))

}
