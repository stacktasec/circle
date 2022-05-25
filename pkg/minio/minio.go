package minio

import (
	"fmt"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func InitMinio() {
	endpoint := ""
	accessKeyID := ""
	secretAccessKey := ""

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:     credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Transport: &http3.RoundTripper{},
		Secure:    true,
	})

	fmt.Println(minioClient, err)
}
