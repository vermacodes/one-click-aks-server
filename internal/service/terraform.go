package service

import (
	"bufio"
	"context"
	"fmt"
	"io"

	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"
)

type terraformService struct {
	terraformRepository   entity.TerraformRepository
	labService            entity.LabService
	workspaceService      entity.WorkspaceService
	logStreamService      entity.LogStreamService
	actionStatusService   entity.ActionStatusService
	kVersionService       entity.KVersionService
	aroVersionService     entity.AROVersionService
	storageAccountService entity.StorageAccountService // Some information is needed from storage account service.
	authService           entity.AuthService
}

func NewTerraformService(
	terraformRepository entity.TerraformRepository,
	labService entity.LabService,
	workspaceService entity.WorkspaceService,
	logStreamService entity.LogStreamService,
	actionStatusService entity.ActionStatusService,
	kVersionService entity.KVersionService,
	aroVersionService entity.AROVersionService,
	storageAccountService entity.StorageAccountService,
	authService entity.AuthService,
) entity.TerraformService {
	return &terraformService{
		terraformRepository:   terraformRepository,
		labService:            labService,
		logStreamService:      logStreamService,
		actionStatusService:   actionStatusService,
		kVersionService:       kVersionService,
		aroVersionService:     aroVersionService,
		workspaceService:      workspaceService,
		storageAccountService: storageAccountService,
		authService:           authService,
	}
}

func (t *terraformService) Init(ctx context.Context) error {
	logging.LogInfo(ctx, "running terraform init")
	lab, err := t.labService.GetLabFromRedis(ctx)
	if err != nil {
		return err
	}

	if err := helperTerraformAction(ctx, t, lab.Template, "init"); err != nil {
		logging.LogError(ctx, "terraform init failed",
			"lab_id", lab.Id,
			"lab_name", lab.Name,
			"lab_type", lab.Type,
			"error", err.Error(),
		)
		return fmt.Errorf("terraform init failed %s", err.Error())
	}

	// Invalidate workspace cache
	if err := t.workspaceService.DeleteAllWorkspaceFromRedis(ctx); err != nil {
		return err
	}

	return nil
}

func (t *terraformService) Plan(ctx context.Context, lab entity.LabType) error {

	logging.LogInfo(ctx, "running terraform plan", "lab_id", lab.Id)

	if err := helperTerraformAction(ctx, t, lab.Template, "plan"); err != nil {
		logging.LogError(ctx, "terraform plan failed",
			"lab_id", lab.Id,
			"lab_name", lab.Name,
			"lab_type", lab.Type,
			"error", err.Error(),
		)
		return fmt.Errorf("terraform plan failed %s", err.Error())
	}

	return nil
}

func (t *terraformService) Apply(ctx context.Context, lab entity.LabType) error {

	// if lab is assignment, update assignment status to InProgress
	if lab.Type == "assignment" {
		userId := helper.GetUserIDFromContext(ctx)
		if err := t.UpdateAssignment(ctx, userId, lab.Id, "InProgress"); err != nil {
			return fmt.Errorf("not able to update assignment status, try again")
		}
	}

	if err := helperTerraformAction(ctx, t, lab.Template, "apply"); err != nil {
		logging.LogError(ctx, "terraform apply failed",
			"lab_id", lab.Id,
			"lab_name", lab.Name,
			"lab_type", lab.Type,
			"error", err.Error(),
		)
		return fmt.Errorf("terraform apply failed %s", err.Error())
	}

	// Invalidate workspace cache
	if err := t.workspaceService.DeleteAllWorkspaceFromRedis(ctx); err != nil {
		return err
	}

	return t.Extend(ctx, lab, "apply")

}

func (t *terraformService) Extend(ctx context.Context, lab entity.LabType, mode string) error {
	logging.LogInfo(ctx, "running extend script",
		"lab_id", lab.Id,
		"lab_name", lab.Name,
		"lab_type", lab.Type,
		"mode", mode,
	)

	// Getting back redacted values
	if lab.ExtendScript == "redacted" {
		lab, err := t.labService.GetProtectedLab(ctx, lab.Type, lab.Id)
		if err != nil {
			return err
		}

		err = helperExecuteScript(ctx, t, lab.ExtendScript, mode)
		if err != nil {
			return err
		}

		// if lab is assignment and mode is validate,
		// update assignment status to completed if the validation was good.
		if lab.Type == "assignment" && mode == "validate" {
			userId := helper.GetUserIDFromContext(ctx)
			if err := t.UpdateAssignment(ctx, userId, lab.Id, "Completed"); err != nil {
				return fmt.Errorf("validation was successful but not able to update status, try again")
			}
		}

		// if lab is challenge and mode is validate,
		// update challenge status to completed if the validation was good.
		if lab.Type == "challenge" && mode == "validate" {
			userId := helper.GetUserIDFromContext(ctx)
			if err := t.UpdateChallenge(ctx, userId, lab.Id, "completed"); err != nil { // it is completed. not Completed.
				return fmt.Errorf("validation was successful but not able to update status, try again")
			}
		}

		return nil
	}

	return helperExecuteScript(ctx, t, lab.ExtendScript, mode)
}

