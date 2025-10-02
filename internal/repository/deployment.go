package repository

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"
	"one-click-aks-server/internal/logging"

	"github.com/redis/go-redis/v9"
)

type deploymentRepository struct {
	appConfig *config.Config
	auth      *auth.Auth
	rdb       *redis.Client
}

func NewDeploymentRepository(appConfig *config.Config, auth *auth.Auth, rdb *redis.Client) entity.DeploymentRepository {
	return &deploymentRepository{
		appConfig: appConfig,
		auth:      auth,
		rdb:       rdb,
	}
}

func (d *deploymentRepository) GetDeployments(ctx context.Context) ([]entity.Deployment, error) {
	return nil, nil
}

func (d *deploymentRepository) GetMyDeployments(ctx context.Context, userId string, subscriptionId string) ([]entity.Deployment, error) {
	deployments := []entity.Deployment{}

	// check if user deployments already exist in redis
	deploymentsString, err := d.rdb.Get(ctx, userId+"-deployments").Result()
	if err != nil {
		logging.LogError(ctx, "error getting deployments from redis continue to get from table storage ",
			"error", err,
		)
	}
	if deploymentsString != "" {
		if err := json.Unmarshal([]byte(deploymentsString), &deployments); err == nil {
			return deployments, nil
		}
		logging.LogError(ctx, "error unmarshal deployment found in redis continue to get from table storage ",
			"error", err,
		)
	}

	url := d.appConfig.ActlabsHubURL + "deployments"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		logging.LogError(ctx, "error getting deployments ", err)
		return nil, err
	}

	armAccessToken, err := d.auth.GetARMAccessToken()
	if err != nil {
		logging.LogError(ctx, "error getting arm access token ", err)
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+armAccessToken)
	req.Header.Set("x-ms-client-principal-name", d.appConfig.ArmUserPrincipalName)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("ProtectedLabSecret", entity.ProtectedLabSecret)

	client := &http.Client{
		Timeout: time.Second * time.Duration(d.appConfig.HttpRequestTimeoutSeconds),
	}
	resp, err := client.Do(req)
	if err != nil {
		logging.LogError(ctx, "error getting deployments ", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.LogError(ctx, "error getting deployments ", err)
		return nil, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
		logging.LogError(ctx, "error unmarshal deployments ", err)
		return nil, err
	}

	// save deployments to redis
	marshalledDeployments, err := json.Marshal(deployments)
	if err != nil {
		logging.LogError(ctx, "error occurred marshalling the deployments record.",
			"error", err,
		)

		return deployments, err
	}

	err = d.rdb.Set(ctx, userId+"-deployments", marshalledDeployments, 0).Err()
	if err != nil {
		logging.LogError(ctx, "error occurred saving the deployments record to redis.",
			"error", err,
		)
	}

	return deployments, nil
}

func (d *deploymentRepository) GetDeployment(ctx context.Context, userId string, workspace string, subscriptionId string) (entity.Deployment, error) {
	deployment := entity.Deployment{}

	// check if deployment already exist in redis
	deploymentString, err := d.rdb.Get(ctx, userId+"-"+subscriptionId+"-"+workspace).Result()
	if err != nil {
		logging.LogError(ctx, "error getting deployment from redis continue to get from table storage ",
			"subscription_id", subscriptionId,
			"workspace", workspace,
			"error", err,
		)
	}
	if deploymentString != "" {
		if err := json.Unmarshal([]byte(deploymentString), &deployment); err == nil {
			return deployment, nil
		}
		logging.LogError(ctx, "error unmarshal deployment found in redis continue to get from table storage ",
			"subscription_id", subscriptionId,
			"workspace", workspace,
			"error", err,
		)
	}

	deployments, err := d.GetMyDeployments(ctx, userId, subscriptionId)
	if err != nil {
		logging.LogError(ctx, "error getting deployments ", err)
		return entity.Deployment{}, err
	}

	for _, deployment := range deployments {
		if deployment.DeploymentWorkspace == workspace &&
			deployment.DeploymentUserId == userId &&
			deployment.DeploymentSubscriptionId == subscriptionId {

			// save deployment to redis
			marshalledDeployment, err := json.Marshal(deployment)
			if err != nil {
				logging.LogError(ctx, "error occurred marshalling the deployment record.",
					"subscription_id", subscriptionId,
					"workspace", workspace,
					"error", err,
				)

				return deployment, err
			}

			err = d.rdb.Set(ctx, userId+"-"+subscriptionId+"-"+workspace, marshalledDeployment, 0).Err()
			if err != nil {
				logging.LogError(ctx, "error occurred saving the deployment record to redis.",
					"subscription_id", subscriptionId,
					"workspace", workspace,
					"error", err,
				)
			}

			return deployment, nil
		}
	}

	return entity.Deployment{}, errors.New("deployment not found")
}

