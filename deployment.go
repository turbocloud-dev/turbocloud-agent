package main

type DeploymentStatus int

const (
	ToDeploy   DeploymentStatus = iota + 1 // EnumIndex = 1
	InProgress                             // EnumIndex = 2
	Deployed                               // EnumIndex = 3
)

func (d DeploymentStatus) String() string {
	return [...]string{"ToDeploy", "InProgress", "Deployed"}[d-1]
}

func (d DeploymentStatus) EnumIndex() int {
	return int(d)
}

type Deployment struct {
	Id        string
	Status    DeploymentStatus
	MachineId string
}
