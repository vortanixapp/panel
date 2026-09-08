package handlers

import (
	"strings"
	"testing"
)

// Вложения хранились в поле meta самого сообщения: на сообщение приходился
// ровно один файл, а заведённая под них таблица не использовалась нигде.
func TestAttachmentsUseDedicatedTable(t *testing.T) {
	src := readSource(t, "support_ext.go")

	if !strings.Contains(src, "core.support_message_attachments") {
		t.Fatal("таблица вложений снова не используется")
	}
	upload := funcBody(t, src, "UploadSupportAttachment")
	if !strings.Contains(upload, `r.MultipartForm.File["file"]`) {
		t.Error("загрузка принимает только один файл")
	}
	if !strings.Contains(upload, "supportAttachmentMaxCount") {
		t.Error("число файлов за раз ничем не ограничено")
	}
	if !strings.Contains(upload, "tx.Commit") {
		t.Error("сообщение и вложения пишутся не одной транзакцией")
	}
	if !strings.Contains(upload, "cleanup()") {
		t.Error("при ошибке записи файлы остаются на диске")
	}
}

// Ссылки на файлы, залитые до появления таблицы, указывают на id сообщения.
// Они должны продолжать открываться.
func TestDownloadFallsBackToLegacyAttachments(t *testing.T) {
	body := funcBody(t, readSource(t, "support_ext.go"), "DownloadSupportAttachment")

	if !strings.Contains(body, "core.support_message_attachments") {
		t.Error("новые вложения не ищутся в таблице")
	}
	if !strings.Contains(body, "meta") {
		t.Error("старые вложения перестали открываться")
	}
	if !strings.Contains(body, "claims.TenantID") {
		t.Error("выдача файла не ограничена арендатором")
	}
}

// Путь до файла хранится относительным: каталог загрузок задаётся настройкой и
// на разных установках разный.
func TestAttachmentPathIsRelative(t *testing.T) {
	upload := funcBody(t, readSource(t, "support_ext.go"), "UploadSupportAttachment")

	if !strings.Contains(upload, "relPath") {
		t.Fatal("путь до файла не приводится к относительному")
	}
	if strings.Contains(upload, "h.uploadDir, relPath") {
		t.Error("в базу уходит абсолютный путь")
	}
}
