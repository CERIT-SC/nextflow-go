package main

import (
	"fmt"

	"nextflow-go/pkg/kube"
)

func main() {
	fmt.Println("Running Nextflow K8s Job...")
	kube.Execute(false)
}

