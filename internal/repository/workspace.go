package repository

import (
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"one-click-aks-server/internal/cache"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"

	"github.com/redis/go-redis/v9"
)

type tfWorkspaceRepository struct {
	appConfig *config.Config
	rdb       *redis.Client
}

func NewTfWorkspaceRepository(appConfig *config.Config) entity.WorkspaceRepository {
	return &tfWorkspaceRepository{
		appConfig: appConfig,
		rdb:       cache.NewRedisClient(),
	}
}

// buildWorkspaceEnvironment creates user-specific environment variables for workspace operations
func (t *tfWorkspaceRepository) buildWorkspaceEnvironment(ctx context.Context, storageAccountName string, subscriptionId string) []string {
	userAlias := helper.GetUserAliasFromContext(ctx)

	// Start with current environment
	env := os.Environ()

	// Add user-specific terraform environment variables
	userEnvVars := map[string]string{
		"terraform_directory":  "tf",
		"root_directory":       t.appConfig.RootDir,
		"subscription_id":      subscriptionId,
		"resource_group_name":  t.appConfig.ActLabsHubResourceGroupName,
		"storage_account_name": storageAccountName,
		"container_name":       "repro-project-tf-state-files",
		"tf_state_file_name":   userAlias + "-terraform.tfstate",
	}

	// Add authentication configuration
	if t.appConfig.UseMsi {
		userEnvVars["ARM_USE_MSI"] = "true"
		userEnvVars["ARM_USE_AZUREAD"] = "true"
		userEnvVars["ARM_CLIENT_ID"] = t.appConfig.AzureClientID
		userEnvVars["ARM_MSI_ENDPOINT"] = "http://localhost:" + os.ExpandEnv("$ARM_MSI_API_PROXY_PORT") + "/msi/token"
		userEnvVars["ARM_MSI_API_VERSION"] = "2019-08-01"
		userEnvVars["ARM_SUBSCRIPTION_ID"] = subscriptionId
		userEnvVars["ARM_TENANT_ID"] = t.appConfig.AzureTenantID
	}
	if t.appConfig.UseServicePrincipal {
		userEnvVars["ARM_CLIENT_ID"] = t.appConfig.AzureClientID
		userEnvVars["ARM_CLIENT_SECRET"] = t.appConfig.AzureClientSecret
		userEnvVars["ARM_SUBSCRIPTION_ID"] = subscriptionId
		userEnvVars["ARM_TENANT_ID"] = t.appConfig.AzureTenantID
	}

	// Add user environment variables to the environment slice
	for key, value := range userEnvVars {
		env = append(env, key+"="+value)
	}

	return env
}

// ensureUserDirectory creates user-specific directory and copies tf files
func (t *tfWorkspaceRepository) ensureUserDirectory(ctx context.Context) (string, error) {
	userAlias := helper.GetUserAliasFromContext(ctx)
	userDir := filepath.Join(t.appConfig.RootDir, "user", userAlias)

	if err := os.MkdirAll(userDir, 0755); err != nil {
		logging.LogError(ctx, "failed to create user directory", "dir", userDir, "error", err)
		return "", err
	}

	// Copy terraform files from /tf directory to user directory
	tfSourceDir := filepath.Join(t.appConfig.RootDir, "tf")
	tfUserDir := filepath.Join(userDir, "tf")

	if err := t.copyTerraformFiles(ctx, tfSourceDir, tfUserDir); err != nil {
		logging.LogError(ctx, "failed to copy terraform files", "error", err)
		return "", err
	}

	logging.LogDebug(ctx, "ensured user directory with terraform files", "dir", userDir)
	return userDir, nil
}

// copyTerraformFiles copies terraform files from source to destination
func (t *tfWorkspaceRepository) copyTerraformFiles(ctx context.Context, srcDir, dstDir string) error {
	// Check if terraform files already exist in user directory
	if _, err := os.Stat(filepath.Join(dstDir, "main.tf")); err == nil {
		return nil // Files already exist
	}

	// Create destination directory
	if err := os.MkdirAll(dstDir, 0755); err != nil {
		return err
	}

	// Copy terraform files
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tf") {
			srcPath := filepath.Join(srcDir, entry.Name())
			dstPath := filepath.Join(dstDir, entry.Name())

			if err := t.copyFile(srcPath, dstPath); err != nil {
				logging.LogError(ctx, "failed to copy terraform file",
					"src", srcPath,
					"dst", dstPath,
					"error", err,
				)
				return err
			}
		}
	}

	logging.LogInfo(ctx, "terraform files copied to user directory", "dst", dstDir)
	return nil
}

