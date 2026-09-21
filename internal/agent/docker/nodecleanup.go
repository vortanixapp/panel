package docker

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/dockerapi"
)

const (
	CleanDanglingImages      = "dangling_images"
	CleanAgentImages         = "agent_images"
	CleanGameImages          = "game_images"
	CleanForeignImages       = "foreign_images"
	CleanAbandonedContainers = "abandoned_containers"
	CleanHelperContainers    = "helper_containers"
	CleanTempFiles           = "temp_files"
	CleanBuildCache          = "build_cache"

	tempFileAge = time.Hour
)

var tempPrefixes = []string{"vtx-archive-", "vtx-plugin-", "vtx-extract-", "vtx-bundle"}

type CleanupItem struct {
	ID        string `json:"id"`
	Category  string `json:"category"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	Selected  bool   `json:"selected"`
	Running   bool   `json:"running,omitempty"`
	ServerID  string `json:"server_id,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
}

type CleanupScope struct {
	KnownServers map[string]bool
	GameRepos    map[string]bool
}

type CleanupSkip struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type CleanupResult struct {
	Removed    []CleanupItem `json:"removed"`
	Skipped    []CleanupSkip `json:"skipped"`
	FreedBytes int64         `json:"freed_bytes"`
}

func repoWithoutRegistry(repo string) string {
	first, rest, ok := strings.Cut(repo, "/")
	if ok && (strings.ContainsAny(first, ".:") || first == "localhost") {
		return rest
	}
	return repo
}

func imageCategory(img dockerapi.ImageSummary, self selfRef, scope CleanupScope) string {
	tags := img.Tags()
	if len(tags) == 0 {
		return CleanDanglingImages
	}
	for _, t := range tags {
		repo, _ := dockerapi.SplitRef(t)
		if self.repo != "" && repoWithoutRegistry(repo) == repoWithoutRegistry(self.repo) {
			return CleanAgentImages
		}
	}
	for _, t := range tags {
		repo, _ := dockerapi.SplitRef(t)
		if scope.GameRepos[repoWithoutRegistry(repo)] {
			return CleanGameImages
		}
	}
	return CleanForeignImages
}

func CleanupPreview(ctx context.Context, scope CleanupScope) ([]CleanupItem, error) {
	cli := dockerapi.New()
	containers, err := cli.List(ctx, true, nil)
	if err != nil {
		return nil, err
	}
	images, err := cli.Images(ctx)
	if err != nil {
		return nil, err
	}
	self := agentSelf(ctx, cli)
	used := map[string]bool{self.image: true}
	for _, c := range containers {
		used[c.ImageID] = true
	}
	sizes := map[string]int64{}
	if df, err := cli.SystemDF(ctx); err == nil {
		for _, c := range df.Containers {
			sizes[c.ID] = c.SizeRw
		}
		var cache int64
		for _, b := range df.BuildCache {
			if !b.InUse {
				cache += b.Size
			}
		}
		if cache > 0 {
			sizes["buildcache"] = cache
		}
	}

	items := []CleanupItem{}
	for _, img := range images {
		if used[img.ID] {
			continue
		}
		cat := imageCategory(img, self, scope)
		name := strings.Join(img.Tags(), ", ")
		if name == "" {
			name = shortID(img.ID)
		}
		items = append(items, CleanupItem{
			ID: "image:" + img.ID, Category: cat, Name: name, SizeBytes: img.Size,
			Selected:  cat != CleanForeignImages,
			CreatedAt: time.Unix(img.Created, 0).UTC().Format(time.RFC3339),
		})
	}
	for _, c := range containers {
		kind, serverID := classifyContainer(c, self)
		running := c.State == "running" || c.State == "restarting"
		switch {
		case kind == KindServer && scope.KnownServers != nil && !scope.KnownServers[serverID]:
			items = append(items, CleanupItem{
				ID: "container:" + c.ID, Category: CleanAbandonedContainers, Name: c.CleanName(),
				SizeBytes: sizes[c.ID], Selected: !running, Running: running, ServerID: serverID,
				CreatedAt: time.Unix(c.Created, 0).UTC().Format(time.RFC3339),
			})
		case kind == KindHelper && !running && c.State != "created":
			items = append(items, CleanupItem{
				ID: "container:" + c.ID, Category: CleanHelperContainers, Name: c.CleanName(),
				SizeBytes: sizes[c.ID], Selected: true,
				CreatedAt: time.Unix(c.Created, 0).UTC().Format(time.RFC3339),
			})
		}
	}
	items = append(items, staleTempFiles()...)
	if n := sizes["buildcache"]; n > 0 {
		items = append(items, CleanupItem{ID: "buildcache", Category: CleanBuildCache, Name: "build cache", SizeBytes: n, Selected: true})
	}
	sort.SliceStable(items, func(a, b int) bool {
		if items[a].Category != items[b].Category {
			return items[a].Category < items[b].Category
		}
		return items[a].SizeBytes > items[b].SizeBytes
	})
	return items, nil
}

