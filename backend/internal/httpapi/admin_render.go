package httpapi

import (
	"embed"
	"html/template"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/digkill/tarot-app/backend/internal/billing"
)

//go:embed adminhtml/*.html
var adminFS embed.FS

var msk = func() *time.Location {
	loc, err := time.LoadLocation("Europe/Moscow")
	if err != nil {
		return time.FixedZone("MSK", 3*3600)
	}
	return loc
}()

var adminTmpl = template.Must(template.New("admin").Funcs(template.FuncMap{
	"fmtTime": func(t time.Time) string {
		if t.IsZero() {
			return "—"
		}
		return t.In(msk).Format("02.01.2006 15:04")
	},
	"fmtTimePtr": func(t *time.Time) string {
		if t == nil || t.IsZero() {
			return "—"
		}
		return t.In(msk).Format("02.01.2006 15:04")
	},
	"fmtDay": func(t time.Time) string {
		return t.Format("02.01.2006")
	},
	"fmtRub":   formatRubKop,
	"fmtRubI":  func(v int) string { return formatRubKop(int64(v)) },
	"titleOf":  billing.Title,
	"deref":    derefString,
	"derefInt": derefInt,
	"short":    shortID,
	"kopRub":   func(k int) int { return k / 100 },
}).ParseFS(adminFS, "adminhtml/*.html"))

func derefString(s *string) string {
	if s == nil || *s == "" {
		return "—"
	}
	return *s
}

func derefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	if id == "" {
		return "—"
	}
	return id
}

func formatRubKop(kop int64) string {
	rub := kop / 100
	neg := rub < 0
	if neg {
		rub = -rub
	}
	s := strconv.FormatInt(rub, 10)
	var b strings.Builder
	n := len(s)
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			b.WriteByte(' ')
		}
		b.WriteRune(c)
	}
	out := b.String() + " ₽"
	if neg {
		return "−" + out
	}
	return out
}

type adminBase struct {
	Title      string
	AdminEmail string
	CSRF       string
	Flash      string
	Error      string
	Nav        string
}

type pagerView struct {
	Page    int
	Pages   int
	Total   int
	HasPrev bool
	HasNext bool
	PrevURL string
	NextURL string
}

func makePager(page, limit, total int, path string, q map[string]string) pagerView {
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 50
	}
	pages := (total + limit - 1) / limit
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	build := func(p int) string {
		u := path + "?page=" + strconv.Itoa(p)
		for k, v := range q {
			if v == "" {
				continue
			}
			u += "&" + k + "=" + template.URLQueryEscaper(v)
		}
		return u
	}
	return pagerView{
		Page:    page,
		Pages:   pages,
		Total:   total,
		HasPrev: page > 1,
		HasNext: page < pages,
		PrevURL: build(page - 1),
		NextURL: build(page + 1),
	}
}

func parsePage(r *http.Request) int {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		return 1
	}
	return page
}

func (h *Handler) renderAdmin(w http.ResponseWriter, name string, status int, data any) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-Robots-Tag", "noindex, nofollow")
	w.WriteHeader(status)
	if err := adminTmpl.ExecuteTemplate(w, name, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func flashFromQuery(r *http.Request) (ok string, errMsg string) {
	switch r.URL.Query().Get("ok") {
	case "premium_on":
		ok = "Премиум выдан."
	case "premium_off":
		ok = "Премиум снят."
	case "refunded":
		ok = "Возврат записан. Премиум снят, если не осталось других оплат."
	case "created":
		ok = "Транзакция создана."
	case "logged_out":
		ok = "Сессия завершена."
	case "deck_saved":
		ok = "Колода сохранена."
	case "imported":
		ok = "Карты импортированы (JPEG для приложения)."
	case "deck_granted":
		ok = "Колода выдана пользователю."
	}
	switch r.URL.Query().Get("err") {
	case "not_found":
		errMsg = "Не найдено."
	case "csrf":
		errMsg = "Сессия формы устарела, обновите страницу."
	case "not_refundable":
		errMsg = "Эту транзакцию нельзя вернуть."
	case "conflict":
		errMsg = "Такой invoice уже есть."
	case "validation":
		errMsg = "Проверьте поля формы."
	case "import":
		errMsg = "Не удалось разобрать ZIP. Нужны файлы как в tarot_deck_hq: 00_THE_FOOL.png и CARD_BACK.png."
	}
	if m := strings.TrimSpace(r.URL.Query().Get("msg")); m != "" && errMsg == "" && ok == "" {
		errMsg = m
	}
	return ok, errMsg
}
