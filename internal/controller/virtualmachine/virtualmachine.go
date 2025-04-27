package virtualmachine

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	xpv1 "github.com/crossplane/crossplane-runtime/apis/common/v1"
	"github.com/crossplane/crossplane-runtime/pkg/connection"
	"github.com/crossplane/crossplane-runtime/pkg/controller"
	"github.com/crossplane/crossplane-runtime/pkg/event"
	"github.com/crossplane/crossplane-runtime/pkg/logging"
	"github.com/crossplane/crossplane-runtime/pkg/ratelimiter"
	"github.com/crossplane/crossplane-runtime/pkg/reconciler/managed"
	"github.com/crossplane/crossplane-runtime/pkg/resource"
	"github.com/crossplane/provider-ucan/pkg/ucansdk"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/crossplane/provider-ucan/apis/osgalaxy/v1alpha1"
	apisv1alpha1 "github.com/crossplane/provider-ucan/apis/v1alpha1"
	"github.com/crossplane/provider-ucan/internal/features"
	"github.com/crossplane/provider-ucan/pkg/httpclient"
)

const (
	errNotVirtualMachine = "managed resource is not a VirtualMachine custom resource"
	errTrackPCUsage      = "cannot track ProviderConfig usage"
	errGetPC             = "cannot get ProviderConfig"
	errGetCreds          = "cannot get credentials"
	errNewClient         = "cannot create new Service"

	vmUUIDAnnotationKey = "ucan.io/virtualmachine-uuid"
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

// Setup adds a controller that reconciles VirtualMachine managed resources.
func Setup(mgr ctrl.Manager, o controller.Options) error {
	name := managed.ControllerName(v1alpha1.VirtualMachineGroupKind)

	cps := []managed.ConnectionPublisher{managed.NewAPISecretPublisher(mgr.GetClient(), mgr.GetScheme())}
	if o.Features.Enabled(features.EnableAlphaExternalSecretStores) {
		cps = append(cps, connection.NewDetailsManager(mgr.GetClient(), apisv1alpha1.StoreConfigGroupVersionKind))
	}

	r := managed.NewReconciler(mgr,
		resource.ManagedKind(v1alpha1.VirtualMachineGroupVersionKind),
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
		For(&v1alpha1.VirtualMachine{}).
		Complete(ratelimiter.NewReconciler(name, r, o.GlobalRateLimiter))
}

// A connector is expected to produce an ExternalClient when its Connect method
// is called.
type connector struct {
	kube         client.Client
	usage        resource.Tracker
	newServiceFn func(creds []byte) (*UcanClient, error)
}

func (c *connector) Connect(ctx context.Context, mg resource.Managed) (managed.ExternalClient, error) {
	cr, ok := mg.(*v1alpha1.VirtualMachine)
	if !ok {
		return nil, errors.New(errNotVirtualMachine)
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
	log := logging.NewLogrLogger(zl.WithName("provider-vm"))
	return &external{service: svc, logger: log}, nil
}

type external struct {
	service *UcanClient
	logger  logging.Logger
}

func (c *external) Observe(ctx context.Context, mg resource.Managed) (managed.ExternalObservation, error) {
	cr, ok := mg.(*v1alpha1.VirtualMachine)
	if !ok {
		return managed.ExternalObservation{}, errors.New(errNotVirtualMachine)
	}

	uuid, ok := cr.GetAnnotations()[vmUUIDAnnotationKey]
	if !ok {
		c.logger.Info("Resource Status", "msg", "uuid not found")
		return managed.ExternalObservation{
			ResourceExists: false,
		}, nil
	}

	vm, code, err := ucansdk.GetVm(c.service.HttpClient, uuid)
	if err != nil {
		c.logger.Info("get Resource err", "msg", err)
		return managed.ExternalObservation{}, errors.Wrap(err, "cannot get virtual machine")
	}
	// if code >= http.StatusBadRequest {
	//	c.logger.Info("get Resource err", "code", code, "body", string(vm))
	//	return managed.ExternalObservation{}, errors.New("cannot get virtual machine")
	// }

	c.logger.Info("get Resource", "code", code, "parmar", string(vm))
	resourceExists := code != http.StatusInternalServerError
	if resourceExists {
		//将状态置为可用
		cr.SetConditions(xpv1.Available())
	}

	return managed.ExternalObservation{
		ResourceExists:    resourceExists,
		ResourceUpToDate:  true,
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Create(ctx context.Context, mg resource.Managed) (managed.ExternalCreation, error) {
	cr, ok := mg.(*v1alpha1.VirtualMachine)
	if !ok {
		return managed.ExternalCreation{}, errors.New(errNotVirtualMachine)
	}

	req := ucansdk.CreateServerReq{
		Name:      cr.Spec.ForProvider.Name,
		ProjectID: cr.Spec.ForProvider.ProjectId,
		ImageRef:  cr.Spec.ForProvider.ImageRef,
		FlavorRef: cr.Spec.ForProvider.FlavorRef,
	}
	reqData, err := json.Marshal(req)
	if err != nil {
		c.logger.Info("Marshal err create Resource", "msg", err)
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create virtual machine")
	}
	c.logger.Info("create Resource", "req", string(reqData))
	vm, code, err := ucansdk.CreateVm(c.service.HttpClient, reqData)
	if err != nil {
		c.logger.Info("create Resource err", "msg", err)
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot create virtual machine")
	}
	if code >= http.StatusBadRequest {
		c.logger.Info("create Resource err", "code", code, "body", string(vm))
		return managed.ExternalCreation{}, errors.New("cannot create virtual machine")
	}

	var response ucansdk.ServerResp
	if err = json.Unmarshal(vm, &response); err != nil {
		c.logger.Info("unmarshal err create Resource", "msg", err)
		return managed.ExternalCreation{}, errors.Wrap(err, "cannot unmarshal virtual machine")
	}
	c.logger.Info("unmarshal Resource", "id", response.Server.ID, "name", response.Server.Name)

	//将虚拟机的uuid存入标签中
	mg.SetAnnotations(map[string]string{
		vmUUIDAnnotationKey: response.Server.ID,
	})
	return managed.ExternalCreation{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{
			"observableField": []byte(response.Server.ID),
		},
	}, nil
}

func (c *external) Update(ctx context.Context, mg resource.Managed) (managed.ExternalUpdate, error) {
	cr, ok := mg.(*v1alpha1.VirtualMachine)
	if !ok {
		return managed.ExternalUpdate{}, errors.New(errNotVirtualMachine)
	}

	fmt.Printf("Updating: name:%s\tfinalizers: %+v\n", cr.Name, cr.Finalizers)

	return managed.ExternalUpdate{
		// Optionally return any details that may be required to connect to the
		// external resource. These will be stored as the connection secret.
		ConnectionDetails: managed.ConnectionDetails{},
	}, nil
}

func (c *external) Delete(ctx context.Context, mg resource.Managed) (managed.ExternalDelete, error) {
	cr, ok := mg.(*v1alpha1.VirtualMachine)
	if !ok {
		return managed.ExternalDelete{}, errors.New(errNotVirtualMachine)
	}

	fmt.Printf("Deleting: name:%s\tfinalizers: %+v\n", cr.Name, cr.Finalizers)

	return managed.ExternalDelete{}, nil
}

func (c *external) Disconnect(ctx context.Context) error {
	return nil
}

// func (c *external) isUpToDate(cr *v1alpha1.VirtualMachine, externalResource map[string]any) (bool, string) {
//	return true, ""
// }
