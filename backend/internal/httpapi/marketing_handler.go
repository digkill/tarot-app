package httpapi

import (
	_ "embed"
	"net/http"
)

//go:embed webassets/icon.png
var marketingIcon []byte

func (h *Handler) MarketingIcon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	_, _ = w.Write(marketingIcon)
}

// MarketingPage is the public landing page used as the App Store / RuStore
// "Marketing URL". Update the download links once the app is live in each
// store.
func (h *Handler) MarketingPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(marketingHTML))
}

const marketingHTML = `<!doctype html>
<html lang="ru">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Tarot — расклады Таро с ИИ</title>
<meta name="description" content="Tarot — приложение раскладов Таро с ИИ-трактованиями. Классические и авторские расклады, библиотека колод, история раскладов.">
<style>
  :root{color-scheme:dark}
  *{box-sizing:border-box}
  body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#0B1220;color:#f7f4ea;margin:0;line-height:1.6}
  .wrap{max-width:720px;margin:0 auto;padding:0 20px 80px}
  .hero{text-align:center;padding:64px 20px 40px}
  .icon{width:96px;height:96px;border-radius:22px;box-shadow:0 12px 40px rgba(108,92,231,0.35)}
  h1{color:#f7f4ea;font-size:34px;margin:24px 0 8px}
  .tagline{color:#d4af37;font-size:18px;margin:0 0 24px}
  .lead{color:#c9c4dd;font-size:16px;max-width:520px;margin:0 auto 32px}
  .cta{display:inline-flex;gap:12px;flex-wrap:wrap;justify-content:center}
  .btn{display:inline-block;padding:12px 24px;border-radius:14px;font-weight:600;text-decoration:none;font-size:15px}
  .btn-primary{background:#6c5ce7;color:#fff}
  .btn-ghost{border:1px solid rgba(212,175,55,0.5);color:#d4af37}
  .promo{background:rgba(212,175,55,0.1);border:1px solid rgba(212,175,55,0.3);border-radius:18px;padding:20px 24px;margin:40px 0;text-align:center}
  .promo strong{color:#d4af37}
  h2{color:#d4af37;font-size:22px;text-align:center;margin:56px 0 28px}
  .features{display:grid;grid-template-columns:1fr 1fr;gap:20px}
  @media (max-width:520px){.features{grid-template-columns:1fr}}
  .feature{background:rgba(108,92,231,0.08);border-radius:16px;padding:20px}
  .feature .emoji{font-size:26px;display:block;margin-bottom:8px}
  .feature h3{margin:0 0 6px;font-size:16px;color:#f7f4ea}
  .feature p{margin:0;color:#9a93b3;font-size:14px}
  footer{text-align:center;margin-top:64px;padding-top:24px;border-top:1px solid rgba(212,175,55,0.15);color:#9a93b3;font-size:13px}
  footer a{color:#6c5ce7;text-decoration:none}
</style>
</head>
<body>
<div class="wrap">

  <div class="hero">
    <img class="icon" src="/marketing-icon.png" alt="Tarot">
    <h1>Tarot</h1>
    <p class="tagline">Карты не врут — но и не разжёвывают</p>
    <p class="lead">ИИ переводит расклад Таро на человеческий язык: без воды, прямо к сути. Не гадание вслепую, а инструмент для честного разговора с собой.</p>
    <div class="cta">
      <a class="btn btn-primary" href="/support">Поддержка</a>
      <a class="btn btn-ghost" href="/privacy">Конфиденциальность</a>
    </div>
  </div>

  <div class="promo">
    🎴 <strong>Японская колода</strong> — сакура, тушь, шёлк. Обычная цена 999 ₽, сейчас бесплатно. Ограниченное предложение.
  </div>

  <h2>Что внутри</h2>
  <div class="features">
    <div class="feature">
      <span class="emoji">🔮</span>
      <h3>Расклады на любой случай</h3>
      <p>От карты дня до полного расклада «Кельтский крест».</p>
    </div>
    <div class="feature">
      <span class="emoji">🤖</span>
      <h3>ИИ-трактования</h3>
      <p>Разбор именно вашего расклада, а не заученные фразы.</p>
    </div>
    <div class="feature">
      <span class="emoji">🎴</span>
      <h3>Библиотека колод</h3>
      <p>Классика Райдер-Уэйт и авторские художественные колоды.</p>
    </div>
    <div class="feature">
      <span class="emoji">📖</span>
      <h3>Значение карты по нажатию</h3>
      <p>Прямое и перевёрнутое положение — сразу и подробно.</p>
    </div>
    <div class="feature">
      <span class="emoji">🕰</span>
      <h3>История раскладов</h3>
      <p>Возвращайтесь к прошлым раскладам и смотрите, что сбылось.</p>
    </div>
    <div class="feature">
      <span class="emoji">💎</span>
      <h3>Премиум</h3>
      <p>До 20 карт дня и 50 ИИ-трактований в сутки.</p>
    </div>
  </div>

  <footer>
    <p>Вопросы: <a href="mailto:tarot@sorapure.fun">tarot@sorapure.fun</a> · <a href="/privacy">Политика конфиденциальности</a> · <a href="/support">Поддержка</a></p>
  </footer>

</div>
</body>
</html>`
