package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rqlite/gorqlite"
)

type EnvironmentLog struct {
	Id            string
	Message       string
	MachineId     string
	DeploymentId  string
	EnvironmentId string
	ImageId       string
	Level         int64
	CreatedAt     string
}

type JournalctlDockerLog struct {
	Message       string `json:"MESSAGE"`
	Timestamp     string `json:"__REALTIME_TIMESTAMP"` //to think if we need _SOURCE_REALTIME_TIMESTAMP - https://www.freedesktop.org/software/systemd/man/latest/systemd.journal-fields.html
	ImageId       string `json:"IMAGE_NAME"`
	ContainerName string `json:"CONTAINER_NAME"`
}

func handleLogsEnvironmentGet(w http.ResponseWriter, r *http.Request) {

	environmentId := r.PathValue("environmentId")
	beforeOrAfter := r.PathValue("before_after")

	if beforeOrAfter != "before" && beforeOrAfter != "after" {
		http.Error(w, "Invalid format, use 'before' or 'after' instead of '"+beforeOrAfter+"'", http.StatusNotFound)
		return
	}

	timestamp := r.PathValue("timestamp")

	jsonBytes, err := json.Marshal(getLogsByEnvironmentId(environmentId, beforeOrAfter, timestamp))
	if err != nil {
		fmt.Println("Cannot convert Services object into JSON:", err)
		return
	}

	fmt.Fprint(w, string(jsonBytes))
}

func getLogsByEnvironmentId(environmentId string, beforeOrAfter string, timestamp string) []EnvironmentLog {

	beforeAfterSign := ""

	if beforeOrAfter == "before" {
		beforeAfterSign = "<="
	} else if beforeOrAfter == "after" {
		beforeAfterSign = ">="
	} else {
		return []EnvironmentLog{}
	}

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Message, EnvironmentId, ImageId, MachineId, DeploymentId, Level, CreatedAt from EnvLogs" + environmentId + " where EnvironmentId = ? AND CreatedAt " + beforeAfterSign + " datetime(?) ORDER BY CreatedAt DESC LIMIT 100",
			Arguments: []interface{}{environmentId, timestamp},
		},
	)

	return handleLogsQuery(rows, err)

}

func handleLogsQuery(rows gorqlite.QueryResult, err error) []EnvironmentLog {

	var environmentLogs = []EnvironmentLog{}

	if err != nil {
		fmt.Printf(" Cannot read from EnvironmentLog table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var EnvironmentId string
		var ImageId string
		var Message string
		var MachineId string
		var DeploymentId string
		var Level int64
		var CreatedAt string

		err := rows.Scan(&Id, &Message, &EnvironmentId, &ImageId, &MachineId, &DeploymentId, &Level, &CreatedAt)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedEnvironmentLog := EnvironmentLog{
			Id:            Id,
			EnvironmentId: EnvironmentId,
			ImageId:       ImageId,
			Message:       Message,
			MachineId:     MachineId,
			DeploymentId:  DeploymentId,
			Level:         Level,
			CreatedAt:     CreatedAt,
		}
		environmentLogs = append(environmentLogs, loadedEnvironmentLog)
	}

	return environmentLogs

}

func saveEnvironmentLog(environmentLog EnvironmentLog) {

	id, err := NanoId(10)

	//Get EnvironmentId from ImageId in case it's empty
	if environmentLog.EnvironmentId == "" && environmentLog.ImageId != "" {
		getImageById(environmentLog.ImageId)
	}

	if environmentLog.EnvironmentId == "" {
		//We cannot save a log with environmentId
		return
	}

	if err != nil {
		fmt.Printf(" Cannot generate id for a new EnvLogs%s record: %s\n", environmentLog.EnvironmentId, err.Error())
	}

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO EnvLogs" + environmentLog.EnvironmentId + "( Id, Message, MachineId, EnvironmentId, DeploymentId, Level, ImageId) VALUES(?, ?, ?, ?, ?, ?, ?)",
				Arguments: []interface{}{id, environmentLog.Message, environmentLog.MachineId, environmentLog.EnvironmentId, environmentLog.DeploymentId, environmentLog.Level, environmentLog.ImageId},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to EnvLog table: %s\n", err.Error())
	}

}

func handleDockerLogs() {
	_, _ = executeScriptString("journalctl -f -b -o json -u docker.service", func(logLine string) {
		//We get all docker logs with journalctl and check CONTAINER_NAME and IMAGE_NAME inside each log entry
		//If IMAGE_NAME presents, we save a log to LogEnvEnv_id
		//If no CONTAINER_NAME and IMAGE_NAME included with a log, that log is related to Docker itself and we should save that log to LogMachineMachineId
		// - like removing a container, fail to start a container etc. In this case we cannot know the environment and cannot save to LogEnvEnv_Id
		log := JournalctlDockerLog{}
		json.Unmarshal([]byte(logLine), &log)

		if log.ImageId != "" {
			var envLog EnvironmentLog
			envLog.MachineId = thisMachine.Id
			envLog.ImageId = log.ImageId
			envLog.Level = 0
			envLog.Message = log.Message
			saveEnvironmentLog(envLog)
		}

		fmt.Println(log.Message)
		fmt.Println(log.Timestamp)
		fmt.Println(log.ContainerName)
		fmt.Println(log.ImageId)

		//fmt.Println(logLine)
		////////////////////////
	})
}
