package docker

import (
	"context"
	"fmt"
)

func ReadFile(ctx context.Context, serverID, path string) (string, error) {
	out, err := ReadFileBytes(ctx, serverID, path)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func ReadFileBytes(ctx context.Context, serverID, path string) ([]byte, error) {
	target, err := resolveServerPath(path)
	if err != nil {
		return nil, err
	}
	if isServerRoot(target) {
		return []byte{}, nil
	}
	return readFileFromHost(serverID, target)
}

func WriteFile(ctx context.Context, serverID, path, content string) error {
	return WriteFileBytes(ctx, serverID, path, []byte(content))
}

func WriteFileBytes(ctx context.Context, serverID, path string, content []byte) error {
	target, err := resolveServerPath(path)
	if err != nil {
		return err
	}
	if isServerRoot(target) {
		return fmt.Errorf("нельзя записать в корень данных сервера")
	}
	return writeFileToHost(ctx, serverID, target, content)
}
