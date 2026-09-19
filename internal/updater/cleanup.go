package updater

import (
	"context"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/dockerapi"
	"github.com/vortanixapp/panel/pkg/updates"
)

const (
	cleanupDelay   = 2 * time.Minute
	cleanupPoll    = 15 * time.Second
	cleanupTimeout = time.Hour
)

func (s *Server) CleanupImages(ctx context.Context) {
	ctx, cancel := context.WithTimeout(ctx, cleanupTimeout)
	defer cancel()
	wait := cleanupDelay
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(wait):
		}
		wait = cleanupPoll
		done, err := s.removeStaleImages(ctx)
		if err != nil {
			log.Printf("очистка образов: %v", err)
			return
		}
		if done {
			return
		}
	}
}

func (s *Server) removeStaleImages(ctx context.Context) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	p, err := s.discover(ctx)
	if err != nil || p.Mode != "images" {
		return true, nil
	}
	_, own := dockerapi.SplitRef(p.Ref)
	if !updates.IsSemver(own) {
		return true, nil
	}
	if job := s.job(ctx, p); job != nil {
		switch job.State {
		case "running":
			return false, nil
		case "failed":
			return true, nil
		}
	}
	containers, err := s.docker.List(ctx, true, map[string][]string{
		"label": {"com.docker.compose.project=" + p.Name},
	})
	if err != nil {
		return true, err
	}
	images, err := s.docker.Images(ctx)
	if err != nil {
		return true, err
	}
	var removed []string
	for _, ref := range staleImages(own, containers, images) {
		if err := s.docker.RemoveImage(ctx, ref); err == nil {
			removed = append(removed, ref)
		}
	}
	if len(removed) > 0 {
		log.Printf("удалены старые образы панели (%d): %s", len(removed), strings.Join(removed, ", "))
	}
	return true, nil
}

func staleImages(own string, containers []dockerapi.Summary, images []dockerapi.ImageSummary) []string {
	ours := map[string]bool{}
	for _, c := range containers {
		if repo, tag := dockerapi.SplitRef(c.Image); tag == own {
			ours[repo] = true
		}
	}
	keep := map[string]bool{}
	for _, c := range containers {
		if repo, tag := dockerapi.SplitRef(c.Image); ours[repo] {
			keep[repo+":"+tag] = true
		}
	}
	var out []string
	for _, img := range images {
		tagged := false
		for _, ref := range img.RepoTags {
			if ref == "<none>:<none>" {
				continue
			}
			tagged = true
			repo, tag := dockerapi.SplitRef(ref)
			if ours[repo] && !keep[repo+":"+tag] {
				out = append(out, ref)
			}
		}
		if tagged {
			continue
		}
		for _, digest := range img.RepoDigests {
			if repo, _, _ := strings.Cut(digest, "@"); ours[repo] {
				out = append(out, img.ID)
				break
			}
		}
	}
	sort.Strings(out)
	return out
}
