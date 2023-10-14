package cftemplates

import "embed"

//go:embed *.yaml
var templates embed.FS

func Templates() embed.FS {
	return templates
}

//go:embed vpc.yaml
var vpc string

func VPC() string {
	return vpc
}

//go:embed nodegroup.yaml
var nodeGroup string

func NodeGroup() string {
	return nodeGroup
}
