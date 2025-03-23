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