func (t *terraformService) Destroy(ctx context.Context, lab entity.LabType) error {
	logging.LogInfo(ctx, "terraform destroy",
		"lab_id", lab.Id,
		"lab_name", lab.Name,
		"lab_type", lab.Type,
	)

	if err := t.Extend(ctx, lab, "destroy"); err != nil {
		return err
	}

	if err := helperTerraformAction(ctx, t, lab.Template, "destroy"); err != nil {
		logging.LogError(ctx, "terraform destroy failed",
			"lab_id", lab.Id,
			"lab_name", lab.Name,
			"lab_type", lab.Type,
			"error", err.Error(),
		)
		return fmt.Errorf("terraform destroy failed %s", err.Error())
	}

	// Invalidate workspace cache
	if err := t.workspaceService.DeleteAllWorkspaceFromRedis(ctx); err != nil {
		return err
	}

	return nil
}

func (t *terraformService) UpdateAssignment(ctx context.Context, userId string, labId string, status string) error {
	logging.LogInfo(ctx, "updating assignment status",
		"user_id", userId,
		"lab_id", labId,
		"status", status,
	)
	if err := t.terraformRepository.UpdateAssignment(ctx, userId, labId, status); err != nil {
		logging.LogError(ctx, "not able to update assignment status",
			"user_id", userId,
			"lab_id", labId,
			"status", status,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func (t *terraformService) UpdateChallenge(ctx context.Context, userId string, labId string, status string) error {
	logging.LogInfo(ctx, "updating challenge status",
		"user_id", userId,
		"lab_id", labId,
		"status", status,
	)
	if err := t.terraformRepository.UpdateChallenge(ctx, userId, labId, status); err != nil {
		logging.LogError(ctx, "not able to update challenge status",
			"user_id", userId,
			"lab_id", labId,
			"status", status,
			"error", err.Error(),
		)
		return err
	}

	return nil
}

func helperTerraformAction(ctx context.Context, t *terraformService, tfvar entity.TfvarConfigType, action string) error {

	storageAccountName, err := t.storageAccountService.GetStorageAccountName(ctx)
	if err != nil {
		return err
	}

	helperEnsureKubernetesVersion(ctx, t, &tfvar)

	helperEnsureAroVersion(ctx, t, &tfvar)

	helperEnsureAro(ctx, &tfvar)

	cmd, rPipe, wPipe, err := t.terraformRepository.TerraformAction(ctx, tfvar, action, storageAccountName, t.authService.GetSubscriptionId(ctx))
	if err != nil {
		return err
	}

	// Getting current logs.
	if _, err := t.logStreamService.GetLogs(ctx); err != nil {
		return err
	}

	// GO routine that takes care of running command and moving logs to redis.
	go func(input io.ReadCloser) {
		in := bufio.NewScanner(input)

		for in.Scan() {
			// Appending logs to redis.
			t.logStreamService.AppendLogs(ctx, fmt.Sprintf("%s\n", in.Text()))
		}
		input.Close()
	}(rPipe)

	err = cmd.Wait()
	wPipe.Close()

	return err
}

// Ensure that the version of kubernetes exists.
// if the version is old, it sets the version to current default.
// best known use case is when a lab is created with an old version of kubernetes.
func helperEnsureKubernetesVersion(ctx context.Context, t *terraformService, tfvar *entity.TfvarConfigType) {
	for i, cluster := range tfvar.KubernetesClusters {
		if !t.kVersionService.DoesVersionExist(ctx, cluster.KubernetesVersion) {
			tfvar.KubernetesClusters[i].KubernetesVersion = t.kVersionService.GetDefaultVersion(ctx)
		}
	}
}

// Ensure that the version of ARO exists.
func helperEnsureAroVersion(ctx context.Context, t *terraformService, tfvar *entity.TfvarConfigType) {
	for i, cluster := range tfvar.AroClusters {
		if !t.aroVersionService.DoesVersionExist(ctx, cluster.Version) {
			tfvar.AroClusters[i].Version = t.aroVersionService.GetDefaultAROVersion(ctx)
		}
	}
}

// Ensure ARO exists.
// ARO is introduced late, so the older lab objects will not have it in them.
// We just need to add empty array to tfvar object.
func helperEnsureAro(ctx context.Context, tfvar *entity.TfvarConfigType) {
	if tfvar.AroClusters == nil {
		tfvar.AroClusters = []entity.TfvarAroClusterType{}
	}
}

func helperExecuteScript(ctx context.Context, t *terraformService, script string, mode string) error {
	storageAccountName, err := t.storageAccountService.GetStorageAccountName(ctx)
	if err != nil {
		logging.LogError(ctx, "not able to get storage account name",
			"error", err,
		)
		return fmt.Errorf("not able to get storage account name")
	}

	cmd, rPipe, wPipe, err := t.terraformRepository.ExecuteScript(ctx, script, mode, storageAccountName, t.authService.GetSubscriptionId(ctx))
	if err != nil {
		logging.LogError(ctx, "not able to run terraform script",
			"error", err,
		)
		return fmt.Errorf("not able to run script")
	}

	// GO routine that takes care of running command and moving logs to redis.
	go func(input io.ReadCloser) {
		in := bufio.NewScanner(input)

		for in.Scan() {
			t.logStreamService.AppendLogs(ctx, fmt.Sprintf("%s\n", in.Text()))
		}
		input.Close()
	}(rPipe)

	err = cmd.Wait()
	wPipe.Close()

	return err
}
