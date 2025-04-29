package ucansdk

import (
	"fmt"
	"time"

	"github.com/crossplane/provider-ucan/pkg/httpclient"
)

type CreateEipReqParam struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ProjectID       string `json:"project_id"`
	FloatingNetwork string `json:"floating_network_id"`
	CellId          string `json:"cell_id"`
	// QosPolicyId     string  `json:"qos_policy_id"`
	RouteId        string  `json:"route_id"`
	Bandwidth      int     `json:"bandwidth" binding:"required"`
	Isp            string  `json:"isp" binding:"required"`
	Description    string  `json:"description"`
	FloatingIP     *string `json:"floating_ip_address"`
	FixedIPAddress string  `json:"fixed_ip_address"`
	UserID         string  `json:"user_id"`
	ReservationID  string  `json:"reservation_id"`
}

type CreateEipReq struct {
	FloatingIp CreateEipReqParam `json:"floatingip"`
}

type EipGetResponse struct {
	FloatingIps EipResp `json:"floatingips"`
}

type EipResp struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	ProjectID       string    `json:"project_id"`
	FloatingNetwork string    `json:"floating_network_id"`
	CellId          string    `json:"cell_id"`
	Status          string    `json:"status"`
	QosPolicyId     string    `json:"qos_policy_id"`
	RouteId         string    `json:"route_id"`
	Bandwidth       int       `json:"bandwidth"`
	Isp             string    `json:"isp"`
	Description     string    `json:"description"`
	FloatingIP      *string   `json:"floating_ip_address"`
	FixedIPAddress  string    `json:"fixed_ip_address"`
	UserID          string    `json:"user_id"`
	Created         time.Time `json:"created_at"`
	Updated         time.Time `json:"updated_at"`
}

var eipHost = "http://zed-network-apiserver.ucan-system.svc.cluster.local:8088"

// var eipHost = "http://volume.ucan.ustack.com"

func GetEip(client *httpclient.HttpClient, eipId string) ([]byte, int, error) {
	url := fmt.Sprintf("%s/v3/floatingips/%s", eipHost, eipId)
	// url := fmt.Sprintf("%s/network/v3/floatingips/%s", eipHost, eipId)
	return client.GET(url, nil)
}

func DelEip(client *httpclient.HttpClient, eipId string) ([]byte, int, error) {
	url := fmt.Sprintf("%s/v3/floatingips/%s", eipHost, eipId)
	// url := fmt.Sprintf("%s/network/v3/floatingips/%s", eipHost, eipId)
	return client.DELETE(url, nil)
}

func CreateEip(client *httpclient.HttpClient, req []byte) ([]byte, int, error) {
	url := fmt.Sprintf("%s/v3/floatingips", eipHost)
	// url := fmt.Sprintf("%s/network/v3/floatingips", eipHost)
	return client.POST(url, req)
}
