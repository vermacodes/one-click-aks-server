package service

import (
	"bufio"
	"fmt"
	"io"
	"os"

	"one-click-aks-server/internal/entity"

	"golang.org/x/exp/slog"
)

type terraformService struct {
	terraformRepository   entity.TerraformRepository
	labService            entity.LabService
	workspaceService      entity.WorkspaceService
	logStreamService      entity.LogStreamService
	actionStatusService   entity.ActionStatusService
	kVersionService       entity.KVersionService
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
	storageAccountService entity.StorageAccountService,
	authService entity.AuthService,
) entity.TerraformService {
	return &terraformService{
		terraformRepository:   terraformRepository,
		labService:            labService,
		logStreamService:      logStreamService,
		actionStatusService:   actionStatusService,
		kVersionService:       kVersionService,
		workspaceService:      workspaceService,
		storageAccountService: storageAccountService,
		authService:           authService,
	}
}

func (t *terraformService) Init() error {
	lab, err := t.labService.GetLabFromRedis()
	if err != nil {
		return err
	}

	// Invalidate workspace cache
	if err := t.workspaceService.DeleteAllWorkspaceFromRedis(); err != nil {
		return err
	}

	if err := helperTerraformAction(t, lab.Template, "init"); err != nil {
		slog.Error("terraform init failed",
			slog.String("labId", lab.Id),
			slog.String("labName", lab.Name),
			slog.String("labType", lab.Type),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("terraform init failed %s", err.Error())
	}

	return nil
}

func (t *terraformService) Plan(lab entity.LabType) error {
	if err := helperTerraformAction(t, lab.Template, "plan"); err != nil {
		slog.Error("terraform plan failed",
			slog.String("labId", lab.Id),
			slog.String("labName", lab.Name),
			slog.String("labType", lab.Type),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("terraform plan failed %s", err.Error())
	}

	return nil
}

func (t *terraformService) Apply(lab entity.LabType) error {

	// Invalidate workspace cache
	if err := t.workspaceService.DeleteAllWorkspaceFromRedis(); err != nil {
		return err
	}

	// if lab is assignment, update assignment status to InProgress
	if lab.Type == "assignment" {
		userId := os.Getenv("ARM_USER_PRINCIPAL_NAME")
		if err := t.UpdateAssignment(userId, lab.Id, "InProgress"); err != nil {
			return err
		}
	}

	if err := helperTerraformAction(t, lab.Template, "apply"); err != nil {
		slog.Error("terraform apply failed",
			slog.String("labId", lab.Id),
			slog.String("labName", lab.Name),
			slog.String("labType", lab.Type),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("terraform apply failed %s", err.Error())
	}

	return t.Extend(lab, "apply")

}

func (t *terraformService) Extend(lab entity.LabType, mode string) error {
	slog.Info("running extend script",
		slog.String("labId", lab.Id),
		slog.String("labName", lab.Name),
		slog.String("labType", lab.Type),
		slog.String("mode", mode),
	)

	// Getting back redacted values
	if lab.ExtendScript == "redacted" {
		lab, err := t.labService.GetProtectedLab(lab.Type, lab.Id)
		if err != nil {
			return err
		}

		err = helperExecuteScript(t, lab.ExtendScript, mode)
		if err != nil {
			return err
		}

		// if lab is assignment and mode is validate,
		// update assignment status to completed if the validation was good.
		if lab.Type == "assignment" && mode == "validate" {
			userId := os.Getenv("ARM_USER_PRINCIPAL_NAME")
			if err := t.UpdateAssignment(userId, lab.Id, "Completed"); err != nil {
				return fmt.Errorf("validation was successful but not able to update status, try again")
			}
		}

		return nil
	}

	return helperExecuteScript(t, lab.ExtendScript, mode)
}

func (t *terraformService) Destroy(lab entity.LabType) error {
	slog.Info("terraform destroy",
		slog.String("labId", lab.Id),
		slog.String("labName", lab.Name),
		slog.String("labType", lab.Type),
	)
	// Invalidate workspace cache
	if err := t.workspaceService.DeleteAllWorkspaceFromRedis(); err != nil {
		return err
	}

	if err := t.Extend(lab, "destroy"); err != nil {
		return err
	}

	if err := helperTerraformAction(t, lab.Template, "destroy"); err != nil {
		slog.Error("terraform destroy failed",
			slog.String("labId", lab.Id),
			slog.String("labName", lab.Name),
			slog.String("labType", lab.Type),
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("terraform destroy failed %s", err.Error())
	}

	return nil
}

func (t *terraformService) UpdateAssignment(userId string, labId string, status string) error {
	slog.Info("updating assignment status",
		slog.String("userId", userId),
		slog.String("labId", labId),
		slog.String("status", status),
	)
	if err := t.terraformRepository.UpdateAssignment(userId, labId, status); err != nil {
		slog.Error("not able to update assignment status",
			slog.String("userId", userId),
			slog.String("labId", labId),
			slog.String("status", status),
			slog.String("error", err.Error()),
		)
		return err
	}

	return nil
}

func helperTerraformAction(t *terraformService, tfvar entity.TfvarConfigType, action string) error {

	storageAccountName, err := t.storageAccountService.GetStorageAccountName()
	if err != nil {
		return err
	}

	for i, cluster := range tfvar.KubernetesClusters {
		if !t.kVersionService.DoesVersionExist(cluster.KubernetesVersion) {
			tfvar.KubernetesClusters[i].KubernetesVersion = t.kVersionService.GetDefaultVersion()
		}
	}

	cmd, rPipe, wPipe, err := t.terraformRepository.TerraformAction(tfvar, action, storageAccountName)
	if err != nil {
		return err
	}

	// Getting current logs.
	if _, err := t.logStreamService.GetLogs(); err != nil {
		return err
	}

	// GO routine that takes care of running command and moving logs to redis.
	go func(input io.ReadCloser) {
		in := bufio.NewScanner(input)

		for in.Scan() {
			// Appending logs to redis.
			t.logStreamService.AppendLogs(fmt.Sprintf("%s\n", in.Text()))
		}
		input.Close()
	}(rPipe)

	err = cmd.Wait()
	wPipe.Close()

	return err
}

func helperExecuteScript(t *terraformService, script string, mode string) error {
	storageAccountName, err := t.storageAccountService.GetStorageAccountName()
	if err != nil {
		slog.Error("not able to get storage account name",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("not able to get storage account name")
	}

	cmd, rPipe, wPipe, err := t.terraformRepository.ExecuteScript(script, mode, storageAccountName)
	if err != nil {
		slog.Error("not able to run terraform script",
			slog.String("error", err.Error()),
		)
		return fmt.Errorf("not able to run script")
	}

	// GO routine that takes care of running command and moving logs to redis.
	go func(input io.ReadCloser) {
		in := bufio.NewScanner(input)

		for in.Scan() {
			t.logStreamService.AppendLogs(fmt.Sprintf("%s\n", in.Text()))
		}
		input.Close()
	}(rPipe)

	err = cmd.Wait()
	wPipe.Close()

	return err
}
