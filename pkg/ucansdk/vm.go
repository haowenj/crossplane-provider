package ucansdk

import (
	"fmt"

	"github.com/crossplane/provider-ucan/config"
	"github.com/crossplane/provider-ucan/pkg/httpclient"
)

func GetVm(client *httpclient.HttpClient, vmId string) ([]byte, int, error) {
	url := fmt.Sprintf("%s/api/v2/virtualmachines/%s", config.Cfg.GetString("apiHost"), vmId)
	return client.GET(url, nil)
}