func (d *deploymentRepository) UpsertDeployment(ctx context.Context, deployment entity.Deployment) error {
	url := d.appConfig.ActlabsHubURL + "deployments"
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		logging.LogError(ctx, "error creating new request ",
			"error", err,
		)
		return err
	}

	armAccessToken, err := d.auth.GetARMAccessToken()
	if err != nil {
		logging.LogError(ctx, "error getting arm access token ", "error", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+armAccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ms-client-principal-name", d.appConfig.ArmUserPrincipalName)
	req.Header.Set("ProtectedLabSecret", entity.ProtectedLabSecret)

	marshalledDeployment, err := json.Marshal(deployment)
	if err != nil {
		logging.LogError(ctx, "error occurred marshalling the deployment.", "error", err)
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(marshalledDeployment))

	client := &http.Client{
		Timeout: time.Second * time.Duration(d.appConfig.HttpRequestTimeoutSeconds),
	}

	resp, err := client.Do(req)
	if err != nil {
		logging.LogError(ctx, "error upserting deployments ", "error", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logging.LogError(ctx, "error upserting deployments ", "status_code", resp.StatusCode)
		return errors.New("unexpected status code")
	}

	// save deployment to redis
	if err := d.rdb.Set(ctx, deployment.DeploymentUserId+"-"+deployment.DeploymentSubscriptionId+"-"+deployment.DeploymentWorkspace, marshalledDeployment, 0).Err(); err != nil {
		logging.LogError(ctx, "error occurred saving the deployment record to redis.",
			"subscription_id", deployment.DeploymentSubscriptionId,
			"workspace", deployment.DeploymentWorkspace,
			"error", err,
		)

		// if not able to add deployment, delete existing deployment from redis if any
		if err := d.rdb.Del(ctx, deployment.DeploymentUserId+"-"+deployment.DeploymentSubscriptionId+"-"+deployment.DeploymentWorkspace).Err(); err != nil {
			logging.LogError(ctx, "error occurred deleting the deployment record from redis.",
				"subscription_id", deployment.DeploymentSubscriptionId,
				"workspace", deployment.DeploymentWorkspace,
				"error", err,
			)

			return err
		}
	}

	// delete deployments for user from redis
	if err := d.rdb.Del(ctx, deployment.DeploymentUserId+"-deployments").Err(); err != nil {
		logging.LogError(ctx, "error occurred deleting the deployments record from redis.",
			"subscription_id", deployment.DeploymentSubscriptionId,
			"workspace", deployment.DeploymentWorkspace,
			"error", err,
		)

		return err
	}

	return nil
}

func (d *deploymentRepository) DeleteDeployment(ctx context.Context, userId string, workspace string, subscriptionId string) error {
	url := d.appConfig.ActlabsHubURL + "deployments/" + subscriptionId + "/" + workspace

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		logging.LogError(ctx, "error creating new request ", "error", err)
		return err
	}

	armAccessToken, err := d.auth.GetARMAccessToken()
	if err != nil {
		logging.LogError(ctx, "error getting arm access token ", "error", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+armAccessToken)
	req.Header.Set("x-ms-client-principal-name", d.appConfig.ArmUserPrincipalName)
	req.Header.Set("ProtectedLabSecret", entity.ProtectedLabSecret)

	client := &http.Client{
		Timeout: time.Second * time.Duration(d.appConfig.HttpRequestTimeoutSeconds),
	}

	resp, err := client.Do(req)
	if err != nil {
		logging.LogError(ctx, "error deleting deployments ", "error", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		logging.LogError(ctx, "error deleting deployments ", "status_code", resp.StatusCode)
		return errors.New("unexpected status code")
	}

	// delete deployment from redis
	if err := d.rdb.Del(ctx, userId+"-"+subscriptionId+"-"+workspace).Err(); err != nil {
		logging.LogError(ctx, "error occurred deleting the deployment record from redis.",
			"subscription_id", subscriptionId,
			"workspace", workspace,
			"error", err,
		)

		return err
	}

	// delete deployments for user from redis
	if err := d.rdb.Del(ctx, userId+"-deployments").Err(); err != nil {
		logging.LogError(ctx, "error occurred deleting the deployments record from redis.",
			"subscription_id", subscriptionId,
			"workspace", workspace,
			"error", err,
		)

		return err
	}

	return nil
}
