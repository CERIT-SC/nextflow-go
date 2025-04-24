package main

import (
	"fmt"
        "os"

	"nextflow-go/pkg/kube"
)

func main() {
        if len(os.Args) == 1 {
                fmt.Println("usage: nextflow-go [all nextflow arguments]")
                os.Exit(0)
        }
	fmt.Println("Running Nextflow K8s Job...")
	kube.Execute(false)
}

