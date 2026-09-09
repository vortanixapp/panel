package handlers

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/jlaffaye/ftp"
)

func testS3Storage(in map[string]any) error {
	key := strings.TrimSpace(fmt.Sprint(in["files_storage_s3_key"]))
	secret := strings.TrimSpace(fmt.Sprint(in["files_storage_s3_secret"]))
	region := strings.TrimSpace(fmt.Sprint(in["files_storage_s3_region"]))
	bucket := strings.TrimSpace(fmt.Sprint(in["files_storage_s3_bucket"]))
	if bucket == "" {
		return fmt.Errorf("укажите S3 bucket")
	}
	if key == "" || secret == "" {
		return fmt.Errorf("укажите S3 key и secret")
	}
	if region == "" {
		region = "us-east-1"
	}

	endpoint := strings.TrimSpace(fmt.Sprint(in["files_storage_s3_endpoint"]))
	usePathStyle := formTruthy(fmt.Sprint(in["files_storage_s3_use_path_style_endpoint"]))

	cfg := aws.Config{
		Region:      region,
		Credentials: credentials.NewStaticCredentialsProvider(key, secret, ""),
	}
	if endpoint != "" {
		if !strings.HasPrefix(endpoint, "http") {
			endpoint = "https://" + endpoint
		}
		cfg.BaseEndpoint = aws.String(endpoint)
	}

	client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.UsePathStyle = usePathStyle
	})

	objectKey := "support/test-connection/vtx-test-" + time.Now().Format("20060102150405") + "-" + randomHex(6) + ".txt"
	body := []byte("vortanix storage test " + time.Now().Format(time.RFC3339))
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err := client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
		Body:   bytes.NewReader(body),
	})
	if err != nil {
		return err
	}

	_, err = client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
	if err != nil {
		return fmt.Errorf("файл не найден после записи: %w", err)
	}

	_, err = client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(objectKey),
	})
	return err
}

func testFTPStorage(in map[string]any, password string) error {
	host := strings.TrimSpace(fmt.Sprint(in["files_storage_ftp_host"]))
	if host == "" {
		return fmt.Errorf("укажите FTP host")
	}
	port := strings.TrimSpace(fmt.Sprint(in["files_storage_ftp_port"]))
	if port == "" {
		port = "21"
	}
	user := strings.TrimSpace(fmt.Sprint(in["files_storage_ftp_username"]))
	if user == "" {
		return fmt.Errorf("укажите FTP username")
	}
	root := strings.TrimSpace(fmt.Sprint(in["files_storage_ftp_root"]))
	if root == "" {
		root = "/"
	}
	useSSL := formTruthy(fmt.Sprint(in["files_storage_ftp_ssl"]))

	timeout := 30 * time.Second
	if t := fmt.Sprint(in["files_storage_ftp_timeout"]); t != "" {
		if sec, err := parsePositiveInt(t); err == nil {
			timeout = time.Duration(sec) * time.Second
		}
	}

	addr := net.JoinHostPort(host, port)
	var conn *ftp.ServerConn
	var err error
	dialOpts := []ftp.DialOption{ftp.DialWithTimeout(timeout)}
	if useSSL {
		dialOpts = append(dialOpts, ftp.DialWithTLS(&tls.Config{InsecureSkipVerify: true}))
	}
	conn, err = ftp.Dial(addr, dialOpts...)
	if err != nil {
		return err
	}
	defer conn.Quit()

	if err := conn.Login(user, password); err != nil {
		return err
	}
	if err := conn.ChangeDir(root); err != nil {
		return fmt.Errorf("не удалось перейти в root %q: %w", root, err)
	}

	testName := "vtx-test-" + randomHex(6) + ".txt"
	content := "vortanix storage test " + time.Now().Format(time.RFC3339)
	if err := conn.Stor(testName, strings.NewReader(content)); err != nil {
		return fmt.Errorf("не удалось загрузить тестовый файл: %w", err)
	}

	entries, err := conn.List(".")
	if err != nil {
		return fmt.Errorf("не удалось получить список файлов: %w", err)
	}
	found := false
	for _, e := range entries {
		if e.Name == testName {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("файл не найден после записи. Проверьте права доступа")
	}

	if err := conn.Delete(testName); err != nil {
		return fmt.Errorf("не удалось удалить тестовый файл: %w", err)
	}
	return nil
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := io.ReadFull(rand.Reader, b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func parsePositiveInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	if err != nil || n <= 0 {
		return 0, fmt.Errorf("invalid")
	}
	return n, nil
}
