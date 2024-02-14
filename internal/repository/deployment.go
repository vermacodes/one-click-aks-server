package repository

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"one-click-aks-server/internal/auth"
	"one-click-aks-server/internal/config"
	"one-click-aks-server/internal/entity"

	"github.com/redis/go-redis/v9"
	"golang.org/x/exp/slog"
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

func (d *deploymentRepository) GetDeployments() ([]entity.Deployment, error) {
	return nil, nil
}

func (d *deploymentRepository) GetMyDeployments(userId string, subscriptionId string) ([]entity.Deployment, error) {
	deployments := []entity.Deployment{}

	// check if user deployments already exist in redis
	deploymentsString, err := d.rdb.Get(ctx, userId+"-deployments").Result()
	if err != nil {
		slog.Debug("error getting deployments from redis continue to get from table storage ",
			slog.String("userId", userId),
			slog.String("error", err.Error()),
		)
	}
	if deploymentsString != "" {
		if err := json.Unmarshal([]byte(deploymentsString), &deployments); err == nil {
			slog.Debug("deployments found in redis ",
				slog.String("userId", userId),
			)

			return deployments, nil
		}
		slog.Debug("error unmarshal deployment found in redis continue to get from table storage ",
			slog.String("userId", userId),
			slog.String("error", err.Error()),
		)
	}

	url := d.appConfig.ActlabsHubURL + "deployments"
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		slog.Error("error getting deployments ", err)
		return nil, err
	}

	armAccessToken, err := d.auth.GetARMAccessToken()
	if err != nil {
		slog.Error("error getting arm access token ", err)
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
		slog.Error("error getting deployments ", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("error getting deployments ", err)
		return nil, err
	}

	if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
		slog.Error("error unmarshal deployments ", err)
		return nil, err
	}

	// save deployments to redis
	marshalledDeployments, err := json.Marshal(deployments)
	if err != nil {
		slog.Debug("error occurred marshalling the deployments record.",
			slog.String("userId", userId),
			slog.String("error", err.Error()),
		)

		return deployments, err
	}

	err = d.rdb.Set(ctx, userId+"-deployments", marshalledDeployments, 0).Err()
	if err != nil {
		slog.Debug("error occurred saving the deployments record to redis.",
			slog.String("userId", userId),
			slog.String("error", err.Error()),
		)
	}

	return deployments, nil
}

func (d *deploymentRepository) GetDeployment(userId string, workspace string, subscriptionId string) (entity.Deployment, error) {
	deployment := entity.Deployment{}

	// check if deployment already exist in redis
	deploymentString, err := d.rdb.Get(ctx, userId+"-"+subscriptionId+"-"+workspace).Result()
	if err != nil {
		slog.Debug("error getting deployment from redis continue to get from table storage ",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)
	}
	if deploymentString != "" {
		if err := json.Unmarshal([]byte(deploymentString), &deployment); err == nil {
			slog.Debug("deployment found in redis ",
				slog.String("userId", userId),
				slog.String("subscriptionId", subscriptionId),
				slog.String("workspace", workspace),
			)

			return deployment, nil
		}
		slog.Debug("error unmarshal deployment found in redis continue to get from table storage ",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)
	}

	deployments, err := d.GetMyDeployments(userId, subscriptionId)
	if err != nil {
		slog.Error("error getting deployments ", err)
		return entity.Deployment{}, err
	}

	for _, deployment := range deployments {
		if deployment.DeploymentWorkspace == workspace &&
			deployment.DeploymentUserId == userId &&
			deployment.DeploymentSubscriptionId == subscriptionId {

			// save deployment to redis
			marshalledDeployment, err := json.Marshal(deployment)
			if err != nil {
				slog.Debug("error occurred marshalling the deployment record.",
					slog.String("userId", userId),
					slog.String("subscriptionId", subscriptionId),
					slog.String("workspace", workspace),
					slog.String("error", err.Error()),
				)

				return deployment, err
			}

			err = d.rdb.Set(ctx, userId+"-"+subscriptionId+"-"+workspace, marshalledDeployment, 0).Err()
			if err != nil {
				slog.Debug("error occurred saving the deployment record to redis.",
					slog.String("userId", userId),
					slog.String("subscriptionId", subscriptionId),
					slog.String("workspace", workspace),
					slog.String("error", err.Error()),
				)
			}

			return deployment, nil
		}
	}

	return entity.Deployment{}, errors.New("deployment not found")
}

