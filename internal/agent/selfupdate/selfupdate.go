package selfupdate

import (
	"context"
	"flag"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/vortanixapp/panel/pkg/dockerapi"
)

const (
	labelUpgrade    = "app.vortanix.agent-upgrade"
	labelTarget     = "app.vortanix.agent-upgrade.target"
	errorPrefix     = "ОШИБКА: "
	connectedMarker = "connected to relay as node="
)

var (
	startDelay     = 3 * time.Second
	pollInterval   = 3 * time.Second
	connectTimeout = 90 * time.Second
)

type Reporter func(stage, target, errMsg string)

type Result struct {
	State  string
	Target string
	Error  string
}

func helperName(self string) string {
	return self + "-upgrade"
}

func socketSource(c *dockerapi.Container) string {
	if binds, ok := c.HostConfig["Binds"].([]any); ok {
		for _, b := range binds {
			parts := strings.Split(fmt.Sprint(b), ":")
			if len(parts) >= 2 && parts[1] == "/var/run/docker.sock" {
				return parts[0]
			}
		}
	}
	return "/var/run/docker.sock"
}

func Start(ctx context.Context, image, target string, report Reporter) {
	if target == "" {
		_, target = dockerapi.SplitRef(image)
	}
	d := dockerapi.New()
	id := dockerapi.SelfContainerID()
	if id == "" {
		report("failed", target, "агент запущен не в контейнере, обновите его вручную")
		return
	}
	self, err := d.Inspect(ctx, id)
	if err != nil {
		report("failed", target, "нет доступа к docker: "+err.Error())
		return
	}

	report("pulling", target, "")
	pullCtx, cancel := context.WithTimeout(ctx, 20*time.Minute)
	err = d.Pull(pullCtx, image, nil)
	cancel()
	if err != nil {
		report("failed", target, "образ "+image+" не загрузился: "+err.Error())
		return
	}

	name := helperName(self.CleanName())
	if prev, err := d.Inspect(ctx, name); err == nil {
		if prev.State.Running {
			report("failed", target, "обновление агента уже идёт")
			return
		}
		_ = d.Remove(ctx, prev.ID, true)
	}
	spec := map[string]any{
		"Image": self.Image,
		"Cmd":   []string{"agent-upgrade", "--container", self.ID, "--image", image},
		"User":  "0:0",
		"Labels": map[string]string{
			labelUpgrade: self.ID,
			labelTarget:  target,
		},
		"HostConfig": map[string]any{
			"Binds":         []string{socketSource(self) + ":/var/run/docker.sock"},
			"RestartPolicy": map[string]any{"Name": "no"},
		},
	}
	helperID, err := d.Create(ctx, name, spec)
	if err != nil {
		report("failed", target, "не удалось подготовить перезапуск: "+err.Error())
		return
	}
	report("restarting", target, "")
	if err := d.Start(ctx, helperID); err != nil {
		_ = d.Remove(ctx, helperID, true)
		report("failed", target, "не удалось запустить перезапуск: "+err.Error())
	}
}

func CollectResult(ctx context.Context) (*Result, bool) {
	d := dockerapi.New()
	id := dockerapi.SelfContainerID()
	if id == "" {
		return nil, false
	}
	self, err := d.Inspect(ctx, id)
	if err != nil {
		return nil, false
	}
	name := helperName(self.CleanName())
	deadline := time.Now().Add(4 * time.Minute)
	for {
		helper, err := d.Inspect(ctx, name)
		if err != nil {
			return nil, false
		}
		if !helper.State.Running && !helper.State.Restarting && helper.State.Status != "created" {
			res := &Result{Target: helper.Labels()[labelTarget], State: "done"}
			if helper.State.ExitCode != 0 {
				res.State = "failed"
				res.Error = fmt.Sprintf("перезапуск завершился с кодом %d", helper.State.ExitCode)
				if lines, err := d.Logs(ctx, helper.ID, 100); err == nil {
					for i := len(lines) - 1; i >= 0; i-- {
						if strings.HasPrefix(lines[i], errorPrefix) {
							res.Error = strings.TrimPrefix(lines[i], errorPrefix)
							break
						}
					}
				}
			}
			_ = d.Remove(ctx, helper.ID, true)
			return res, true
		}
		if time.Now().After(deadline) {
			return nil, false
		}
		time.Sleep(5 * time.Second)
	}
}

