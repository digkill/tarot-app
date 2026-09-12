package httpapi

import (
	_ "embed"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
)

//go:embed webassets/icon.png
var marketingIcon []byte

func (h *Handler) MarketingIcon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=604800")
	_, _ = w.Write(marketingIcon)
}

type marketingFeature struct {
	Emoji, Title, Desc string
}

type marketingLang struct {
	Label          string // shown in the language switcher
	HTMLTitle      string
	MetaDesc       string
	Tagline        string
	Lead           string
	CTASupport     string
	CTAPrivacy     string
	PromoHTML      string
	FeaturesHeader string
	Features       [6]marketingFeature
	FooterPrivacy  string
	FooterSupport  string
}

// marketingLangOrder fixes the switcher's left-to-right order; map iteration
// order in Go is random, so this can't be derived from marketingLangs.
var marketingLangOrder = []string{"ru", "en", "th", "zh", "ja", "ko"}

var marketingLangs = map[string]marketingLang{
	"ru": {
		Label:          "Русский",
		HTMLTitle:      "Tarot — расклады Таро с ИИ",
		MetaDesc:       "Tarot — приложение раскладов Таро с ИИ-трактованиями. Классические и авторские расклады, библиотека колод, история раскладов.",
		Tagline:        "Карты не врут — но и не разжёвывают",
		Lead:           "ИИ переводит расклад Таро на человеческий язык: без воды, прямо к сути. Не гадание вслепую, а инструмент для честного разговора с собой.",
		CTASupport:     "Поддержка",
		CTAPrivacy:     "Конфиденциальность",
		PromoHTML:      `🎴 <strong>Японская колода</strong> — сакура, тушь, шёлк. Обычная цена 999 ₽, сейчас бесплатно. Ограниченное предложение.`,
		FeaturesHeader: "Что внутри",
		Features: [6]marketingFeature{
			{"🔮", "Расклады на любой случай", "От карты дня до полного расклада «Кельтский крест»."},
			{"🤖", "ИИ-трактования", "Разбор именно вашего расклада, а не заученные фразы."},
			{"🎴", "Библиотека колод", "Классика Райдер-Уэйт и авторские художественные колоды."},
			{"📖", "Значение карты по нажатию", "Прямое и перевёрнутое положение — сразу и подробно."},
			{"🕰", "История раскладов", "Возвращайтесь к прошлым раскладам и смотрите, что сбылось."},
			{"💎", "Премиум", "До 20 карт дня и 50 ИИ-трактований в сутки."},
		},
		FooterPrivacy: "Политика конфиденциальности",
		FooterSupport: "Поддержка",
	},
	"en": {
		Label:          "English",
		HTMLTitle:      "Tarot — tarot readings with AI",
		MetaDesc:       "Tarot is a tarot reading app with AI-powered interpretations. Classic and original spreads, a deck library, reading history.",
		Tagline:        "Cards don't lie — but they don't spell it out either",
		Lead:           "AI turns a tarot spread into plain language: no filler, straight to the point. Not blind fortune-telling — a tool for an honest conversation with yourself.",
		CTASupport:     "Support",
		CTAPrivacy:     "Privacy",
		PromoHTML:      `🎴 <strong>Japanese deck</strong> — sakura, ink, silk. Regular price 999 ₽, free right now. Limited-time offer.`,
		FeaturesHeader: "What's inside",
		Features: [6]marketingFeature{
			{"🔮", "A spread for any occasion", "From a single daily card to the full Celtic Cross."},
			{"🤖", "AI interpretations", "A reading of your actual spread, not stock phrases."},
			{"🎴", "A deck library", "The classic Rider-Waite and exclusive art decks."},
			{"📖", "Full card meanings on tap", "Upright and reversed — right there, in detail."},
			{"🕰", "Reading history", "Revisit past spreads and see what came true."},
			{"💎", "Premium", "Up to 20 daily cards and 50 AI interpretations a day."},
		},
		FooterPrivacy: "Privacy Policy",
		FooterSupport: "Support",
	},
	"th": {
		Label:          "ไทย",
		HTMLTitle:      "Tarot — ไพ่ทาโรต์พร้อม AI",
		MetaDesc:       "Tarot แอปดูไพ่ทาโรต์พร้อมการตีความด้วย AI",
		Tagline:        "การ์ดไม่โกหก — แต่ก็ไม่อธิบายให้ฟังตรงๆ",
		Lead:           "AI แปลไพ่ทาโรต์เป็นภาษาที่เข้าใจง่าย ตรงประเด็น ไม่ใช่การทำนายแบบสุ่มสี่สุ่มห้า แต่เป็นเครื่องมือสำหรับการพูดคุยกับตัวเองอย่างตรงไปตรงมา",
		CTASupport:     "ฝ่ายสนับสนุน",
		CTAPrivacy:     "ความเป็นส่วนตัว",
		PromoHTML:      `🎴 <strong>สำรับไพ่ญี่ปุ่น</strong> — ซากุระ หมึก ผ้าไหม ราคาปกติ 999 ₽ ตอนนี้ฟรี ข้อเสนอมีเวลาจำกัด`,
		FeaturesHeader: "สิ่งที่อยู่ภายใน",
		Features: [6]marketingFeature{
			{"🔮", "ไพ่สำหรับทุกโอกาส", "ตั้งแต่ไพ่ประจำวันไปจนถึงราคาแบบเคลติกครอสเต็มรูปแบบ"},
			{"🤖", "การตีความด้วย AI", "วิเคราะห์ไพ่ของคุณโดยเฉพาะ ไม่ใช่วลีสำเร็จรูป"},
			{"🎴", "คลังสำรับไพ่", "ไรเดอร์-เวทคลาสสิกและสำรับไพ่ศิลปะสุดพิเศษ"},
			{"📖", "ความหมายไพ่ครบถ้วนเพียงแตะเดียว", "ตั้งตรงและกลับหัว พร้อมรายละเอียด"},
			{"🕰", "ประวัติการดูไพ่", "ย้อนดูไพ่ที่เคยดูและดูว่าอะไรเป็นจริง"},
			{"💎", "พรีเมียม", "ไพ่ประจำวันสูงสุด 20 ใบ และการตีความ AI 50 ครั้งต่อวัน"},
		},
		FooterPrivacy: "นโยบายความเป็นส่วนตัว",
		FooterSupport: "ฝ่ายสนับสนุน",
	},
	"zh": {
		Label:          "中文",
		HTMLTitle:      "Tarot — AI 塔罗牌占卜",
		MetaDesc:       "Tarot 是一款带有 AI 解读功能的塔罗牌应用",
		Tagline:        "牌不会说谎——但也不会把话说透",
		Lead:           "AI 将塔罗牌阵翻译成通俗易懂的语言：直击要点，没有废话。这不是盲目占卜，而是一个与自己坦诚对话的工具。",
		CTASupport:     "支持",
		CTAPrivacy:     "隐私",
		PromoHTML:      `🎴 <strong>日式塔罗牌</strong> —— 樱花、水墨、丝绸。原价 999 ₽，现在免费。限时优惠。`,
		FeaturesHeader: "应用亮点",
		Features: [6]marketingFeature{
			{"🔮", "适合任何场合的牌阵", "从单张日运牌到完整的凯尔特十字牌阵。"},
			{"🤖", "AI 解读", "针对您的实际牌阵进行解读，而非套话。"},
			{"🎴", "牌组库", "经典韦特塔罗与独家艺术牌组。"},
			{"📖", "点击查看完整牌意", "正位与逆位，详细说明。"},
			{"🕰", "占卜历史", "回顾过去的牌阵，看看哪些成真了。"},
			{"💎", "高级会员", "每天最多 20 张日运牌，50 次 AI 解读。"},
		},
		FooterPrivacy: "隐私政策",
		FooterSupport: "支持",
	},
	"ja": {
		Label:          "日本語",
		HTMLTitle:      "Tarot — AIによるタロット占い",
		MetaDesc:       "TarotはAI解釈機能付きのタロット占いアプリです",
		Tagline:        "カードは嘘をつかない——でも、すべてを言い切ってはくれない",
		Lead:           "AIがタロットのスプレッドを分かりやすい言葉に変換します。無駄なく、核心をついて。当てずっぽうの占いではなく、自分自身と正直に向き合うためのツールです。",
		CTASupport:     "サポート",
		CTAPrivacy:     "プライバシー",
		PromoHTML:      `🎴 <strong>和風デッキ</strong> — 桜、墨、絹。通常価格999₽のところ、今なら無料。期間限定。`,
		FeaturesHeader: "できること",
		Features: [6]marketingFeature{
			{"🔮", "あらゆる場面に対応するスプレッド", "デイリーカード1枚からフルのケルト十字まで。"},
			{"🤖", "AIによる解釈", "定型文ではなく、あなたのスプレッドそのものを解釈。"},
			{"🎴", "デッキライブラリ", "クラシックなライダー・ウェイトから限定アートデッキまで。"},
			{"📖", "タップでカードの意味を全文表示", "正位置・逆位置を詳しく解説。"},
			{"🕰", "リーディング履歴", "過去のスプレッドを見返し、何が当たったか確認。"},
			{"💎", "プレミアム", "1日最大20枚のデイリーカードと50回のAI解釈。"},
		},
		FooterPrivacy: "プライバシーポリシー",
		FooterSupport: "サポート",
	},
	"ko": {
		Label:          "한국어",
		HTMLTitle:      "Tarot — AI 타로 리딩",
		MetaDesc:       "Tarot는 AI 해석 기능을 갖춘 타로 카드 앱입니다",
		Tagline:        "카드는 거짓말하지 않는다 — 하지만 다 설명해주지도 않는다",
		Lead:           "AI가 타로 스프레드를 이해하기 쉬운 언어로 번역합니다. 군더더기 없이 핵심만. 맹목적인 점술이 아니라 자신과 솔직하게 대화하는 도구입니다.",
		CTASupport:     "지원",
		CTAPrivacy:     "개인정보",
		PromoHTML:      `🎴 <strong>일본풍 덱</strong> — 벚꽃, 먹, 실크. 정가 999 ₽, 지금은 무료. 한정 특가.`,
		FeaturesHeader: "앱 소개",
		Features: [6]marketingFeature{
			{"🔮", "모든 상황에 맞는 스프레드", "데일리 카드 한 장부터 완전한 켈틱 크로스까지."},
			{"🤖", "AI 해석", "정형화된 문구가 아닌, 당신의 실제 스프레드에 대한 해석."},
			{"🎴", "덱 라이브러리", "클래식 라이더-웨이트와 독점 아트 덱."},
			{"📖", "탭 한 번으로 카드 의미 전체 보기", "정방향과 역방향, 자세한 설명까지."},
			{"🕰", "리딩 기록", "지난 스프레드를 다시 보고 무엇이 실현되었는지 확인하세요."},
			{"💎", "프리미엄", "하루 최대 20장의 데일리 카드와 50회의 AI 해석."},
		},
		FooterPrivacy: "개인정보처리방침",
		FooterSupport: "지원",
	},
}

