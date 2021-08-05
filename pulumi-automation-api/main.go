package main

import (
	"context"
	"fmt"
	"os"

	"github.com/pulumi/pulumi/sdk/v3/go/auto"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optdestroy"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optrefresh"
	"github.com/pulumi/pulumi/sdk/v3/go/auto/optup"
)

func StackDestroy(ctx context.Context, stack *auto.Stack) error {
	fmt.Println("Starting stack destroy")
	_, err := stack.Destroy(ctx, optdestroy.ProgressStreams(os.Stdout))
	if err != nil {
		return err
	}
	fmt.Println("Stack successfully destroyed")
	return nil
}

func Stack(ctx context.Context, projectName, repoURL, branch, projectPath string, destroy bool, secrets map[string]string) (*auto.Stack, error) {
	var stack auto.Stack
	var err error

	stackName := auto.FullyQualifiedStackName("dirien", projectName, "dev")
	fmt.Println(stackName)
	repo := auto.GitRepo{
		URL:         repoURL,
		Branch:      branch,
		ProjectPath: projectPath,
	}
	stack, err = auto.UpsertStackRemoteSource(ctx, stackName, repo)
	if err != nil {
		return nil, err
	}

	for _, key := range secrets {
		stack.SetConfig(ctx, key, auto.ConfigValue{Value: os.Getenv(secrets[key]), Secret: true})
	}

	fmt.Println("Starting refresh")
	_, err = stack.Refresh(ctx, optrefresh.ProgressStreams(os.Stdout))
	if err != nil {
		return nil, err
	}
	fmt.Println("Refresh succeeded!")

	if destroy {
		return &stack, nil
	}
	fmt.Println("Starting update")
	_, err = stack.Up(ctx, optup.ProgressStreams(os.Stdout))
	if err != nil {
		return nil, err
	}
	fmt.Println("Update succeeded!")
	return &stack, nil
}

func main() {
	destroy := false
	argsWithoutProg := os.Args[1:]
	if len(argsWithoutProg) > 0 {
		if argsWithoutProg[0] == "destroy" {
			destroy = true
		}
	}
	ctx := context.Background()
	infra, err := Stack(ctx, "00-infrastructure--aa", "https://github.com/dirien/pulumi-aiven-openfaas",
		"refs/remotes/origin/automation",
		"00-infrastructure", destroy, map[string]string{
			"linode:token": "LINODE_TOKEN",
		})
	if err != nil {
		fmt.Printf("Failed to create or select stack: %v\n", err)
		os.Exit(1)
	}

	aiven, err := Stack(ctx, "01-aiven-aa", "https://github.com/dirien/pulumi-aiven-openfaas",
		"refs/remotes/origin/automation",
		"01-aiven", destroy, map[string]string{
			"aiven:apiToken": "AIVEN_TOKEN",
		})
	if err != nil {
		fmt.Printf("Failed to create or select stack: %v\n", err)
		os.Exit(1)
	}

	openfaas, err := Stack(ctx, "02-openfaas-aa", "https://github.com/dirien/pulumi-aiven-openfaas",
		"refs/remotes/origin/automation",
		"02-openfaas", destroy, map[string]string{
			"openfaas": "LICENSE",
		})
	if err != nil {
		fmt.Printf("Failed to create or select stack: %v\n", err)
		os.Exit(1)
	}

	//Destroy the stacks in the right order.
	if destroy {
		err := StackDestroy(ctx, openfaas)
		if err != nil {
			fmt.Printf("Failed to delete stack: %v\n", err)
			os.Exit(1)
		}
		err = StackDestroy(ctx, aiven)
		if err != nil {
			fmt.Printf("Failed to delete stack: %v\n", err)
			os.Exit(1)
		}
		err = StackDestroy(ctx, infra)
		if err != nil {
			fmt.Printf("Failed to delete stack: %v\n", err)
			os.Exit(1)
		}
	}
}
