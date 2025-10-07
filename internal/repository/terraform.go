package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"

	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/helper"
	"one-click-aks-server/internal/logging"

	"github.com/Rican7/conjson"
	"github.com/Rican7/conjson/transform"
)

type terraformRepository struct {
	appConfig *config.Config
}

func NewTerraformRepository(appConfig *config.Config) entity.TerraformRepository {
	return &terraformRepository{
		appConfig: appConfig,
	}
}

// buildUserEnvironment creates user-specific environment variables
func (t *terraformRepository) buildUserEnvironment(ctx context.Context, tfvar entity.TfvarConfigType, storageAccountName string, subscriptionId string) []string {
	userAlias := helper.GetUserAliasFromContext(ctx)

	// Start with current environment
	env := os.Environ()

	// Add user-specific terraform environment variables
	userEnvVars := map[string]string{
		"terraform_directory":  "tf",
		"root_directory":       os.ExpandEnv("$ROOT_DIR"),
		"subscription_id":      subscriptionId,
		"resource_group_name":  t.appConfig.ActLabsHubResourceGroupName,
		"storage_account_name": storageAccountName,
		"container_name":       "repro-project-tf-state-files",
		"tf_state_file_name":   userAlias + "-terraform.tfstate",
		"TF_VAR_aro_rp_first_party_service_principal_id": t.appConfig.AroRpFirstPartySpID,
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

	// Add terraform variables from tfvar struct
	tr := reflect.TypeOf(tfvar)
	for i := 0; i < tr.NumField(); i++ {
		field := reflect.TypeOf(tfvar).Field(i)
		value := reflect.ValueOf(tfvar).Field(i)

		encoded, _ := json.Marshal(conjson.NewMarshaler(value.Interface(), transform.ConventionalKeys()))

		logging.LogDebug(ctx, "Field :"+field.Name+" Encoded String : "+string(encoded))

		if string(encoded) != "null" {
			userEnvVars["TF_VAR_"+helper.CamelToConventional(field.Name)] = string(encoded)
		}
	}

	// Add user environment variables to the environment slice
	for key, value := range userEnvVars {
		env = append(env, key+"="+value)
	}

	return env
}

// ensureUserDirectory creates user-specific directory and copies terraform files
func (t *terraformRepository) ensureUserDirectory(ctx context.Context) (string, error) {
	userAlias := helper.GetUserAliasFromContext(ctx)
	userDir := filepath.Join(os.ExpandEnv("$ROOT_DIR"), "user", userAlias)

	if err := os.MkdirAll(userDir, 0755); err != nil {
		logging.LogError(ctx, "failed to create user directory", "dir", userDir, "error", err)
		return "", err
	}

	// Copy terraform files from /tf directory to user directory
	tfSourceDir := filepath.Join(os.ExpandEnv("$ROOT_DIR"), "tf")
	tfUserDir := filepath.Join(userDir, "tf")

	if err := t.copyTerraformFiles(ctx, tfSourceDir, tfUserDir); err != nil {
		logging.LogError(ctx, "failed to copy terraform files", "error", err)
		return "", err
	}

	logging.LogDebug(ctx, "ensured user directory with terraform files", "dir", userDir)
	return userDir, nil
}

// copyTerraformFiles copies terraform files from source to destination
func (t *terraformRepository) copyTerraformFiles(ctx context.Context, srcDir, dstDir string) error {
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
func (t *terraformRepository) copyFile(src, dst string) error {
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

func (t *terraformRepository) TerraformAction(ctx context.Context, tfvar entity.TfvarConfigType, action string, storageAccountName string, subscriptionId string) (*exec.Cmd, *os.File, *os.File, error) {

	// Create user-specific directory with terraform files
	userDir, err := t.ensureUserDirectory(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	// Build user-specific environment
	userEnv := t.buildUserEnvironment(ctx, tfvar, storageAccountName, subscriptionId)

	// Execute terraform script with appropriate action in user's tf directory
	tfDir := filepath.Join(userDir, "tf")
	cmd := exec.Command(os.ExpandEnv("$ROOT_DIR")+"/scripts/terraform.sh", action)
	cmd.Dir = tfDir
	cmd.Env = userEnv

	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		return cmd, rPipe, wPipe, err
	}

	cmd.Stdout = wPipe
	cmd.Stderr = wPipe
	if err := cmd.Start(); err != nil {
		return cmd, rPipe, wPipe, err
	}

	// Return stuff to the service.
	return cmd, rPipe, wPipe, nil
}

func (t *terraformRepository) ExecuteScript(ctx context.Context, script string, mode string, storageAccountName string, subscriptionId string) (*exec.Cmd, *os.File, *os.File, error) {
	// Create user-specific directory with terraform files
	userDir, err := t.ensureUserDirectory(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	// Build user-specific environment with script mode
	userEnv := t.buildUserEnvironment(ctx, entity.TfvarConfigType{}, storageAccountName, subscriptionId)
	userEnv = append(userEnv, "SCRIPT_MODE="+mode)

	// Execute script in user's tf directory
	tfDir := filepath.Join(userDir, "tf")
	cmd := exec.Command("bash", "-c", "echo '"+script+"' | base64 -d | dos2unix | bash")
	cmd.Dir = tfDir
	cmd.Env = userEnv

	rPipe, wPipe, err := os.Pipe()
	if err != nil {
		return cmd, rPipe, wPipe, err
	}
	cmd.Stdout = wPipe
	cmd.Stderr = wPipe
	if err := cmd.Start(); err != nil {
		return cmd, rPipe, wPipe, err
	}

	// Return stuff to the service.
	return cmd, rPipe, wPipe, nil
}

func (t *terraformRepository) UpdateAssignment(ctx context.Context, userId string, labId string, status string) error {

	// http call to actlabs-hub
	req, err := http.NewRequest("PUT", t.appConfig.ActlabsHubURL+"assignment/"+userId+"/"+labId+"/"+status, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("ACTLABS_AUTH_TOKEN"))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("ProtectedLabSecret", entity.ProtectedLabSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("not able to update assignment status")
	}

	return nil
}

func (t *terraformRepository) UpdateChallenge(ctx context.Context, userId string, labId string, status string) error {

	// http call to actlabs-hub
	req, err := http.NewRequest("PUT", t.appConfig.ActlabsHubURL+"challenge/"+userId+"/"+labId+"/"+status, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("ACTLABS_AUTH_TOKEN"))
	req.Header.Set("Accept", "application/json")
	req.Header.Set("ProtectedLabSecret", entity.ProtectedLabSecret)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("not able to update challenge status")
	}

	return nil
}
