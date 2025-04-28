package ucansdk

import (
	"fmt"
	"time"

	"github.com/crossplane/provider-ucan/pkg/httpclient"
)

type VolumeSpec struct {
	Size               int            `json:"size" binding:"required"`
	AvailabilityZone   *string        `json:"availability_zone"`
	SourceVolumeID     *string        `json:"source_volume_id"`
	Description        *string        `json:"description"`
	Multiattach        bool           `json:"multiattach"`
	SnapshotID         *string        `json:"snapshot_id"`
	BackupID           *string        `json:"backup_id"`
	Name               *string        `json:"name"`
	ImageRef           *string        `json:"imageRef"`
	VolumeType         *string        `json:"volume_type" binding:"required"`
	Metadata           map[string]any `json:"metadata"`
	ConsistencyGroupID *string        `json:"consistency_group_id"`
	ReservationID      string         `json:"reservation_id"`
	CellID             string         `json:"cell_id"`
}

type VolumeSchedulerHints struct {
	SameHost []string `json:"same_host"`
}

type CreateVolumeReq struct {
	Volume              VolumeSpec           `json:"volume"`
	OSSCHSchedulerHints VolumeSchedulerHints `json:"OS-SCH-HNT:scheduler_hints"`
}

type VolumeResp struct {
	Volume struct {
		ID               string         `json:"id"`
		Size             int            `json:"size"`
		Status           string         `json:"status"`
		AvailabilityZone string         `json:"availability_zone"`
		CreatedAt        time.Time      `json:"created_at"`
		UpdatedAt        *time.Time     `json:"updated_at,omitempty"`
		InternalID       string         `json:"internal_id"`
		Name             *string        `json:"name"`
		Description      *string        `json:"description"`
		VolumeType       string         `json:"volume_type"`
		Bootable         bool           `json:"bootable"`
		Encrypted        bool           `json:"encrypted"`
		Multiattach      bool           `json:"multiattach"`
		SourceVolid      *string        `json:"source_volid"`
		SnapshotID       *string        `json:"snapshot_id"`
		Metadata         map[string]any `json:"metadata"`
		Links            []Link         `json:"links"`

		ConsistencyGroupID *string `json:"consistency_group_id,omitempty"`
		MigrationStatus    *string `json:"migration_status,omitempty"`
		ReplicationStatus  *string `json:"replication_status,omitempty"`
		UserID             string  `json:"user_id"`
		ProjectID          string  `json:"os-vol-tenant-attr:tenant_id"`
		Host               *string `json:"os-vol-host-attr:host,omitempty"`
		MigrationNameID    *string `json:"os-vol-mig-status-attr:name_id,omitempty"`
		ProviderID         *string `json:"provider_id,omitempty"`
		GroupID            *string `json:"group_id,omitempty"`
		ServiceUUID        *string `json:"service_uuid,omitempty"`
		SharedTargets      bool    `json:"shared_targets"`
		ClusterName        *string `json:"cluster_name,omitempty"`
	} `json:"volume"`
}

func GetVolume(client *httpclient.HttpClient, projectId, volumeId string) ([]byte, int, error) {
	volumeHost := "http://volume.ucan.ustack.com"
	url := fmt.Sprintf("%s/volume/v3/%s/volumes/%s", volumeHost, projectId, volumeId)
	return client.GET(url, nil)
}

func DelVolume(client *httpclient.HttpClient, projectId, volumeId string) ([]byte, int, error) {
	volumeHost := "http://volume.ucan.ustack.com"
	url := fmt.Sprintf("%s/volume/v3/%s/volumes/%s", volumeHost, projectId, volumeId)
	return client.DELETE(url, nil)
}

func CreateVolume(client *httpclient.HttpClient, req []byte, projectId string) ([]byte, int, error) {
	volumeHost := "http://volume.ucan.ustack.com"
	url := fmt.Sprintf("%s/volume/v3/%s/volumes", volumeHost, projectId)
	return client.POST(url, req)
}
