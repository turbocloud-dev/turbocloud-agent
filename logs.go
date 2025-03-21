package main

import (
	"fmt"

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