func say(format string, args ...any) {
	fmt.Printf("%s  %s\n", time.Now().Format("15:04:05"), fmt.Sprintf(format, args...))
}

func RunHelper(args []string) int {
	fs := flag.NewFlagSet("agent-upgrade", flag.ContinueOnError)
	container := fs.String("container", "", "")
	image := fs.String("image", "", "")
	if err := fs.Parse(args); err != nil || *container == "" || *image == "" {
		fmt.Printf("%sнужны --container и --image\n", errorPrefix)
		return 2
	}
	if err := upgrade(context.Background(), dockerapi.New(), *container, *image); err != nil {
		fmt.Printf("%s%s\n", errorPrefix, err)
		return 1
	}
	return 0
}

func upgrade(ctx context.Context, d *dockerapi.Client, containerID, image string) error {
	time.Sleep(startDelay)

	old, err := d.Inspect(ctx, containerID)
	if err != nil {
		return fmt.Errorf("контейнер агента не найден: %w", err)
	}
	name := old.CleanName()
	oldImage, err := d.InspectImage(ctx, old.Image)
	if err != nil {
		oldImage = &dockerapi.Image{Config: map[string]any{}}
	}
	if _, err := d.InspectImage(ctx, image); err != nil {
		return fmt.Errorf("образ %s не найден: %w", image, err)
	}

	spec := buildSpec(old, oldImage, image)
	primary, extra := networks(old)
	if primary != nil {
		spec["NetworkingConfig"] = map[string]any{"EndpointsConfig": primary}
	}

	say("Остановка %s", name)
	if err := d.Stop(ctx, old.ID, 30); err != nil {
		return fmt.Errorf("агент не остановился: %w", err)
	}
	previous := name + "-previous"
	if stale, err := d.Inspect(ctx, previous); err == nil && !stale.State.Running {
		_ = d.Remove(ctx, stale.ID, true)
	}
	if err := d.Rename(ctx, old.ID, previous); err != nil {
		_ = d.Start(ctx, old.ID)
		return fmt.Errorf("агент не переименован: %w", err)
	}

	rollback := func(newID string, cause error) error {
		if newID != "" {
			_ = d.Remove(ctx, newID, true)
		}
		_ = d.Rename(ctx, old.ID, name)
		if err := d.Start(ctx, old.ID); err != nil {
			return fmt.Errorf("%v; прежний агент не запустился: %v", cause, err)
		}
		say("Возвращена прежняя версия агента")
		return fmt.Errorf("%v, агент остался на прежней версии", cause)
	}

	say("Запуск %s", image)
	newID, err := d.Create(ctx, name, spec)
	if err != nil {
		return rollback("", fmt.Errorf("новый контейнер не создан: %w", err))
	}
	for network, endpoint := range extra {
		if err := d.ConnectNetwork(ctx, network, newID, endpoint); err != nil {
			return rollback(newID, fmt.Errorf("сеть %s не подключена: %w", network, err))
		}
	}
	if err := d.Start(ctx, newID); err != nil {
		return rollback(newID, fmt.Errorf("новый агент не запустился: %w", err))
	}
	if err := waitConnected(ctx, d, newID); err != nil {
		return rollback(newID, err)
	}
	if err := d.Remove(ctx, old.ID, true); err != nil {
		say("Прежний контейнер не удалён: %v", err)
	}
	say("Агент обновлён: %s", image)
	return nil
}

