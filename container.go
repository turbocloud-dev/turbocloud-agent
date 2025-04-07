/*
Deployments
*/

package main

import (
	"bytes"
	"fmt"
	"time"

	"github.com/rqlite/gorqlite"
)

const ContainerJobTypeDelete = "container_del"

const ContainerJobStatusPlanned = "planned"
const ContainerJobStatusFinished = "finished"

type ContainerJob struct {
	Id            string
	Status        string
	EnvironmentId string
	MachineId     string
	JobType       string
}

func addContainerJob(job ContainerJob) ContainerJob {
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for DeploymentJob:", err)
		return job
	}

	job.Id = id

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO ContainerJob( Id, Status, EnvironmentId, MachineId, JobType) VALUES(?, ?, ?, ?, ?)",
				Arguments: []interface{}{job.Id, job.Status, job.EnvironmentId, job.MachineId, job.JobType},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to ContainerJob table: %s\n", err.Error())
	}

	fmt.Printf("New ContainerJob is scheduled for EnvironmentId %s\n", job.EnvironmentId)

	return job
}

func getContainerJobsByMachineIdAndStatusAndJobType(machineId string, status string, jobType string) []ContainerJob {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, EnvironmentId, MachineId, JobType from ContainerJob WHERE MachineId = ? AND Status = ? AND JobType = ?",
			Arguments: []interface{}{machineId, status, jobType},
		},
	)

	return handleContainerJobQuery(rows, err)
}

func handleContainerJobQuery(rows gorqlite.QueryResult, err error) []ContainerJob {

	var jobs = []ContainerJob{}

	if err != nil {
		fmt.Printf(" Cannot read from ContainerJob table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var Status string
		var EnvironmentId string
		var MachineId string
		var JobType string

		err := rows.Scan(&Id, &Status, &EnvironmentId, &MachineId, &JobType)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedJob := ContainerJob{
			Id:            Id,
			Status:        Status,
			EnvironmentId: EnvironmentId,
			MachineId:     MachineId,
			JobType:       JobType,
		}
		jobs = append(jobs, loadedJob)
	}

	return jobs

}

func stopPreviousContainer(environmentId string) {
	//Stop old container
	//Get all deployments and take the previous to get deploymentId that we use a container name
	deployments := getDeploymentsByEnvironmentId(environmentId)
	if len(deployments) > 1 {
		//Stop the previous container
		stopAndRemoveContainer(deployments[1].Id)
	}
}

func stopAndRemoveContainer(deploymentId string) {
	fmt.Printf("Removing container with ID %s\n", deploymentId)

	scriptTemplate := createTemplate("caddyfile", `
	#!/bin/sh
	docker stop {{.DEPLOYMENT_ID}}
	docker container rm -f {{.DEPLOYMENT_ID}}
`)

	var templateBytes bytes.Buffer
	templateData := map[string]string{
		"DEPLOYMENT_ID": deploymentId,
	}

	if err := scriptTemplate.Execute(&templateBytes, templateData); err != nil {
		fmt.Println("Cannot execute template for Caddyfile:", err)
	}

	scriptString := templateBytes.String()

	_, err := executeScriptString(scriptString, func(logLine string) {
		//Save log message
		var envLog EnvironmentLog
		envLog.DeploymentId = deploymentId
		envLog.MachineId = thisMachine.Id
		envLog.Level = "6"
		envLog.Message = logLine
		saveEnvironmentLog(envLog)
		////////////////////////
	})
	if err != nil {
		fmt.Println("Cannot remove a container")
		return
	}
}

func startContainerJobsCheckerWorker() {
	for range time.Tick(time.Second * 5) {
		go func() {
			containerJobs := getContainerJobsByMachineIdAndStatusAndJobType(thisMachine.Id, ContainerJobStatusPlanned, ContainerJobTypeDelete)
			if len(containerJobs) > 0 {
				fmt.Printf("Found %d ContainerJobs with status %s\n", len(containerJobs), ContainerJobStatusPlanned)
				for _, containerJob := range containerJobs {
					//Find all deployments
					deployments := getLastDeploymentByEnvironmentId(containerJob.EnvironmentId)
					if len(deployments) > 0 {
						//Stop the last container only because all previous should be stopped and removed already
						stopAndRemoveContainer(deployments[0].Id)
						//Update ContainerJob status
						updateContainerJobStatus(containerJob, ContainerJobStatusFinished)
					}
				}
			}

		}()
	}
}

func updateContainerJobStatus(job ContainerJob, status string) error {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE ContainerJob SET Status = ? WHERE Id = ?",
				Arguments: []interface{}{status, job.Id},
			},
		},
	)

	if err != nil {
		fmt.Printf("Cannot update a row in ContainerJob: %s\n", err.Error())
		return err
	}

	return nil
}
