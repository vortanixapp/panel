package settingsreg

const (
	GroupUploads = "uploads"
)

var (
	UploadsAvatarMB = def(Setting{
		Key: "uploads.avatar_mb", Group: GroupUploads, Section: "files",
		Kind: KindInt, Default: "4", Min: 1, Max: 20, Unit: UnitMB, Public: true,
	})
	UploadsSiteAssetMB = def(Setting{
		Key: "uploads.site_asset_mb", Group: GroupUploads, Section: "files",
		Kind: KindInt, Default: "5", Min: 1, Max: 32, Unit: UnitMB, Public: true,
	})
	UploadsServerFileMB = def(Setting{
		Key: "uploads.server_file_mb", Group: GroupUploads, Section: "files",
		Kind: KindInt, Default: "256", Min: 16, Max: 4096, Unit: UnitMB,
	})

	SupportAttachmentMB = def(Setting{
		Key: "support.attachment_mb", Group: GroupUploads, Section: "support",
		Kind: KindInt, Default: "8", Min: 1, Max: 64, Unit: UnitMB, Public: true,
	})
	SupportAttachmentCount = def(Setting{
		Key: "support.attachment_count", Group: GroupUploads, Section: "support",
		Kind: KindInt, Default: "5", Min: 1, Max: 20, Unit: UnitCount, Public: true,
	})
	SupportTicketFilesMax = def(Setting{
		Key: "support.ticket_files_max", Group: GroupUploads, Section: "support",
		Kind: KindInt, Default: "25", Min: 1, Max: 200, Unit: UnitCount,
	})
	SupportDailyMB = def(Setting{
		Key: "support.user_daily_mb", Group: GroupUploads, Section: "support",
		Kind: KindInt, Default: "100", Min: 10, Max: 2048, Unit: UnitMB,
	})
	SupportBodyMax = def(Setting{
		Key: "support.body_max", Group: GroupUploads, Section: "support",
		Kind: KindInt, Default: "20000", Min: 500, Max: 100000, Unit: UnitCount, Public: true,
	})
	SupportSubjectMax = def(Setting{
		Key: "support.subject_max", Group: GroupUploads, Section: "support",
		Kind: KindInt, Default: "200", Min: 20, Max: 500, Unit: UnitCount,
	})
	SupportNewPerHour = def(Setting{
		Key: "support.new_per_hour", Group: GroupUploads, Section: "support",
		Kind: KindInt, Default: "10", Min: 1, Max: 200, Unit: UnitCount,
	})
	SupportReplyPerWindow = def(Setting{
		Key: "support.reply_per_10min", Group: GroupUploads, Section: "support",
		Kind: KindInt, Default: "30", Min: 1, Max: 500, Unit: UnitCount,
	})
)
