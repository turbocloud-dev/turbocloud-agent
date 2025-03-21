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

const ImageJobTypeDelete = "container_del"

const ImageJobStatusPlanned = "planned"
const ImageJobStatusFinished = "finished"

type ImageJob struct {
	Id            string
	Status        string
	EnvironmentId string
	JobType       string
}

func addImageJob(job ImageJob) ImageJob {
	id, err := NanoId(7)
	if err != nil {
		fmt.Println("Cannot generate new NanoId for ImageJob:", err)
		return job
	}

	job.Id = id

	_, err = connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "INSERT INTO ImageJob( Id, Status, EnvironmentId, JobType) VALUES(?, ?, ?, ?)",
				Arguments: []interface{}{job.Id, job.Status, job.EnvironmentId, job.JobType},
			},
		},
	)

	if err != nil {
		fmt.Printf(" Cannot write to ImageJob table: %s\n", err.Error())
	}

	fmt.Printf("New ImageJob is scheduled for EnvironmentId %s\n", job.EnvironmentId)

	return job
}

func getImageJobsByStatus(status string) []ImageJob {

	rows, err := connection.QueryOneParameterized(
		gorqlite.ParameterizedStatement{
			Query:     "SELECT Id, Status, EnvironmentId, JobType from ImageJob WHERE Status = ?",
			Arguments: []interface{}{status},
		},
	)

	return handleImageJobQuery(rows, err)
}

func handleImageJobQuery(rows gorqlite.QueryResult, err error) []ImageJob {

	var jobs = []ImageJob{}

	if err != nil {
		fmt.Printf(" Cannot read from ContainerJob table: %s\n", err.Error())
	}

	for rows.Next() {
		var Id string
		var Status string
		var EnvironmentId string
		var JobType string

		err := rows.Scan(&Id, &Status, &EnvironmentId, &JobType)
		if err != nil {
			fmt.Printf(" Cannot run Scan: %s\n", err.Error())
		}
		loadedJob := ImageJob{
			Id:            Id,
			Status:        Status,
			EnvironmentId: EnvironmentId,
			JobType:       JobType,
		}
		jobs = append(jobs, loadedJob)
	}

	return jobs

}

func removeImage(imageId string, environmentId string) {
	fmt.Printf("Removing image with ID %s\n", imageId)

	//Save log message
	var envLog EnvironmentLog
	envLog.EnvironmentId = environmentId
	envLog.MachineId = thisMachine.Id
	envLog.ImageId = imageId
	envLog.Level = 0
	envLog.Message = "Removing image with ID " + imageId
	saveEnvironmentLog(envLog)
	////////////////////////

	scriptTemplate := createTemplate("caddyfile", `
	#!/bin/sh
	docker image rm -f {{.IMAGE_ID}}
	docker image rm -f {{.CONTAINER_REGISTRY_IP}}:7000/{{.IMAGE_ID}}
`)

	var templateBytes bytes.Buffer
	templateData := map[string]string{
		"IMAGE_ID":              imageId,
		"CONTAINER_REGISTRY_IP": containerRegistryIp,
	}

	if err := scriptTemplate.Execute(&templateBytes, templateData); err != nil {
		fmt.Println("Cannot execute template for Caddyfile:", err)
	}

	scriptString := templateBytes.String()

	_, err := executeScriptString(scriptString, func(logLine string) {
		//Save log message
		var envLog EnvironmentLog
		envLog.EnvironmentId = environmentId
		envLog.MachineId = thisMachine.Id
		envLog.ImageId = imageId
		envLog.Level = 0
		envLog.Message = logLine
		saveEnvironmentLog(envLog)
		////////////////////////
	})
	if err != nil {
		fmt.Println("Cannot remove an image with ID %s", imageId)
		return
	}
}

func startImageJobsCheckerWorker() {
	for range time.Tick(time.Second * 5) {
		go func() {
			imageJobs := getImageJobsByStatus(ImageJobStatusPlanned)
			if len(imageJobs) > 0 {
				fmt.Printf("Found %d ImageJobs with status %s\n", len(imageJobs), ImageJobStatusPlanned)
				for _, imageJob := range imageJobs {
					//Find all deployments
					images := getImagesByEnvironmentId(imageJob.EnvironmentId)
					if len(images) > 0 {
						//Remove 2 last images because we keep only 2 last images
						removeImage(images[0].Id, imageJob.EnvironmentId)
						if len(images) > 1 {
							removeImage(images[1].Id, imageJob.EnvironmentId)
						}
						//Update ImageJob status
						updateImageJobStatus(imageJob, ImageJobStatusFinished)
					}
				}
			}

		}()
	}
}

func updateImageJobStatus(job ImageJob, status string) error {

	_, err := connection.WriteParameterized(
		[]gorqlite.ParameterizedStatement{
			{
				Query:     "UPDATE ImageJob SET Status = ? WHERE Id = ?",
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
