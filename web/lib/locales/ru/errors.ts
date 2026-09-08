// Страницы ошибок 401/403/404/500 и режим обслуживания.
export const errors = {
  "errors.back": "Назад",
  "errors.to_home": "На главную",
  "errors.sign_in": "Войти",

  "errors.403.title": "Доступ запрещён",
  "errors.403.desc": "У вас нет прав для просмотра этой страницы.",

  "errors.404.title": "Страница не найдена",
  "errors.404.desc": "Запрашиваемая страница не существует.",

  "errors.401.title": "Требуется авторизация",
  "errors.401.desc": "Войдите в аккаунт для доступа к этой странице.",

  "errors.500.title": "Что-то пошло не так",
  "errors.500.desc": "Попробуйте позже или вернитесь на главную.",

  "errors.maintenance.title": "Панель на обслуживании",
  "errors.maintenance.desc":
    "Сайт временно недоступен. Мы скоро вернёмся — загляните позже.",

  // Сообщения из lib/api.ts: их показывают тостом на месте вызова.
  "errors.activity.export_failed": "Не удалось выгрузить журнал",
  "errors.analytics.export_failed": "Не удалось выгрузить отчёт",
  "errors.billing.receipt_failed": "Не удалось получить квитанцию",
  "errors.logs.stream_unavailable": "Поток журнала недоступен ({code})",
  "errors.logs.stream_interrupted": "Поток журнала прервался",
  "errors.support.upload_failed": "Ошибка загрузки ({code})",
  "errors.support.attachment_download_failed": "Не удалось скачать вложение",
  "errors.upload.failed": "Не удалось загрузить файл",
};
