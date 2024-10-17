/*
We create DeploymentJob for each deployment on each server. If Environment has 5 MachineIds in Environment where we should deploy we should create 5 DeploymentJobs
*/

package main

import (
	"fmt"

	"github.com/rqlite/gorqlite"
)

const StatusToDeploy = "to_deploy"
const StatusInProgress = "in_progress"
const StatusDeployed = "deployed"

type DeploymentJob struct {
	Id           string
	Status       string
	DeploymentId string
	MachineId    string
}

func addDeploymentJob(job DeploymentJob) DeploymentJob {
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for DeploymentJob:", err)
		return job
	}

	job.Id = id

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO DeploymentJob( Id, Status, DeploymentId, MachineId) VALUES(?, ?, ?, ?)",
				Arguments: []interface{}{job.Id, job.Status, job.DeploymentId, job.MachineId},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to DeploymentJob table: %s\n", err.Error())
	}
	return job
}

func getDeploymentJobsByStatus(status string) []DeploymentJob {
	var jobs []DeploymentJob

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, DeploymentId, MachineId from DeploymentJob WHERE STATUS = ?",
			Arguments: []interface{}{status},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot read from DeploymentJob table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var Status string
		var DeploymentId string
		var MachineId string

		err := rows.Scan(&Id, &Status, &DeploymentId, &MachineId)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedJob := DeploymentJob{
			Id:           Id,
			Status:       Status,
			DeploymentId: DeploymentId,
			MachineId:    MachineId,
		}
		jobs = append(jobs, loadedJob)
	}

	return jobs
}