func waitConnected(ctx context.Context, d *dockerapi.Client, id string) error {
	deadline := time.Now().Add(connectTimeout)
	last := ""
	for time.Now().Before(deadline) {
		time.Sleep(pollInterval)
		lines, err := d.Logs(ctx, id, 200)
		if err != nil {
			continue
		}
		for _, line := range lines {
			if strings.Contains(line, connectedMarker) {
				return nil
			}
		}
		if len(lines) > 0 {
			last = lines[len(lines)-1]
		}
	}
	seconds := int(connectTimeout.Seconds())
	if last != "" {
		return fmt.Errorf("новый агент не подключился к панели за %d с: %s", seconds, last)
	}
	return fmt.Errorf("новый агент не подключился к панели за %d с", seconds)
}

func buildSpec(old *dockerapi.Container, oldImage *dockerapi.Image, image string) map[string]any {
	imageCfg := oldImage.Config
	if imageCfg == nil {
		imageCfg = map[string]any{}
	}
	spec := map[string]any{}
	for k, v := range old.Config {
		spec[k] = v
	}
	spec["Image"] = image
	spec["Env"] = filterEnv(old.Config["Env"], imageCfg["Env"])
	spec["Labels"] = filterLabels(old.Config["Labels"], imageCfg["Labels"])
	for _, key := range []string{"Cmd", "Entrypoint", "WorkingDir", "User", "StopSignal", "Healthcheck", "ExposedPorts", "Volumes", "Shell", "OnBuild"} {
		if v, ok := spec[key]; ok && reflect.DeepEqual(v, imageCfg[key]) {
			delete(spec, key)
		}
	}
	if host, _ := spec["Hostname"].(string); host != "" && strings.HasPrefix(old.ID, host) {
		delete(spec, "Hostname")
	}
	delete(spec, "MacAddress")
	spec["HostConfig"] = old.HostConfig
	return spec
}

func filterEnv(containerEnv, imageEnv any) []string {
	inherited := map[string]bool{}
	if list, ok := imageEnv.([]any); ok {
		for _, item := range list {
			inherited[fmt.Sprint(item)] = true
		}
	}
	out := []string{}
	if list, ok := containerEnv.([]any); ok {
		for _, item := range list {
			kv := fmt.Sprint(item)
			if inherited[kv] || strings.HasPrefix(kv, "VORTANIX_VERSION=") {
				continue
			}
			out = append(out, kv)
		}
	}
	return out
}

func filterLabels(containerLabels, imageLabels any) map[string]any {
	inherited, _ := imageLabels.(map[string]any)
	out := map[string]any{}
	if labels, ok := containerLabels.(map[string]any); ok {
		for k, v := range labels {
			if iv, ok := inherited[k]; ok && iv == v {
				continue
			}
			out[k] = v
		}
	}
	return out
}

func networks(old *dockerapi.Container) (map[string]any, map[string]map[string]any) {
	mode, _ := old.HostConfig["NetworkMode"].(string)
	if mode == "host" || mode == "none" || strings.HasPrefix(mode, "container:") {
		return nil, nil
	}
	defaultMode := mode == "" || mode == "default" || mode == "bridge"
	var primary map[string]any
	extra := map[string]map[string]any{}
	for name, endpoint := range old.NetworkSettings.Networks {
		if defaultMode && name == "bridge" {
			continue
		}
		cfg := endpointConfig(endpoint, old.ID)
		if name == mode {
			primary = map[string]any{name: cfg}
			continue
		}
		extra[name] = cfg
	}
	return primary, extra
}

func endpointConfig(endpoint map[string]any, containerID string) map[string]any {
	cfg := map[string]any{}
	if aliases, ok := endpoint["Aliases"].([]any); ok {
		kept := []string{}
		for _, a := range aliases {
			alias := fmt.Sprint(a)
			if alias != "" && !strings.HasPrefix(containerID, alias) {
				kept = append(kept, alias)
			}
		}
		if len(kept) > 0 {
			cfg["Aliases"] = kept
		}
	}
	for _, key := range []string{"IPAMConfig", "Links", "DriverOpts"} {
		if v, ok := endpoint[key]; ok && v != nil {
			cfg[key] = v
		}
	}
	return cfg
}
