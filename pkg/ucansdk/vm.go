package ucansdk

import (
	"fmt"
	"time"

	"github.com/crossplane/provider-ucan/pkg/httpclient"
)

type CreateServerReq struct {
	Name               string               `json:"name" binding:"required"`
	ProjectID          string               `json:"project_id" binding:"required"`
	CellID             string               `json:"cell_id"`
	ReservationID      string               `json:"reservation_id"`
	AccessIPv4         string               `json:"accessIPv4"`
	AccessIPv6         string               `json:"accessIPv6"`
	ImageRef           string               `json:"imageRef" binding:"required"`
	FlavorRef          string               `json:"flavorRef" binding:"required"`
	AvailabilityZone   string               `json:"availability_zone"`
	Metadata           map[string]string    `json:"metadata"`
	Personality        []FileInjection      `json:"personality"`
	SecurityGroups     []SecurityGroup      `json:"security_groups"`
	UserData           string               `json:"user_data"`
	BlockDeviceMapping []BlockDeviceMapping `json:"block_device_mapping"`
}

type BlockDeviceMapping struct {
	BootIndex           int    `json:"boot_index,omitempty" binding:"omitempty"`
	DeleteOnTermination bool   `json:"delete_on_termination,omitempty"`
	DeviceName          string `json:"device_name,omitempty"`
	DeviceType          string `json:"device_type,omitempty"`
	DiskBus             string `json:"disk_bus,omitempty"`
	GuestFormat         string `json:"guest_format,omitempty"`
	NoDevice            bool   `json:"no_device,omitempty"`
	SourceType          string `json:"source_type,omitempty" binding:"required_without=NoDevice"`
	DestinationType     string `json:"destination_type,omitempty"`
	UUID                string `json:"uuid,omitempty" binding:"required_if=SourceType image snapshot volume"`
	VolumeSize          int    `json:"volume_size,omitempty"`
	VolumeType          string `json:"volume_type,omitempty"`
	Tag                 string `json:"tag,omitempty"`
}

type FileInjection struct {
	Path     string `json:"path" binding:"required"`
	Contents string `json:"contents" binding:"required"`
}

type SecurityGroup struct {
	Name string `json:"name"`
}

type ServerResp struct {
	Server struct {
		AccessIPv4      string               `json:"accessIPv4"`
		AccessIPv6      string               `json:"accessIPv6"`
		Addresses       map[string][]Address `json:"addresses"`
		Created         time.Time            `json:"created"`
		Description     string               `json:"description"`
		Flavor          FlavorResponse       `json:"flavor"`
		HostID          string               `json:"hostId"`
		ID              string               `json:"id"`
		Image           Image                `json:"image"`
		KeyName         *string              `json:"key_name"`
		Links           []Link               `json:"links"`
		Metadata        map[string]string    `json:"metadata"`
		Name            string               `json:"name"`
		ConfigDrive     string               `json:"config_drive"`
		Locked          bool                 `json:"locked"`
		LockedReason    string               `json:"locked_reason"`
		PinnedAZ        string               `json:"pinned_availability_zone"`
		Progress        int                  `json:"progress"`
		SchedulerHints  SchedulerHints       `json:"scheduler_hints"`
		SecurityGroups  []SecurityGroup      `json:"security_groups"`
		Status          string               `json:"status"`
		Tags            []string             `json:"tags"`
		TenantID        string               `json:"tenant_id"`
		TrustedCerts    *string              `json:"trusted_image_certificates"`
		Updated         time.Time            `json:"updated"`
		UserID          string               `json:"user_id"`
		VolumesAttached []VolumeAttached     `json:"os-extended-volumes:volumes_attached"`
	} `json:"server"`
}

type Address struct {
	Addr    string `json:"addr"`
	MACAddr string `json:"OS-EXT-IPS-MAC:mac_addr"`
	IPType  string `json:"OS-EXT-IPS:type"`
	Version int    `json:"version"`
}

type VolumeAttached struct {
	ID                  string `json:"id"`
	DeleteOnTermination bool   `json:"delete_on_termination"`
}

type Image struct {
	ID         string          `json:"id"`
	Links      []Link          `json:"links"`
	Properties ImageProperties `json:"properties"`
}

type ImageProperties struct {
	Architecture    string `json:"architecture"`
	AutoDiskConfig  string `json:"auto_disk_config"`
	BaseImageRef    string `json:"base_image_ref"`
	ContainerFormat string `json:"container_format"`
	DiskFormat      string `json:"disk_format"`
	KernelID        string `json:"kernel_id"`
	MinDisk         string `json:"min_disk"`
	MinRAM          string `json:"min_ram"`
	RamdiskID       string `json:"ramdisk_id"`
}

type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
}

type SchedulerHints struct {
	SameHost []string `json:"same_host"`
}

type FlavorResponse struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Disk        int               `json:"disk"`
	RAM         int               `json:"ram"`
	VCPUs       int               `json:"vcpus"`
	Swap        int               `json:"swap"`
	RxtxFactor  float64           `json:"rxtx_factor"`
	Description *string           `json:"description"`
	ExtraSpecs  map[string]string `json:"extra_specs"`
	Links       []Link            `json:"links"`

	OSFlavorDisabledDisabled bool `json:"OS-FLV-DISABLED:disabled"`
	OSFlavorExtDataEphemeral int  `json:"OS-FLV-EXT-DATA:ephemeral"`
	OSFlavorAccessIsPublic   bool `json:"os-flavor-access:is_public"`
}

func GetVm(client *httpclient.HttpClient, vmId string) ([]byte, int, error) {
	vmHost := "http://virtualmachine.ucan.ustack.com"
	url := fmt.Sprintf("%s/virtualmachine/v3/servers/%s", vmHost, vmId)
	return client.GET(url, nil)
}

func DelVm(client *httpclient.HttpClient, vmId string) ([]byte, int, error) {
	vmHost := "http://virtualmachine.ucan.ustack.com"
	url := fmt.Sprintf("%s/virtualmachine/v3/servers/%s", vmHost, vmId)
	return client.DELETE(url, nil)
}

func CreateVm(client *httpclient.HttpClient, req []byte) ([]byte, int, error) {
	vmHost := "http://virtualmachine.ucan.ustack.com"
	url := fmt.Sprintf("%s/virtualmachine/v3/servers", vmHost)
	return client.POST(url, req)
}