func shortID(id string) string {
	id = strings.TrimPrefix(id, "sha256:")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}

func staleTempFiles() []CleanupItem {
	dir := os.TempDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []CleanupItem
	for _, e := range entries {
		if !hasTempPrefix(e.Name()) {
			continue
		}
		info, err := e.Info()
		if err != nil || time.Since(info.ModTime()) < tempFileAge {
			continue
		}
		path := filepath.Join(dir, e.Name())
		size := info.Size()
		if e.IsDir() {
			size = dirBytes(path)
		}
		out = append(out, CleanupItem{
			ID: "tmp:" + e.Name(), Category: CleanTempFiles, Name: path, SizeBytes: size, Selected: true,
			CreatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}
	return out
}

func hasTempPrefix(name string) bool {
	for _, p := range tempPrefixes {
		if strings.HasPrefix(name, p) {
			return true
		}
	}
	return false
}

func CleanupApply(ctx context.Context, scope CleanupScope, ids []string) (CleanupResult, error) {
	res := CleanupResult{Removed: []CleanupItem{}, Skipped: []CleanupSkip{}}
	fresh, err := CleanupPreview(ctx, scope)
	if err != nil {
		return res, err
	}
	byID := map[string]CleanupItem{}
	for _, it := range fresh {
		byID[it.ID] = it
	}
	cli := dockerapi.New()
	order := func(id string) int {
		switch {
		case strings.HasPrefix(id, "container:"):
			return 0
		case strings.HasPrefix(id, "image:"):
			return 1
		}
		return 2
	}
	sorted := append([]string(nil), ids...)
	sort.SliceStable(sorted, func(a, b int) bool { return order(sorted[a]) < order(sorted[b]) })
	seen := map[string]bool{}
	for _, id := range sorted {
		if seen[id] {
			continue
		}
		seen[id] = true
		it, ok := byID[id]
		if !ok {
			res.Skipped = append(res.Skipped, CleanupSkip{ID: id, Reason: "уже не подходит под чистку: объект удалён, используется или изменился"})
			continue
		}
		var err error
		freed := it.SizeBytes
		switch {
		case strings.HasPrefix(id, "container:"):
			err = cli.Remove(ctx, strings.TrimPrefix(id, "container:"), it.Running)
		case strings.HasPrefix(id, "image:"):
			err = removeImage(ctx, cli, strings.TrimPrefix(id, "image:"))
		case strings.HasPrefix(id, "tmp:"):
			name := strings.TrimPrefix(id, "tmp:")
			if !hasTempPrefix(name) || strings.ContainsAny(name, `/\`) {
				err = errors.New("недопустимое имя")
			} else {
				err = os.RemoveAll(filepath.Join(os.TempDir(), name))
			}
		case id == "buildcache":
			freed, err = cli.PruneBuildCache(ctx)
		default:
			err = errors.New("неизвестный пункт")
		}
		if err != nil {
			res.Skipped = append(res.Skipped, CleanupSkip{ID: id, Name: it.Name, Reason: err.Error()})
			continue
		}
		it.SizeBytes = freed
		res.Removed = append(res.Removed, it)
		res.FreedBytes += freed
	}
	return res, nil
}

func removeImage(ctx context.Context, cli *dockerapi.Client, id string) error {
	imgs, err := cli.Images(ctx)
	if err != nil {
		return err
	}
	for _, img := range imgs {
		if img.ID != id {
			continue
		}
		tags := img.Tags()
		if len(tags) == 0 {
			return cli.RemoveImage(ctx, id)
		}
		for _, t := range tags {
			if err := cli.RemoveImage(ctx, t); err != nil && !dockerapi.IsNotFound(err) {
				return err
			}
		}
		return nil
	}
	return nil
}
