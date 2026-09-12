package httpapi

import "net/http"

func (h *Handler) SupportPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(supportHTML))
}

const supportHTML = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Tarot — Support</title>
<style>
  body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#0B1220;color:#f7f4ea;margin:0;padding:32px 20px 64px;line-height:1.6}
  .wrap{max-width:640px;margin:0 auto}
  h1{color:#d4af37;font-size:28px;margin-bottom:4px}
  .sub{color:#9a93b3;margin-top:0;margin-bottom:32px}
  h2{color:#d4af37;font-size:18px;margin-top:32px}
  a{color:#6c5ce7}
  .card{background:rgba(108,92,231,0.1);border-radius:16px;padding:20px;margin:24px 0}
  ul{padding-left:20px}
  .lang{border-top:1px solid rgba(212,175,55,0.25);margin-top:56px;padding-top:8px}
  .tag{display:inline-block;font-size:12px;letter-spacing:.05em;text-transform:uppercase;color:#9a93b3;margin-bottom:8px}
</style>
</head>
<body>
<div class="wrap">

  <span class="tag">Русский</span>
  <h1>Tarot — поддержка</h1>
  <p class="sub">Приложение раскладов Таро с ИИ-трактованиями</p>

  <div class="card">
    <strong>Есть вопрос или проблема?</strong><br>
    Напишите нам: <a href="mailto:tarot@sorapure.fun">tarot@sorapure.fun</a><br>
    Отвечаем обычно в течение 1–2 рабочих дней.
  </div>

  <h2>Частые вопросы</h2>
  <ul>
    <li><strong>Подписка не активировалась после оплаты.</strong> Перезайдите в приложение (потяните экран вниз до обновления) — статус подписки синхронизируется автоматически. Если не помогло — напишите нам с указанием e-mail аккаунта.</li>
    <li><strong>Как отменить подписку?</strong> Через магазин, где оформлена покупка (App Store / RuStore) — раздел управления подписками в настройках устройства.</li>
    <li><strong>Пропала колода или расклад.</strong> Проверьте, что вы вошли в тот же аккаунт, что и при покупке — колоды и история привязаны к аккаунту, а не к устройству.</li>
    <li><strong>Как удалить аккаунт и данные?</strong> В приложении: Настройки → Удалить аккаунт. Это необратимо удаляет все ваши данные.</li>
  </ul>

  <h2>Документы</h2>
  <p>Пользовательское соглашение и политика конфиденциальности доступны в приложении: Настройки → Правовая информация.</p>

  <div class="lang">
  <span class="tag">English</span>
  <h1>Tarot — Support</h1>
  <p class="sub">A tarot reading app with AI-powered interpretations</p>

  <div class="card">
    <strong>Have a question or an issue?</strong><br>
    Email us: <a href="mailto:tarot@sorapure.fun">tarot@sorapure.fun</a><br>
    We usually reply within 1–2 business days.
  </div>

  <h2>Frequently asked questions</h2>
  <ul>
    <li><strong>My subscription didn't activate after payment.</strong> Reopen the app (pull down to refresh) — subscription status syncs automatically. If that doesn't help, email us with the account's email address.</li>
    <li><strong>How do I cancel my subscription?</strong> Through the store where you purchased it (App Store / RuStore) — the subscriptions section in your device settings.</li>
    <li><strong>A deck or reading is missing.</strong> Make sure you're signed into the same account you purchased with — decks and history are tied to your account, not your device.</li>
    <li><strong>How do I delete my account and data?</strong> In the app: Settings → Delete account. This permanently removes all your data.</li>
  </ul>

  <h2>Legal</h2>
  <p>The terms of service and privacy policy are available in the app: Settings → Legal information.</p>
  </div>

</div>
</body>
</html>`