// copyFile copies a file from src to dst
func (t *tfWorkspaceRepository) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

func (t *tfWorkspaceRepository) List(ctx context.Context, storageAccountName string, subscriptionId string) (string, error) {
	// Create user-specific directory with terraform files
	userDir, err := t.ensureUserDirectory(ctx)
	if err != nil {
		return "", err
	}

	// Build user-specific environment
	userEnv := t.buildWorkspaceEnvironment(ctx, storageAccountName, subscriptionId)

	// Execute workspaces script in user's tf directory
	tfDir := filepath.Join(userDir, "tf")
	cmd := exec.Command(t.appConfig.RootDir+"/scripts/workspaces.sh", "list")
	cmd.Dir = tfDir
	cmd.Env = userEnv

	out, err := cmd.Output()
	return string(out), err
}

func (t *tfWorkspaceRepository) GetListFromRedis(ctx context.Context) (string, error) {
	return t.rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-terraformWorkspaces").Result()
}

func (t *tfWorkspaceRepository) AddListToRedis(ctx context.Context, val string) {
	t.rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-terraformWorkspaces", val, 0)
}

func (t *tfWorkspaceRepository) DeleteListFromRedis(ctx context.Context) {
	t.rdb.Del(ctx, helper.GetUserIDFromContext(ctx)+"-terraformWorkspaces")
}

func (t *tfWorkspaceRepository) Add(ctx context.Context, workspace entity.Workspace) error {
	// Create user-specific directory with terraform files
	userDir, err := t.ensureUserDirectory(ctx)
	if err != nil {
		return err
	}

	// Execute workspaces script in user's tf directory
	tfDir := filepath.Join(userDir, "tf")
	cmd := exec.Command(t.appConfig.RootDir+"/scripts/workspaces.sh", "new", workspace.Name)
	cmd.Dir = tfDir

	_, err = cmd.Output()
	return err
}

func (t *tfWorkspaceRepository) Select(ctx context.Context, workspace entity.Workspace) error {
	// Create user-specific directory with terraform files
	userDir, err := t.ensureUserDirectory(ctx)
	if err != nil {
		return err
	}

	// Execute workspaces script in user's tf directory
	tfDir := filepath.Join(userDir, "tf")
	cmd := exec.Command(t.appConfig.RootDir+"/scripts/workspaces.sh", "select", workspace.Name)
	cmd.Dir = tfDir

	_, err = cmd.Output()
	return err
}

func (t *tfWorkspaceRepository) Delete(ctx context.Context, workspace entity.Workspace) error {
	// Create user-specific directory with terraform files
	userDir, err := t.ensureUserDirectory(ctx)
	if err != nil {
		return err
	}

	// Execute workspaces script in user's tf directory
	tfDir := filepath.Join(userDir, "tf")
	cmd := exec.Command(t.appConfig.RootDir+"/scripts/workspaces.sh", "delete", workspace.Name)
	cmd.Dir = tfDir

	_, err = cmd.Output()
	return err
}

func (t *tfWorkspaceRepository) Resources(ctx context.Context, storageAccountName string, subscriptionId string) (string, error) {
	// Create user-specific directory with terraform files
	userDir, err := t.ensureUserDirectory(ctx)
	if err != nil {
		return "", err
	}

	// Build user-specific environment
	userEnv := t.buildWorkspaceEnvironment(ctx, storageAccountName, subscriptionId)

	// Execute terraform state list in user's tf directory
	tfDir := filepath.Join(userDir, "tf")
	cmd := exec.Command("terraform", "state", "list")
	cmd.Dir = tfDir
	cmd.Env = userEnv

	out, err := cmd.Output()
	return string(out), err
}

func (t *tfWorkspaceRepository) GetResourcesFromRedis(ctx context.Context) (string, error) {
	return t.rdb.Get(ctx, helper.GetUserIDFromContext(ctx)+"-terraformResources").Result()
}

func (t *tfWorkspaceRepository) AddResourcesToRedis(ctx context.Context, val string) {
	t.rdb.Set(ctx, helper.GetUserIDFromContext(ctx)+"-terraformResources", val, 0)
}

func (t *tfWorkspaceRepository) DeleteResourcesFromRedis(ctx context.Context) {
	t.rdb.Del(ctx, helper.GetUserIDFromContext(ctx)+"-terraformResources")
}
