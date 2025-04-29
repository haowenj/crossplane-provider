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

package floatingip

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
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
	errNotFloatingip = "managed resource is not a Floatingip custom resource"
	errTrackPCUsage  = "cannot track ProviderConfig usage"
	errGetPC         = "cannot get ProviderConfig"
	errGetCreds      = "cannot get credentials"
	errNewClient     = "cannot create new Service"

	eipUUIDAnnotationKey = "ucan.io/eip-uuid"
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

// Setup adds a controller that reconciles Floatingip managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.FloatingipGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.FloatingipGroupVersionKind),
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
		For(&v1alpha1.Floatingip{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(creds []byte) (*UcanClient, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.Floatingip)
	if !ok {
		return nil, errors.New(errNotFloatingip)
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
	log := logging.NewLogrLogger(zl.WithName("provider-eip"))
	return &external{service: svc, logger: log}, nil
}

type external struct {
	service *UcanClient
	logger  logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.Floatingip)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotFloatingip)
	}

	uuid, ok := cr.GetAnnotations()[eipUUIDAnnotationKey]
	if !ok {
		c.logger.Info("eip Resource Status", "msg", "uuid not found")
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	c.service.HttpClient.SetHeader("X-UCAN-NS", cr.Spec.ForProvider.ProjectId)
	eip, code, err := ucansdk.GetEip(c.service.HttpClient, uuid)
	if err != nil {
		c.logger.Info("eip Resource err", "msg", err)
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get volume")
	}
	if code >= http.StatusBadRequest && code != http.StatusNotFound {
		c.logger.Info("get Resource err", "code", code, "body", string(eip))
		return managed.ExternalObservation{}, errors.New("cannot get eip")
	}

	var response ucansdk.EipGetResponse
	if err = json.Unmarshal(eip, &response); err != nil {
		c.logger.Info("unmarshal err get Resource", "msg", err)
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot unmarshal eip")
	}
	c.logger.Info("unmarshal Resource", "id", response.FloatingIps.ID, "name", response.FloatingIps.Name, "status", response.FloatingIps.Status)
	resourceExists := code != http.StatusNotFound
	if resourceExists && response.FloatingIps.Status == "running" {
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
	cr, ok := mg.(*v1alpha1.Floatingip)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotFloatingip)
	}

	req := ucansdk.CreateEipReq{
		FloatingIp: ucansdk.CreateEipReqParam{
			Name:            cr.Spec.ForProvider.Name,
			ProjectID:       cr.Spec.ForProvider.ProjectId,
			CellId:          cr.Spec.ForProvider.CellId,
			FloatingNetwork: cr.Spec.ForProvider.FloatingNetworkId,
			Isp:             cr.Spec.ForProvider.Isp,
			Bandwidth:       int(cr.Spec.ForProvider.Bandwidth),
			Description:     cr.Spec.ForProvider.Description,
			RouteId:         cr.Spec.ForProvider.RouteId,
		},
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		c.logger.Info("Marshal err create Resource", "msg", err)
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create eip")
	}
	c.service.HttpClient.SetHeader("X-UCAN-NS", cr.Spec.ForProvider.ProjectId)
	eip, code, err := ucansdk.CreateEip(c.service.HttpClient, reqData)
	if err != nil {
		c.logger.Info("create Resource err", "msg", err)
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create eip")
	}
	if code >= http.StatusBadRequest {
		c.logger.Info("create Resource err", "code", code, "body", string(eip))
		return managed.ExternalCreation{}, errors.New("cannot create eip")
	}
	var response ucansdk.EipGetResponse
	if err = json.Unmarshal(eip, &response); err != nil {
		c.logger.Info("unmarshal err create Resource", "msg", err)
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot unmarshal eip")
	}
	c.logger.Info("unmarshal Resource", "id", response.FloatingIps.ID, "name", response.FloatingIps.Name)

	mg.SetAnnotations(map[string]string{
		eipUUIDAnnotationKey: response.FloatingIps.ID,
	})

	return managed.ExternalCreation{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.Floatingip)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotFloatingip)
	}

	fmt.Printf("Updating: %+v\n", cr.Spec.ForProvider)

	return managed.ExternalUpdate{
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.Floatingip)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotFloatingip)
	}

	uuid, ok := cr.GetAnnotations()[eipUUIDAnnotationKey]
	if !ok {
		c.logger.Info("eip Resource Status", "name", cr.Name, "msg", "uuid not found")
		return managed.ExternalDelete{}, errors.New("uuid not found")
	}
	c.service.HttpClient.SetHeader("X-UCAN-NS", cr.Spec.ForProvider.ProjectId)
	body, code, err := ucansdk.DelEip(c.service.HttpClient, uuid)
	if err != nil {
		c.logger.Info("delete Resource err", "name", cr.Name, "msg", err)
		return managed.ExternalDelete{}, errors.Wrap(err, "cannot delete eip")
	}
	if code != http.StatusNoContent {
		c.logger.Info("delete Resource err", "name", cr.Name, "code", code, "body", string(body))
		return managed.ExternalDelete{}, errors.New("cannot delete eip")
	}

	c.logger.Info("delete Resource", "name", cr.Name, "code", code)
	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}
