/*
Copyright 2022 The Crossplane Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package volume

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"

	"github.com/crossplane/provider-ucan/apis/osgalaxy/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-ucan/apis/v1alpha1"
	"github.com/crossplane/provider-ucan/internal/features"
	"github.com/crossplane/provider-ucan/pkg/httpclient"
	"github.com/crossplane/provider-ucan/pkg/ucansdk"
)

const (
	errNotVolume    = "managed resource is not a Volume custom resource"
	errTrackPCUsage = "cannot track ProviderConfig usage"
	errGetPC        = "cannot get ProviderConfig"
	errGetCreds     = "cannot get credentials"
	errNewClient    = "cannot create new Service"

	volumeUUIDAnnotationKey = "ucan.io/volume-uuid"
)

type UcanClient struct {
	HttpClient *httpclient.HttpClient
}

var (
	newUcanClient = func(credentials []byte) (*UcanClient, error) {
		var signCertificate httpclient.SignCertificate
		if err := json.Unmarshal(credentials, &signCertificate); err != nil {
			fmt.Println("*************cannot get credentials*************")
			return nil, err
		}
		cli := httpclient.NewHttpClient(signCertificate)
		cli.SetHeader("Content-Type", "application/json")
		return &UcanClient{HttpClient: cli}, nil
	}
)

// Setup adds a controller that reconciles Volume managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.VolumeGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.VolumeGroupVersionKind),
		managed.WithExternalConnecter(&connector{
			kube:         mgr.GetClient(),
			usage:        resource.NewProviderConfigUsageTracker(mgr.GetClient(), &apisv1alpha1.ProviderConfigUsage{}),
			newServiceFn: newUcanClient}),
		managed.WithLogger(o.Logger.WithValues("controller", name)),
		managed.WithPollInterval(o.PollInterval),
		managed.WithRecorder(event.NewAPIRecorder(mgr.GetEventRecorderFor(name))),
		managed.WithConnectionPublishers(cps...))

	return ctrl.NewControllerManagedBy(mgr).
		Named(name).
		WithOptions(o.ForControllerRuntime()).
		WithEventFilter(resource.DesiredStateChanged()).
		For(&v1alpha1.Volume{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(creds []byte) (*UcanClient, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Volume)
	if !ok {
		return nil, errors.New(errNotVolume)
	}

	if err := c.usage.Track(ctx, mg); err != nil {
		return nil, errors.Wrap(err, errTrackPCUsage)
	}

	pc := &apisv1alpha1.ProviderConfig{}
	if err := c.kube.Get(ctx, types.NamespacedName{Name: cr.GetProviderConfigReference().Name}, pc); err != nil {
		return nil, errors.Wrap(err, errGetPC)
	}

	cd := pc.Spec.Credentials
	data, err := resource.CommonCredentialExtractor(ctx, cd.Source, c.kube, cd.CommonCredentialSelectors)
	if err != nil {
		return nil, errors.Wrap(err, errGetCreds)
	}

	svc, err := c.newServiceFn(data)
	if err != nil {
		return nil, errors.Wrap(err, errNewClient)
	}

	zl := zap.New(zap.UseDevMode(true))
	log := logging.NewLogrLogger(zl.WithName("provider-volume"))
	return &external{service: svc, logger: log}, nil
}

type external struct {
	service *UcanClient
	logger  logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Volume)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotVolume)
	}

	uuid, ok := cr.GetAnnotations()[volumeUUIDAnnotationKey]
	if !ok {
		c.logger.Info("volume Resource Status", "msg", "uuid not found")
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	volume, code, err := ucansdk.GetVolume(c.service.HttpClient, cr.Spec.ForProvider.ProjectId, uuid)
	if err != nil {
		c.logger.Info("volume Resource err", "msg", err)
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get volume")
	}
	if code >= http.StatusBadRequest && code != http.StatusNotFound {
		c.logger.Info("get Resource err", "code", code, "body", string(volume))
		return managed.ExternalObservation{}, errors.New("cannot get volume")
	}
	resourceExists := code != http.StatusNotFound

	var response ucansdk.VolumeResp
	if err = json.Unmarshal(volume, &response); err != nil {
		c.logger.Info("unmarshal err create Resource", "msg", err)
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot unmarshal volume")
	}
	c.logger.Info("get Resource", "code", code, "name", response.Volume.Name, "status", response.Volume.Status, "uuid", response.Volume.ID)
	if resourceExists && response.Volume.Status == "available" {
		// 将状态置为可用
		cr.SetConditions(xpv1.Available())
	}

	return managed.ExternalObservation{
		ResourceExists:    resourceExists,
		ResourceUpToDate:  true,
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.Volume)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotVolume)
	}

	req := ucansdk.CreateVolumeReq{
		Volume: ucansdk.VolumeSpec{
			Size:        int(cr.Spec.ForProvider.Size),
			Description: &cr.Spec.ForProvider.Description,
			Multiattach: cr.Spec.ForProvider.Multiattach,
			Name:        &cr.Spec.ForProvider.Name,
			VolumeType:  &cr.Spec.ForProvider.VolumeType,
			CellID:      cr.Spec.ForProvider.CellId,
		},
		OSSCHSchedulerHints: ucansdk.VolumeSchedulerHints{
			SameHost: cr.Spec.ForProvider.SchedulerHints[0].SameHost,
		},
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		c.logger.Info("Marshal err create Resource", "msg", err)
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create volume")
	}
	volume, code, err := ucansdk.CreateVolume(c.service.HttpClient, reqData, cr.Spec.ForProvider.ProjectId)
	if err != nil {
		c.logger.Info("create Resource err", "msg", err)
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create volume")
	}
	if code >= http.StatusBadRequest {
		c.logger.Info("create Resource err", "code", code, "body", string(volume))
		return managed.ExternalCreation{}, errors.New("cannot create volume")
	}
	var response ucansdk.VolumeResp
	if err = json.Unmarshal(volume, &response); err != nil {
		c.logger.Info("unmarshal err create Resource", "msg", err)
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot unmarshal volume")
	}
	c.logger.Info("unmarshal Resource", "id", response.Volume.ID, "name", response.Volume.Name)

	mg.SetAnnotations(map[string]string{
		volumeUUIDAnnotationKey: response.Volume.ID,
	})
	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Volume)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotVolume)
	}

	fmt.Printf("Updating: %+v\n", cr.Spec.ForProvider)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Volume)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotVolume)
	}

	uuid, ok := cr.GetAnnotations()[volumeUUIDAnnotationKey]
	if !ok {
		c.logger.Info("volume Resource Status", "name", cr.Name, "msg", "uuid not found")
		return managed.ExternalDelete{}, nil
	}
	body, code, err := ucansdk.DelVolume(c.service.HttpClient, cr.Spec.ForProvider.ProjectId, uuid)
	if err != nil {
		c.logger.Info("delete Resource err", "name", cr.Name, "msg", err)
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete volume")
	}
	if code != http.StatusNoContent {
		c.logger.Info("delete Resource err", "name", cr.Name, "code", code, "body", string(body))
		return managed.ExternalDelete{}, errors.New("cannot delete volume")
	}
	c.logger.Info("delete Resource", "name", cr.Name, "code", code)

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
