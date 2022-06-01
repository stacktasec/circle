package minio

import "github.com/minio/minio-go/v7"

func InitMinio() {
	minio.New("", &minio.Options{})
}
