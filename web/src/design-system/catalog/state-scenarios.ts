import type { FeedbackTone } from "../components/feedback-tone";

export type StateScenario = {
  id: string;
  title: { ru: string; en: string };
  message: { ru: string; en: string };
  action: { ru: string; en: string };
  tone: FeedbackTone;
};
export const stateScenarios: readonly StateScenario[] = [
  {
    id: "UISTATE-01",
    title: { ru: "Загрузка", en: "Loading" },
    message: {
      ru: "Получаем записи. Сумма пока неизвестна.",
      en: "Loading entries. The amount is not known yet.",
    },
    action: { ru: "Показать ответ", en: "Show response" },
    tone: "neutral",
  },
  {
    id: "UISTATE-02",
    title: { ru: "Обновление", en: "Refreshing" },
    message: {
      ru: "На экране данные на 7 сентября, 12:00. Получаем новые записи.",
      en: "Showing data from September 7, 12:00. Fetching new entries.",
    },
    action: { ru: "Завершить обновление", en: "Complete refresh" },
    tone: "neutral",
  },
  {
    id: "UISTATE-03",
    title: { ru: "Пусто", en: "Empty" },
    message: {
      ru: "Добавьте первый счёт, чтобы увидеть доступные средства.",
      en: "Add your first account to see available funds.",
    },
    action: { ru: "Добавить счёт", en: "Add account" },
    tone: "neutral",
  },
  {
    id: "UISTATE-04",
    title: { ru: "Нет совпадений", en: "No matches" },
    message: {
      ru: "По этим условиям ничего не найдено. Ваш фильтр сохранён.",
      en: "Nothing matches these conditions. Your filter is retained.",
    },
    action: { ru: "Изменить фильтр", en: "Change filter" },
    tone: "neutral",
  },
  {
    id: "UISTATE-05",
    title: { ru: "Частичные данные", en: "Partial data" },
    message: {
      ru: "История счёта доступна с 1 сентября. Более ранние операции не входят в итог.",
      en: "Account history starts September 1. Earlier entries are outside the result.",
    },
    action: { ru: "Посмотреть покрытие", en: "View coverage" },
    tone: "warning",
  },
  {
    id: "UISTATE-06",
    title: { ru: "Устаревшие данные", en: "Stale data" },
    message: {
      ru: "Последнее обновление — 7 сентября, 12:00. Новые расходы ещё не учтены.",
      en: "Last updated September 7, 12:00. New expenses are not included yet.",
    },
    action: { ru: "Обновить", en: "Refresh" },
    tone: "warning",
  },
  {
    id: "UISTATE-07",
    title: { ru: "Ошибка", en: "Error" },
    message: {
      ru: "Не удалось получить ответ. Введённые данные сохранены в этой вкладке.",
      en: "Could not obtain a response. Your input is retained in this tab.",
    },
    action: { ru: "Проверить результат", en: "Check outcome" },
    tone: "error",
  },
  {
    id: "UISTATE-08",
    title: { ru: "Offline", en: "Offline" },
    message: {
      ru: "Нет связи. Черновик остаётся в памяти вкладки; сохранение не подтверждено.",
      en: "No connection. The draft stays in tab memory; saving is not confirmed.",
    },
    action: { ru: "Проверить связь", en: "Check connection" },
    tone: "warning",
  },
  {
    id: "UISTATE-09",
    title: { ru: "Сохранение", en: "Saving" },
    message: {
      ru: "Проверяем результат отправки. Повторная отправка недоступна.",
      en: "Checking the submitted request. Duplicate submission is unavailable.",
    },
    action: { ru: "Показать ответ", en: "Show response" },
    tone: "neutral",
  },
  {
    id: "UISTATE-10",
    title: { ru: "Исход неизвестен", en: "Unknown outcome" },
    message: {
      ru: "Ответ потерян. Проверим исходную команду, прежде чем повторять действие.",
      en: "The response was lost. Check the original command before repeating the action.",
    },
    action: { ru: "Проверить исходную команду", en: "Check original command" },
    tone: "warning",
  },
  {
    id: "UISTATE-11",
    title: { ru: "Конфликт версии", en: "Version conflict" },
    message: {
      ru: "Участник Б изменил запись. Сравните версии; ваш ввод сохранён.",
      en: "Member B changed the entry. Compare versions; your input is retained.",
    },
    action: { ru: "Сравнить изменения", en: "Compare changes" },
    tone: "warning",
  },
  {
    id: "UISTATE-12",
    title: { ru: "Недостаточно прав", en: "Insufficient permissions" },
    message: {
      ru: "Личную цель может изменить только её владелец — участник Б.",
      en: "Only its owner, Member B, can change this personal goal.",
    },
    action: { ru: "Посмотреть права", en: "View permissions" },
    tone: "warning",
  },
  {
    id: "UISTATE-13",
    title: { ru: "Сессия истекла", en: "Session expired" },
    message: {
      ru: "Войдите снова под своей учётной записью. Финансовые данные скрыты.",
      en: "Sign in again with your own account. Financial details are hidden.",
    },
    action: { ru: "Перейти ко входу", en: "Go to sign-in" },
    tone: "warning",
  },
  {
    id: "UISTATE-14",
    title: { ru: "Ожидание AI", en: "Waiting for AI" },
    message: {
      ru: "Обработка приостановлена до обновления лимита. Обычный учёт доступен.",
      en: "Processing is paused until the allowance renews. Ordinary accounting remains available.",
    },
    action: { ru: "Внести вручную", en: "Enter manually" },
    tone: "neutral",
  },
  {
    id: "UISTATE-15",
    title: { ru: "Нужен банковский вход", en: "Bank sign-in needed" },
    message: {
      ru: "В банк должен войти владелец подключения — участник А. Партнёру не нужны его секреты.",
      en: "Connection owner Member A needs to sign in to the bank. Their partner does not need the credentials.",
    },
    action: { ru: "Посмотреть подключение", en: "View connection" },
    tone: "warning",
  },
  {
    id: "UISTATE-16",
    title: { ru: "Подтверждено", en: "Confirmed" },
    message: {
      ru: "В демонстрации получен подтверждённый результат. Доступны подробности и исправление.",
      en: "The demo received a confirmed outcome. Details and correction are available.",
    },
    action: { ru: "Исправить", en: "Correct" },
    tone: "success",
  },
  {
    id: "UISTATE-17",
    title: { ru: "Отмена", en: "Cancelled" },
    message: {
      ru: "Действие отменено. Нового подтверждённого результата нет.",
      en: "The action was cancelled. There is no new confirmed outcome.",
    },
    action: { ru: "Повторить явно", en: "Retry explicitly" },
    tone: "neutral",
  },
];
