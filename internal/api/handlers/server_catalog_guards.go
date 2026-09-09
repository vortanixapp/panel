package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type serverOperableState struct {
	Expired      bool
	Provisioning string
	GameSlug     string
}

func (h *Handler) loadServerOperableState(ctx context.Context, serverID string) (serverOperableState, error) {
	var st serverOperableState
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT (s.expires_at IS NOT NULL AND s.expires_at < now()),
		       COALESCE(s.provisioning_status, ''), COALESCE(s.game_id, '')
		FROM core.servers s
		WHERE s.id = $1::uuid
	`, serverID).Scan(&st.Expired, &st.Provisioning, &st.GameSlug)
	return st, err
}

func (st serverOperableState) ensureAcceptsChanges() error {
	if st.Expired {
		return fmt.Errorf("срок аренды сервера истёк")
	}
	switch st.Provisioning {
	case "provisioning", "pending":
		return fmt.Errorf("сервер ещё устанавливается — дождитесь окончания")
	case "deprovisioning":
		return fmt.Errorf("сервер удаляется")
	}
	return nil
}

type pluginInstallSpec struct {
	Active          bool
	InstallPath     string
	SupportedGames  []string
	HasArchive      bool
	HasFileActions  bool
	HasUninstallOps bool
}

func (h *Handler) loadPluginInstallSpec(ctx context.Context, pluginID string) (pluginInstallSpec, error) {
	var spec pluginInstallSpec
	var fileActions, uninstallActions []byte
	var archivePath string
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT active, COALESCE(install_path, ''), supported_games,
		       COALESCE(archive_path, ''), COALESCE(file_actions, '[]'::jsonb),
		       COALESCE(uninstall_actions, '[]'::jsonb)
		FROM core.plugins WHERE id = $1::uuid
	`, pluginID).Scan(&spec.Active, &spec.InstallPath, &spec.SupportedGames,
		&archivePath, &fileActions, &uninstallActions)
	if err != nil {
		return spec, err
	}
	spec.HasArchive = archivePath != ""
	spec.HasFileActions = jsonArrayLen(fileActions) > 0
	spec.HasUninstallOps = jsonArrayLen(uninstallActions) > 0
	return spec, nil
}

func jsonArrayLen(raw []byte) int {
	if len(raw) == 0 {
		return 0
	}
	var arr []any
	if json.Unmarshal(raw, &arr) != nil {
		return 0
	}
	return len(arr)
}

func (spec pluginInstallSpec) supportsGame(gameSlug string) bool {
	if len(spec.SupportedGames) == 0 {
		return true
	}
	for _, code := range spec.SupportedGames {
		if code == "*" {
			return true
		}
		if strings.EqualFold(code, gameSlug) {
			return true
		}
	}
	return false
}

func (spec pluginInstallSpec) ensureInstallable(gameSlug string) error {
	if !spec.Active {
		return fmt.Errorf("плагин отключён администратором")
	}
	if strings.Contains(spec.InstallPath, "..") {
		return fmt.Errorf("некорректный путь установки у плагина")
	}
	if !spec.supportsGame(gameSlug) {
		return fmt.Errorf("плагин не поддерживает эту игру")
	}
	if !spec.HasArchive && !spec.HasFileActions {
		return fmt.Errorf("у плагина нет ни архива, ни действий с файлами")
	}
	return nil
}

func (spec pluginInstallSpec) ensureUninstallable() error {
	if strings.Contains(spec.InstallPath, "..") {
		return fmt.Errorf("некорректный путь установки у плагина")
	}
	if spec.InstallPath == "" && !spec.HasUninstallOps {
		return fmt.Errorf("нельзя удалить плагин из корня сервера: у него не задано, что убирать")
	}
	return nil
}

type mapInstallSpec struct {
	Active     bool
	HasArchive bool
	HasFiles   bool
	GameSlug   string
}

func (h *Handler) loadMapInstallSpec(ctx context.Context, mapID string) (mapInstallSpec, error) {
	var spec mapInstallSpec
	var archivePath string
	var fileList []byte
	err := h.dbOf(ctx).QueryRow(ctx, `
		SELECT m.active, COALESCE(m.archive_path, ''), COALESCE(m.file_list, '[]'::jsonb),
		       COALESCE(g.slug, '')
		FROM core.maps m
		LEFT JOIN core.games g ON g.id = m.game_id
		WHERE m.id = $1::uuid
	`, mapID).Scan(&spec.Active, &archivePath, &fileList, &spec.GameSlug)
	if err != nil {
		return spec, err
	}
	spec.HasArchive = archivePath != ""
	spec.HasFiles = jsonArrayLen(fileList) > 0
	return spec, nil
}

func (spec mapInstallSpec) ensureInstallable(serverGame string) error {
	if !spec.Active {
		return fmt.Errorf("карта отключена администратором")
	}
	if spec.GameSlug != "" && serverGame != "" && !strings.EqualFold(spec.GameSlug, serverGame) {
		return fmt.Errorf("карта не для этой игры")
	}
	if !spec.HasArchive {
		return fmt.Errorf("у карты нет архива")
	}
	if !spec.HasFiles {
		return fmt.Errorf("список файлов карты пуст — загрузите архив заново")
	}
	return nil
}
