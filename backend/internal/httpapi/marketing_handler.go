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
  .lang{border-top:1px solid rgba(212,175,55,0.25);margin-top:64px;padding-top:8px}
  .tag{display:block;text-align:center;font-size:12px;letter-spacing:.05em;text-transform:uppercase;color:#9a93b3;margin-bottom:8px}
</style>
</head>
<body>
<div class="wrap">

  <span class="tag">Русский</span>
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

  <div class="lang">
  <span class="tag">English</span>
  <div class="hero">
    <img class="icon" src="/marketing-icon.png" alt="Tarot">
    <h1>Tarot</h1>
    <p class="tagline">Cards don't lie — but they don't spell it out either</p>
    <p class="lead">AI turns a tarot spread into plain language: no filler, straight to the point. Not blind fortune-telling — a tool for an honest conversation with yourself.</p>
    <div class="cta">
      <a class="btn btn-primary" href="/support">Support</a>
      <a class="btn btn-ghost" href="/privacy">Privacy</a>
    </div>
  </div>

  <div class="promo">
    🎴 <strong>Japanese deck</strong> — sakura, ink, silk. Regular price 999 ₽, free right now. Limited-time offer.
  </div>

  <h2>What's inside</h2>
  <div class="features">
    <div class="feature">
      <span class="emoji">🔮</span>
      <h3>A spread for any occasion</h3>
      <p>From a single daily card to the full Celtic Cross.</p>
    </div>
    <div class="feature">
      <span class="emoji">🤖</span>
      <h3>AI interpretations</h3>
      <p>A reading of your actual spread, not stock phrases.</p>
    </div>
    <div class="feature">
      <span class="emoji">🎴</span>
      <h3>A deck library</h3>
      <p>The classic Rider-Waite and exclusive art decks.</p>
    </div>
    <div class="feature">
      <span class="emoji">📖</span>
      <h3>Full card meanings on tap</h3>
      <p>Upright and reversed — right there, in detail.</p>
    </div>
    <div class="feature">
      <span class="emoji">🕰</span>
      <h3>Reading history</h3>
      <p>Revisit past spreads and see what came true.</p>
    </div>
    <div class="feature">
      <span class="emoji">💎</span>
      <h3>Premium</h3>
      <p>Up to 20 daily cards and 50 AI interpretations a day.</p>
    </div>
  </div>
  </div>

  <div class="lang">
  <span class="tag">ไทย</span>
  <div class="hero">
    <img class="icon" src="/marketing-icon.png" alt="Tarot">
    <h1>Tarot</h1>
    <p class="tagline">การ์ดไม่โกหก — แต่ก็ไม่อธิบายให้ฟังตรงๆ</p>
    <p class="lead">AI แปลไพ่ทาโรต์เป็นภาษาที่เข้าใจง่าย ตรงประเด็น ไม่ใช่การทำนายแบบสุ่มสี่สุ่มห้า แต่เป็นเครื่องมือสำหรับการพูดคุยกับตัวเองอย่างตรงไปตรงมา</p>
    <div class="cta">
      <a class="btn btn-primary" href="/support">ฝ่ายสนับสนุน</a>
      <a class="btn btn-ghost" href="/privacy">ความเป็นส่วนตัว</a>
    </div>
  </div>

  <div class="promo">
    🎴 <strong>สำรับไพ่ญี่ปุ่น</strong> — ซากุระ หมึก ผ้าไหม ราคาปกติ 999 ₽ ตอนนี้ฟรี ข้อเสนอมีเวลาจำกัด
  </div>

  <h2>สิ่งที่อยู่ภายใน</h2>
  <div class="features">
    <div class="feature">
      <span class="emoji">🔮</span>
      <h3>ไพ่สำหรับทุกโอกาส</h3>
      <p>ตั้งแต่ไพ่ประจำวันไปจนถึงราคาแบบเคลติกครอสเต็มรูปแบบ</p>
    </div>
    <div class="feature">
      <span class="emoji">🤖</span>
      <h3>การตีความด้วย AI</h3>
      <p>วิเคราะห์ไพ่ของคุณโดยเฉพาะ ไม่ใช่วลีสำเร็จรูป</p>
    </div>
    <div class="feature">
      <span class="emoji">🎴</span>
      <h3>คลังสำรับไพ่</h3>
      <p>ไรเดอร์-เวทคลาสสิกและสำรับไพ่ศิลปะสุดพิเศษ</p>
    </div>
    <div class="feature">
      <span class="emoji">📖</span>
      <h3>ความหมายไพ่ครบถ้วนเพียงแตะเดียว</h3>
      <p>ตั้งตรงและกลับหัว พร้อมรายละเอียด</p>
    </div>
    <div class="feature">
      <span class="emoji">🕰</span>
      <h3>ประวัติการดูไพ่</h3>
      <p>ย้อนดูไพ่ที่เคยดูและดูว่าอะไรเป็นจริง</p>
    </div>
    <div class="feature">
      <span class="emoji">💎</span>
      <h3>พรีเมียม</h3>
      <p>ไพ่ประจำวันสูงสุด 20 ใบ และการตีความ AI 50 ครั้งต่อวัน</p>
    </div>
  </div>
  </div>

  <div class="lang">
  <span class="tag">中文</span>
  <div class="hero">
    <img class="icon" src="/marketing-icon.png" alt="Tarot">
    <h1>Tarot</h1>
    <p class="tagline">牌不会说谎——但也不会把话说透</p>
    <p class="lead">AI 将塔罗牌阵翻译成通俗易懂的语言：直击要点，没有废话。这不是盲目占卜，而是一个与自己坦诚对话的工具。</p>
    <div class="cta">
      <a class="btn btn-primary" href="/support">支持</a>
      <a class="btn btn-ghost" href="/privacy">隐私</a>
    </div>
  </div>

  <div class="promo">
    🎴 <strong>日式塔罗牌</strong> —— 樱花、水墨、丝绸。原价 999 ₽，现在免费。限时优惠。
  </div>

  <h2>应用亮点</h2>
  <div class="features">
    <div class="feature">
      <span class="emoji">🔮</span>
      <h3>适合任何场合的牌阵</h3>
      <p>从单张日运牌到完整的凯尔特十字牌阵。</p>
    </div>
    <div class="feature">
      <span class="emoji">🤖</span>
      <h3>AI 解读</h3>
      <p>针对您的实际牌阵进行解读，而非套话。</p>
    </div>
    <div class="feature">
      <span class="emoji">🎴</span>
      <h3>牌组库</h3>
      <p>经典韦特塔罗与独家艺术牌组。</p>
    </div>
    <div class="feature">
      <span class="emoji">📖</span>
      <h3>点击查看完整牌意</h3>
      <p>正位与逆位，详细说明。</p>
    </div>
    <div class="feature">
      <span class="emoji">🕰</span>
      <h3>占卜历史</h3>
      <p>回顾过去的牌阵，看看哪些成真了。</p>
    </div>
    <div class="feature">
      <span class="emoji">💎</span>
      <h3>高级会员</h3>
      <p>每天最多 20 张日运牌，50 次 AI 解读。</p>
    </div>
  </div>
  </div>

  <div class="lang">
  <span class="tag">日本語</span>
  <div class="hero">
    <img class="icon" src="/marketing-icon.png" alt="Tarot">
    <h1>Tarot</h1>
    <p class="tagline">カードは嘘をつかない——でも、すべてを言い切ってはくれない</p>
    <p class="lead">AIがタロットのスプレッドを分かりやすい言葉に変換します。無駄なく、核心をついて。当てずっぽうの占いではなく、自分自身と正直に向き合うためのツールです。</p>
    <div class="cta">
      <a class="btn btn-primary" href="/support">サポート</a>
      <a class="btn btn-ghost" href="/privacy">プライバシー</a>
    </div>
  </div>

  <div class="promo">
    🎴 <strong>和風デッキ</strong> — 桜、墨、絹。通常価格999₽のところ、今なら無料。期間限定。
  </div>

  <h2>できること</h2>
  <div class="features">
    <div class="feature">
      <span class="emoji">🔮</span>
      <h3>あらゆる場面に対応するスプレッド</h3>
      <p>デイリーカード1枚からフルのケルト十字まで。</p>
    </div>
    <div class="feature">
      <span class="emoji">🤖</span>
      <h3>AIによる解釈</h3>
      <p>定型文ではなく、あなたのスプレッドそのものを解釈。</p>
    </div>
    <div class="feature">
      <span class="emoji">🎴</span>
      <h3>デッキライブラリ</h3>
      <p>クラシックなライダー・ウェイトから限定アートデッキまで。</p>
    </div>
    <div class="feature">
      <span class="emoji">📖</span>
      <h3>タップでカードの意味を全文表示</h3>
      <p>正位置・逆位置を詳しく解説。</p>
    </div>
    <div class="feature">
      <span class="emoji">🕰</span>
      <h3>リーディング履歴</h3>
      <p>過去のスプレッドを見返し、何が当たったか確認。</p>
    </div>
    <div class="feature">
      <span class="emoji">💎</span>
      <h3>プレミアム</h3>
      <p>1日最大20枚のデイリーカードと50回のAI解釈。</p>
    </div>
  </div>
  </div>

  <div class="lang">
  <span class="tag">한국어</span>
  <div class="hero">
    <img class="icon" src="/marketing-icon.png" alt="Tarot">
    <h1>Tarot</h1>
    <p class="tagline">카드는 거짓말하지 않는다 — 하지만 다 설명해주지도 않는다</p>
    <p class="lead">AI가 타로 스프레드를 이해하기 쉬운 언어로 번역합니다. 군더더기 없이 핵심만. 맹목적인 점술이 아니라 자신과 솔직하게 대화하는 도구입니다.</p>
    <div class="cta">
      <a class="btn btn-primary" href="/support">지원</a>
      <a class="btn btn-ghost" href="/privacy">개인정보</a>
    </div>
  </div>

  <div class="promo">
    🎴 <strong>일본풍 덱</strong> — 벚꽃, 먹, 실크. 정가 999 ₽, 지금은 무료. 한정 특가.
  </div>

  <h2>앱 소개</h2>
  <div class="features">
    <div class="feature">
      <span class="emoji">🔮</span>
      <h3>모든 상황에 맞는 스프레드</h3>
      <p>데일리 카드 한 장부터 완전한 켈틱 크로스까지.</p>
    </div>
    <div class="feature">
      <span class="emoji">🤖</span>
      <h3>AI 해석</h3>
      <p>정형화된 문구가 아닌, 당신의 실제 스프레드에 대한 해석.</p>
    </div>
    <div class="feature">
      <span class="emoji">🎴</span>
      <h3>덱 라이브러리</h3>
      <p>클래식 라이더-웨이트와 독점 아트 덱.</p>
    </div>
    <div class="feature">
      <span class="emoji">📖</span>
      <h3>탭 한 번으로 카드 의미 전체 보기</h3>
      <p>정방향과 역방향, 자세한 설명까지.</p>
    </div>
    <div class="feature">
      <span class="emoji">🕰</span>
      <h3>리딩 기록</h3>
      <p>지난 스프레드를 다시 보고 무엇이 실현되었는지 확인하세요.</p>
    </div>
    <div class="feature">
      <span class="emoji">💎</span>
      <h3>프리미엄</h3>
      <p>하루 최대 20장의 데일리 카드와 50회의 AI 해석.</p>
    </div>
  </div>
  </div>

  <footer>
    <p>tarot@sorapure.fun · <a href="/privacy">Privacy Policy</a> · <a href="/support">Support</a></p>
  </footer>

</div>
</body>
</html>`
