package entity

import (
	"context"
	"os"
	"os/exec"
)

type TerraformService interface {
	// Terraform Init
	Init(ctx context.Context) error

	// Terraform Ensure Init
	// This will send a non-existent action
	// The script will execute and tf init will run if not already.
	// Clever :D
	EnsureInit(ctx context.Context) error

	// Streams logs
	Plan(ctx context.Context, lab LabType) error

	// Apply terraform and then run extend script if any
	// This streams logs.
	Apply(ctx context.Context, lab LabType) error

	// Apply terraform and then run extend script if any
	// This is async and doesn't stream logs.
	// ApplyAsync(LabType) (TerraformOperation, error)

	// Executes shell script to run extension of infra.
	// runs against selected workspace. This doesn't send any response body
	// and logs are streamed.
	Extend(ctx context.Context, lab LabType, mode string) error

	// Executes shell script to run extension of infra.
	// runs against selected workspace. This is async and doesn't stream logs.
	// ExtendAsync(LabType, string) (TerraformOperation, error)

	// destroy the resources in current workspace.
	// Streams logs
	Destroy(ctx context.Context, lab LabType) error

	// destroy the resources in current workspace.
	// This is async and doesn't stream logs.
	// DestroyAsync(LabType) (TerraformOperation, error)

	// Executes shell script to run validation against infra.
	// runs against selected workspace. This doesn't send any response body
	// and logs are streamed.
	// Validate(LabType) error

	UpdateAssignment(ctx context.Context, userId string, labId string, status string) error
	UpdateChallenge(ctx context.Context, userId string, labId string, status string) error
}

type TerraformRepository interface {
	TerraformAction(ctx context.Context, tfvar TfvarConfigType, action string, storageAccountName string, subscriptionId string) (*exec.Cmd, *os.File, *os.File, error)
	ExecuteScript(ctx context.Context, script string, mode string, storageAccountName string, subscriptionId string) (*exec.Cmd, *os.File, *os.File, error)

	UpdateAssignment(ctx context.Context, userId string, labId string, status string) error
	UpdateChallenge(ctx context.Context, userId string, labId string, status string) error
}