// pickMarketingLang resolves the page language: an explicit /{lang}/ path
// segment wins, then a legacy ?lang= query param, then the first supported
// language found in Accept-Language, falling back to Russian (the app's
// primary market).
func pickMarketingLang(r *http.Request) string {
	if p := strings.ToLower(strings.TrimSpace(chi.URLParam(r, "lang"))); p != "" {
		if _, ok := marketingLangs[p]; ok {
			return p
		}
	}
	if q := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang"))); q != "" {
		if _, ok := marketingLangs[q]; ok {
			return q
		}
	}
	accept := strings.ToLower(r.Header.Get("Accept-Language"))
	for _, part := range strings.Split(accept, ",") {
		code := strings.TrimSpace(strings.SplitN(strings.TrimSpace(part), ";", 2)[0])
		code = strings.SplitN(code, "-", 2)[0]
		if _, ok := marketingLangs[code]; ok {
			return code
		}
	}
	return "ru"
}

// MarketingPage is the public landing page used as the App Store / RuStore
// "Marketing URL" (and reused for the per-locale Privacy/Marketing URL
// fields — same URL works for every locale, the page just picks the
// matching language). Update the download links once the app is live in
// each store.
func (h *Handler) MarketingPage(w http.ResponseWriter, r *http.Request) {
	code := pickMarketingLang(r)
	lang := marketingLangs[code]

	var switcher strings.Builder
	for _, c := range marketingLangOrder {
		l := marketingLangs[c]
		class := "switch-link"
		if c == code {
			class += " active"
		}
		switcher.WriteString(fmt.Sprintf(`<a class="%s" href="/%s/">%s</a>`, class, c, l.Label))
	}

	var features strings.Builder
	for _, f := range lang.Features {
		features.WriteString(fmt.Sprintf(`
    <div class="feature">
      <span class="emoji">%s</span>
      <h3>%s</h3>
      <p>%s</p>
    </div>`, f.Emoji, f.Title, f.Desc))
	}

	page := fmt.Sprintf(marketingTemplate,
		code, lang.HTMLTitle, lang.MetaDesc,
		switcher.String(),
		lang.Tagline, lang.Lead, lang.CTASupport, lang.CTAPrivacy,
		lang.PromoHTML,
		lang.FeaturesHeader, features.String(),
		lang.FooterPrivacy, lang.FooterSupport,
	)

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(page))
}

