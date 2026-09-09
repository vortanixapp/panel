package gamecatalog

import (
	"os"
	"strings"
)

type Runtime string

const (
	RuntimeSteam  Runtime = "steam"
	RuntimeSrcds  Runtime = "srcds"
	RuntimeJava   Runtime = "java"
	RuntimeNative Runtime = "native"
)

var runtimeImages = map[Runtime]struct {
	Image string
	Env   string
}{
	RuntimeSteam:  {"vortanix/runtime-steam:latest", "VORTANIX_RUNTIME_STEAM_IMAGE"},
	RuntimeSrcds:  {"vortanix/runtime-srcds:latest", "VORTANIX_RUNTIME_SRCDS_IMAGE"},
	RuntimeJava:   {"vortanix/runtime-java:latest", "VORTANIX_RUNTIME_JAVA_IMAGE"},
	RuntimeNative: {"vortanix/runtime-native:latest", "VORTANIX_RUNTIME_NATIVE_IMAGE"},
}

type Launch struct {
	Runtime  Runtime
	PortEnv  string
	Binaries []string
	Params   map[string]string
}

func LaunchOf(code string) (Launch, bool) {
	g, ok := Resolve(code)
	if !ok {
		return Launch{}, false
	}
	l, ok := launches[g.Key]
	return l, ok
}

func RuntimeImage(code string) string {
	l, ok := LaunchOf(code)
	if !ok {
		return ""
	}
	spec, ok := runtimeImages[l.Runtime]
	if !ok {
		return ""
	}
	if custom := strings.TrimSpace(os.Getenv(spec.Env)); custom != "" {
		return withRegistry(custom)
	}
	return withRegistry(spec.Image)
}

func RuntimeEnv(code string) map[string]string {
	g, ok := Resolve(code)
	if !ok {
		return nil
	}
	l, ok := launches[g.Key]
	if !ok {
		return nil
	}
	env := map[string]string{
		"VTX_GAME_KEY":  g.Key,
		"VTX_GAME_NAME": g.Name,
	}
	if len(l.Binaries) > 0 {
		env["VTX_SERVER_BINARIES"] = strings.Join(l.Binaries, ":")
	}
	if l.PortEnv != "" {
		env["VTX_PORT_ENV"] = l.PortEnv
	}
	for k, v := range l.Params {
		env["VTX_PARAM_"+strings.ToUpper(k)] = v
	}
	return env
}

func AllRuntimes() []Runtime {
	return []Runtime{RuntimeSteam, RuntimeSrcds, RuntimeJava, RuntimeNative}
}