func (d *deploymentRepository) UpsertDeployment(deployment entity.Deployment) error {

	slog.Debug("upserting deployment ",
		slog.String("userId", deployment.DeploymentUserId),
		slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
		slog.String("workspace", deployment.DeploymentWorkspace),
		slog.String("Status", string(deployment.DeploymentStatus)),
	)

	url := d.appConfig.ActlabsHubURL + "deployments"
	req, err := http.NewRequest(http.MethodPut, url, nil)
	if err != nil {
		slog.Error("error creating new request ", err)
		return err
	}

	armAccessToken, err := d.auth.GetARMAccessToken()
	if err != nil {
		slog.Error("error getting arm access token ", err)
		return err
	}

	req.Header.Set("Authorization", "Bearer "+armAccessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-ms-client-principal-name", d.appConfig.ArmUserPrincipalName)
	req.Header.Set("ProtectedLabSecret", entity.ProtectedLabSecret)

	marshalledDeployment, err := json.Marshal(deployment)
	if err != nil {
		slog.Error("error occurred marshalling the deployment.", err)
		return err
	}
	req.Body = io.NopCloser(bytes.NewReader(marshalledDeployment))

	client := &http.Client{
		Timeout: time.Second * time.Duration(d.appConfig.HttpRequestTimeoutSeconds),
	}

	resp, err := client.Do(req)
	if err != nil {
		slog.Error("error upserting deployments ", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("error upserting deployments ", err)
		return err
	}

	// save deployment to redis
	if err := d.rdb.Set(ctx, deployment.DeploymentUserId+"-"+deployment.DeploymentSubscriptionId+"-"+deployment.DeploymentWorkspace, marshalledDeployment, 0).Err(); err != nil {
		slog.Debug("error occurred saving the deployment record to redis.",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
		)

		// if not able to add deployment, delete existing deployment from redis if any
		if err := d.rdb.Del(ctx, deployment.DeploymentUserId+"-"+deployment.DeploymentSubscriptionId+"-"+deployment.DeploymentWorkspace).Err(); err != nil {
			slog.Debug("error occurred deleting the deployment record from redis.",
				slog.String("userId", deployment.DeploymentUserId),
				slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
				slog.String("workspace", deployment.DeploymentWorkspace),
				slog.String("error", err.Error()),
			)

			return err
		}
	}

	// delete deployments for user from redis
	if err := d.rdb.Del(ctx, deployment.DeploymentUserId+"-deployments").Err(); err != nil {
		slog.Debug("error occurred deleting the deployments record from redis.",
			slog.String("userId", deployment.DeploymentUserId),
			slog.String("subscriptionId", deployment.DeploymentSubscriptionId),
			slog.String("workspace", deployment.DeploymentWorkspace),
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}

func (d *deploymentRepository) DeleteDeployment(userId string, workspace string, subscriptionId string) error {
	url := d.appConfig.ActlabsHubURL + "deployments/" + subscriptionId + "/" + workspace

	req, err := http.NewRequest(http.MethodDelete, url, nil)
	if err != nil {
		slog.Error("error creating new request ", err)
		return err
	}

	armAccessToken, err := d.auth.GetARMAccessToken()
	if err != nil {
		slog.Error("error getting arm access token ", err)
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
		slog.Error("error deleting deployments ", err)
		return err
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		slog.Error("error deleting deployments ", err)
		return err
	}

	// delete deployment from redis
	if err := d.rdb.Del(ctx, userId+"-"+subscriptionId+"-"+workspace).Err(); err != nil {
		slog.Debug("error occurred deleting the deployment record from redis.",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)

		return err
	}

	// delete deployments for user from redis
	if err := d.rdb.Del(ctx, userId+"-deployments").Err(); err != nil {
		slog.Debug("error occurred deleting the deployments record from redis.",
			slog.String("userId", userId),
			slog.String("subscriptionId", subscriptionId),
			slog.String("workspace", workspace),
			slog.String("error", err.Error()),
		)

		return err
	}

	return nil
}