const marketingTemplate = `<!doctype html>
<html lang="%s">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>%s</title>
<meta name="description" content="%s">
<style>
  :root{color-scheme:dark}
  *{box-sizing:border-box}
  body{font-family:-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif;background:#0B1220;color:#f7f4ea;margin:0;line-height:1.6}
  .wrap{max-width:720px;margin:0 auto;padding:0 20px 80px}
  .switcher{display:flex;flex-wrap:wrap;gap:8px;justify-content:center;padding:20px 20px 0}
  .switch-link{color:#9a93b3;text-decoration:none;font-size:13px;padding:4px 10px;border-radius:8px}
  .switch-link.active{color:#0B1220;background:#d4af37;font-weight:600}
  .hero{text-align:center;padding:32px 20px 40px}
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

<nav class="switcher">%s</nav>

<div class="wrap">

  <div class="hero">
    <img class="icon" src="/marketing-icon.png" alt="Tarot">
    <h1>Tarot</h1>
    <p class="tagline">%s</p>
    <p class="lead">%s</p>
    <div class="cta">
      <a class="btn btn-primary" href="/support">%s</a>
      <a class="btn btn-ghost" href="/privacy">%s</a>
    </div>
  </div>

  <div class="promo">
    %s
  </div>

  <h2>%s</h2>
  <div class="features">%s
  </div>

  <footer>
    <p>tarot@sorapure.fun · <a href="/privacy">%s</a> · <a href="/support">%s</a></p>
  </footer>

</div>
</body>
</html>`
