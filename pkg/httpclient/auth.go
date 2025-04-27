package httpclient

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
)

type SignCertificate struct {
	AccessKeyID     string `json:"accessKeyId"`
	SecretAccessKey string `json:"secretAccessKey"`
	Service         string `json:"service"`
	Region          string `json:"region"`
}

func Sign(request *http.Request, certificate SignCertificate) error {
	if certificate.AccessKeyID == "" || certificate.SecretAccessKey == "" {
		return fmt.Errorf("invalid certificate")
	}

	cred := aws.Credentials{
		AccessKeyID:     certificate.AccessKeyID,
		SecretAccessKey: certificate.SecretAccessKey,
	}

	var requestBodyBytes []byte
	if request.Body != nil {
		requestBodyBytes, err := io.ReadAll(request.Body)
		if err != nil {
			return fmt.Errorf("read request body failed")
		}
		request.Body = io.NopCloser(bytes.NewReader(requestBodyBytes))
	}
	payloadHash, err := GetPayloadHash(bytes.NewReader(requestBodyBytes))
	if err != nil {
		return fmt.Errorf("get payload hash failed")
	}

	s := v4.NewSigner(func(opts *v4.SignerOptions) {
		opts.DisableURIPathEscaping = true
	})

	return s.SignHTTP(context.Background(), cred, request, payloadHash, certificate.Service, certificate.Region, time.Now())
}

// GetPayloadHash 计算 Payload Hash
func GetPayloadHash(body io.Reader) (string, error) {
	if body == nil || body == http.NoBody {
		return "UNSIGNED-PAYLOAD", nil // 无 Body 时的默认值
	}

	hash := sha256.New()
	_, err := io.Copy(hash, body) // 流式读取 Body
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}
