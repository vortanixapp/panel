package docker

import "testing"

func TestStringFromPayloadOnMissingKey(t *testing.T) {
	payload := map[string]any{"present": " value "}

	if got := StringFromPayload(payload["missing"]); got != "" {
		t.Fatalf("отсутствующий ключ дал %q, ожидалась пустая строка", got)
	}
	if got := StringFromPayload(payload["present"]); got != "value" {
		t.Fatalf("значение %q, ожидалось \"value\"", got)
	}
	if got := StringFromPayload(nil); got != "" {
		t.Fatalf("nil дал %q", got)
	}
	if got := StringFromPayload(42); got != "42" {
		t.Fatalf("число дало %q", got)
	}
}

func TestResolveMySQLInstanceDefaults(t *testing.T) {
	inst, err := resolveMySQLInstance(map[string]any{})
	if err != nil {
		t.Fatalf("пустая нагрузка: %v", err)
	}
	if inst.Port != 3306 {
		t.Errorf("порт %d, ожидался 3306", inst.Port)
	}
	if inst.Container == "" {
		t.Error("контейнер не определён")
	}
}

func TestResolveMySQLInstanceByKey(t *testing.T) {
	payload := map[string]any{
		"mysql_instance_key": "mariadb-3308",
		"mysql_instances": []any{
			map[string]any{"key": "mysql80-3306", "container": "vtx-mysql80", "port": float64(3306), "root_password": "a"},
			map[string]any{"key": "mariadb-3308", "container": "vtx-mariadb", "port": float64(3308), "root_password": "b"},
		},
	}

	inst, err := resolveMySQLInstance(payload)
	if err != nil {
		t.Fatalf("выбор по ключу: %v", err)
	}
	if inst.Container != "vtx-mariadb" || inst.Port != 3308 {
		t.Errorf("выбран %s:%d, ожидался vtx-mariadb:3308", inst.Container, inst.Port)
	}
}

func TestResolveMySQLInstanceUnknownKey(t *testing.T) {
	payload := map[string]any{
		"mysql_instance_key": "no-such-instance",
		"mysql_instances": []any{
			map[string]any{"key": "mysql80-3306", "container": "vtx-mysql80", "port": float64(3306)},
		},
	}

	if _, err := resolveMySQLInstance(payload); err == nil {
		t.Fatal("неизвестный ключ принят без ошибки")
	}
}

func TestResolveMySQLInstanceDefaultKey(t *testing.T) {
	inst, err := resolveMySQLInstance(map[string]any{"mysql_instance_key": "default"})
	if err != nil {
		t.Fatalf("ключ default: %v", err)
	}
	if inst.Port != 3306 {
		t.Errorf("порт %d, ожидался 3306", inst.Port)
	}
}

// Пароль root ротируется шагом установки «MySQL», а новый агент видит только
// после пересоздания своего контейнера на шаге «Vortanix Agent». Между ними
// агент обязан оставаться работоспособным.
func TestMySQLInstanceFallsBackToLegacyPassword(t *testing.T) {
	inst := mysqlInstance{Container: "vortanix-mysql80-3306", Port: 3306, RootPassword: "случайный"}
	got := inst.passwords()
	if len(got) != 2 || got[0] != "случайный" {
		t.Fatalf("passwords() = %v, ожидался сначала актуальный пароль", got)
	}
	if got[1] != defaultMySQLRootPassword(inst.Container, inst.Port) {
		t.Errorf("вторым должен идти прежний вычисляемый пароль, получено %q", got[1])
	}

	same := mysqlInstance{Container: "c", Port: 1, RootPassword: defaultMySQLRootPassword("c", 1)}
	if len(same.passwords()) != 1 {
		t.Error("если пароль и есть вычисляемый, дублировать его незачем")
	}
}
