package httpapi

import "net/http"

// PrivacyPolicyPage mirrors the in-app privacy policy text (i18n/legal_ru.json
// and i18n/legal_en.json, "privacyParagraphs") verbatim, so App Store /
// RuStore reviewers and users have a public URL for it. Keep this in sync
// with those files if the policy changes.
func (h *Handler) PrivacyPolicyPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(privacyHTML))
}

const privacyHTML = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Tarot — Privacy Policy</title>
<style>
  body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#0B1220;color:#f7f4ea;margin:0;padding:32px 20px 64px;line-height:1.6}
  .wrap{max-width:680px;margin:0 auto}
  h1{color:#d4af37;font-size:26px;margin-bottom:4px}
  .sub{color:#9a93b3;margin-top:0;margin-bottom:32px;font-size:14px}
  a{color:#6c5ce7}
  p{margin:0 0 16px}
  .lang{border-top:1px solid rgba(212,175,55,0.25);margin-top:56px;padding-top:8px}
  .tag{display:inline-block;font-size:12px;letter-spacing:.05em;text-transform:uppercase;color:#9a93b3;margin-bottom:8px}
</style>
</head>
<body>
<div class="wrap">

  <span class="tag">Русский</span>
  <h1>Политика конфиденциальности</h1>
  <p class="sub">Редакция 1.0</p>

  <p>Настоящая Политика определяет порядок обработки персональных данных в мобильном приложении Tarot (пакет org.mediarise.tarot) и на сервере https://tarot.sorapure.fun. Политика составлена с учётом Федерального закона от 27.07.2006 № 152-ФЗ «О персональных данных».</p>
  <p>Оператор: MediaRise, приложение «Tarot». Контакт для запросов субъектов персональных данных: <a href="mailto:privacy@sorapure.fun">privacy@sorapure.fun</a>. Актуальный текст Политики доступен в приложении (Настройки → Политика конфиденциальности).</p>
  <p>Категории данных: адрес электронной почты; хэш пароля (сам пароль не хранится в открытом виде); факт и время согласия с документами; флаг премиум-подписки. IP-адрес и User-Agent собираются как техническая статистика входов и регистраций и хранятся только в зашифрованном виде (AES-256-GCM); в приложение они не возвращаются. Расклады и заметки по умолчанию остаются на устройстве.</p>
  <p>Цели обработки: регистрация и аутентификация; исполнение Пользовательского соглашения; хранение истории раскладов по вашей инициативе; формирование ИИ-толкований по запросу премиум-пользователя; фиксация доказательств согласия; рассмотрение обращений; защита от злоупотреблений.</p>
  <p>Правовое основание: ваше согласие (п. 1 ч. 1 ст. 6 152-ФЗ), а также исполнение договора-оферты (Пользовательского соглашения) в части, необходимой для работы аккаунта.</p>
  <p>Действия с данными: сбор, запись, систематизация, хранение, уточнение, использование, передача (в случаях ниже), блокирование, удаление и уничтожение.</p>
  <p>Передача третьим лицам. Хостинг и база данных размещаются у инфраструктурного провайдера Оператора. Для ИИ-толкований текст расклада (названия карт, позиции, язык интерфейса) может направляться провайдеру языковой модели Kie.ai. Email и пароль туда не передаются. Трансграничная передача возможна; регистрируясь, вы даёте согласие на такую передачу в указанных целях.</p>
  <p>Срок хранения: до удаления аккаунта вами либо до отзыва согласия, если иное не требуется законом (например, для защиты прав Оператора в разумный срок). После удаления аккаунта данные уничтожаются из продуктивной базы, резервные копии истекают по циклу бэкапов.</p>
  <p>Ваши права: получить сведения об обработке; требовать уточнения, блокирования или уничтожения; отозвать согласие; удалить аккаунт в Настройках. Для запроса напишите на <a href="mailto:privacy@sorapure.fun">privacy@sorapure.fun</a> с того же email, что указан в аккаунте.</p>
  <p>Отзыв согласия не влияет на законность обработки до момента отзыва. Без согласия на обработку ПДн аккаунт и облачные функции недоступны.</p>
  <p>Меры защиты: HTTPS, ограничение доступа к серверу, хеширование паролей, токены доступа с ограниченным сроком. Абсолютную безопасность в сети гарантировать нельзя.</p>
  <p>Приложение не предназначено для лиц младше 18 лет. Мы не собираем сознательно данные детей.</p>
  <p>Оператор может обновить Политику. Новая редакция публикуется в приложении с новым номером версии. Продолжение регистрации по новой версии возможно только после повторного согласия.</p>
  <p>Таро носит развлекательный и рефлексивный характер и не является медицинской, юридической, финансовой или психологической услугой.</p>

  <div class="lang">
  <span class="tag">English</span>
  <h1>Privacy Policy</h1>
  <p class="sub">Version 1.0</p>

  <p>This Policy describes how the Tarot app (package org.mediarise.tarot) and the server at https://tarot.sorapure.fun process personal data. For users in Russia it is prepared under Federal Law No. 152-FZ.</p>
  <p>Operator: MediaRise, the Tarot application. Data-subject requests: <a href="mailto:privacy@sorapure.fun">privacy@sorapure.fun</a>. The current Policy is always available in the app under Settings.</p>
  <p>Data we process: email; password hash (the password itself is not stored in plain text); the fact and time of your consent; premium flag. IP address and User-Agent are collected as technical sign-in statistics and stored only encrypted (AES-256-GCM); they are never returned to the app. Readings and notes stay on the device by default.</p>
  <p>Purposes: account creation and sign-in; performing the User Agreement; storing your reading history when you use cloud features; generating AI interpretations for premium users; keeping proof of consent; handling support requests; abuse prevention.</p>
  <p>Legal basis: your consent and performance of the User Agreement as needed to run the account.</p>
  <p>Operations: collection, recording, organisation, storage, updating, use, transfer (as described below), blocking, deletion and destruction.</p>
  <p>Third parties: hosting and the database run on the Operator's infrastructure. For AI interpretations, spread text (card names, positions, UI language) may be sent to the Kie.ai language-model provider. Email and password are not sent there. This may involve cross-border transfer; registration includes consent to that transfer for the stated purposes.</p>
  <p>Retention: until you delete the account or withdraw consent, unless a longer period is required by law. After deletion, production data is removed; backups expire on their rotation cycle.</p>
  <p>Your rights: access, correction, blocking or deletion; withdraw consent; delete the account in Settings. Email <a href="mailto:privacy@sorapure.fun">privacy@sorapure.fun</a> from the same address as your account.</p>
  <p>Withdrawing consent does not make earlier processing unlawful. Without personal-data consent the account and cloud features are unavailable.</p>
  <p>Safeguards include HTTPS, hashed passwords and short-lived access tokens. No online service can guarantee absolute security.</p>
  <p>The app is not intended for people under 18. We do not knowingly collect children's data.</p>
  <p>We may update this Policy. A new edition is published in the app with a new version number. New registration under a new edition requires a fresh consent.</p>
  <p>Tarot is for entertainment and self-reflection. It is not a medical, legal, financial or psychological service.</p>
  </div>

</div>
</body>
</html>`
