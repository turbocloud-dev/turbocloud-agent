package main

type Environment struct {
	Id          string
	Name        string
	Branch      string
	Domains     []string
	Port        string
	Deployments []Deployment
	ImageId     string
}
