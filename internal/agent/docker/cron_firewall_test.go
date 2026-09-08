package docker

import (
	"strings"
	"testing"
)

func TestValidateCronSchedule(t *testing.T) {
	got, err := validateCronSchedule("0 */6 * * *")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "0 */6 * * *" {
		t.Fatalf("got %q", got)
	}

	_, err = validateCronSchedule("bad")
	if err == nil {
		t.Fatal("expected error for invalid schedule")
	}
}

func TestValidateCronCommand(t *testing.T) {
	got, err := validateCronCommand("./restart.sh")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "./restart.sh" {
		t.Fatalf("got %q", got)
	}

	_, err = validateCronCommand("   ")
	if err == nil {
		t.Fatal("expected error for empty command")
	}
}

func TestFirewallChainName(t *testing.T) {
	id := "4c4c1bac-e564-4dd7-b78f-09291095467a"
	name := firewallChainName(id)
	if len(name) > 28 {
		t.Fatalf("chain name too long: %q", name)
	}
	if name != "VRTX-SRV-4c4c1bace5644dd7b78f"[:len(name)] {
		if name[:10] != "VRTX-SRV-4" {
			t.Fatalf("unexpected chain name: %q", name)
		}
	}
}

func TestFormatPortRange(t *testing.T) {
	to := 8080
	if got := formatPortRange(8080, &to); got != "8080" {
		t.Fatalf("single port: got %q", got)
	}
	to = 8090
	if got := formatPortRange(8080, &to); got != "8080:8090" {
		t.Fatalf("range: got %q", got)
	}
}

func TestDecodeCronJobs(t *testing.T) {
	raw := []any{
		map[string]any{"id": "1", "schedule": "* * * * *", "command": "echo", "enabled": true},
	}
	jobs, err := DecodeCronJobs(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 1 || jobs[0].ID != "1" {
		t.Fatalf("unexpected jobs: %+v", jobs)
	}
}

// Перевод строки в команде разрывает строку /etc/cron.d, и остаток cron читает
// как отдельную запись — с полем пользователя, то есть исполняет от root на
// ноде, мимо контейнера. Команду задаёт клиент из панели.
func TestCronCommandRejectsNewline(t *testing.T) {
	for _, bad := range []string{
		"./restart.sh\n* * * * * root id > /tmp/pwned",
		"./restart.sh\r\n* * * * * root id",
		"ok\nбольше одной строки",
	} {
		if _, err := validateCronCommand(bad); err == nil {
			t.Errorf("многострочная команда принята: %q", bad)
		}
	}
	if _, err := validateCronCommand("./restart.sh --now"); err != nil {
		t.Errorf("обычная команда отвергнута: %v", err)
	}
}

func TestCronCommentStripsInjection(t *testing.T) {
	got := cronComment("abc-123\n* * * * * root id")
	if strings.ContainsAny(got, "\n\r *") {
		t.Errorf("идентификатор попадает в файл неочищенным: %q", got)
	}
	if got != "abc-123rootid" {
		t.Errorf("cronComment(...) = %q", got)
	}
}
