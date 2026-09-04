package mailer

import (
	"fmt"
	"html"
	"strings"
)

type MailKind string

const (
	KindVerify MailKind = "verify"
	KindReset  MailKind = "reset"
)

func Compose(kind MailKind, lang, code string) (subject, plain, htmlBody string) {
	lang = strings.ToLower(lang)
	switch lang {
	case "ru", "en", "th", "zh":
	default:
		lang = "ru"
	}

	switch kind {
	case KindReset:
		return resetCopy(lang, code)
	default:
		return verifyCopy(lang, code)
	}
}

func verifyCopy(lang, code string) (string, string, string) {
	switch lang {
	case "en":
		subject := "Tarot: confirm your email"
		plain := fmt.Sprintf("Your confirmation code: %s\nIt is valid for 15 minutes.\nIf you did not create an account, ignore this email.", code)
		return subject, plain, htmlCode(subject, "Confirmation code", code, "Valid for 15 minutes. If this was not you, ignore the email.")
	case "th":
		subject := "Tarot: ยืนยันอีเมล"
		plain := fmt.Sprintf("รหัสยืนยันของคุณ: %s\nใช้ได้ 15 นาที\nถ้าไม่ได้สมัครบัญชี ให้เพิกเฉยต่ออีเมลนี้", code)
		return subject, plain, htmlCode(subject, "รหัสยืนยัน", code, "ใช้ได้ 15 นาที หากไม่ได้สมัคร ให้ละเว้นอีเมลนี้")
	case "zh":
		subject := "Tarot：确认邮箱"
		plain := fmt.Sprintf("您的确认码：%s\n15 分钟内有效。\n如非本人操作，请忽略本邮件。", code)
		return subject, plain, htmlCode(subject, "确认码", code, "15 分钟内有效。如非本人操作，请忽略。")
	default:
		subject := "Tarot: подтверждение email"
		plain := fmt.Sprintf("Код подтверждения: %s\nОн действует 15 минут.\nЕсли вы не регистрировались, просто удалите письмо.", code)
		return subject, plain, htmlCode(subject, "Код подтверждения", code, "Действует 15 минут. Если это были не вы — проигнорируйте письмо.")
	}
}

func resetCopy(lang, code string) (string, string, string) {
	switch lang {
	case "en":
		subject := "Tarot: password reset"
		plain := fmt.Sprintf("Your password reset code: %s\nIt is valid for 15 minutes.\nIf you did not request a reset, ignore this email.", code)
		return subject, plain, htmlCode(subject, "Password reset code", code, "Valid for 15 minutes. If you did not request this, ignore the email.")
	case "th":
		subject := "Tarot: รีเซ็ตรหัสผ่าน"
		plain := fmt.Sprintf("รหัสรีเซ็ตรหัสผ่าน: %s\nใช้ได้ 15 นาที\nถ้าไม่ได้ขอรีเซ็ต ให้เพิกเฉยต่ออีเมลนี้", code)
		return subject, plain, htmlCode(subject, "รหัสรีเซ็ตรหัสผ่าน", code, "ใช้ได้ 15 นาที หากไม่ได้ขอ ให้ละเว้นอีเมลนี้")
	case "zh":
		subject := "Tarot：重置密码"
		plain := fmt.Sprintf("您的重置码：%s\n15 分钟内有效。\n如非本人操作，请忽略本邮件。", code)
		return subject, plain, htmlCode(subject, "重置码", code, "15 分钟内有效。如非本人操作，请忽略。")
	default:
		subject := "Tarot: восстановление пароля"
		plain := fmt.Sprintf("Код для сброса пароля: %s\nОн действует 15 минут.\nЕсли вы не запрашивали сброс, просто удалите письмо.", code)
		return subject, plain, htmlCode(subject, "Код сброса пароля", code, "Действует 15 минут. Если это были не вы — проигнорируйте письмо.")
	}
}

func htmlCode(title, heading, code, hint string) string {
	return fmt.Sprintf(`<!doctype html><html><body style="font-family:sans-serif;background:#040307;color:#f7f4ea;padding:24px">
<div style="max-width:480px;margin:0 auto;background:#120f1c;border-radius:16px;padding:24px">
<p style="color:#f4d386;font-size:18px;margin:0 0 12px">%s</p>
<p style="margin:0 0 8px">%s</p>
<p style="font-size:32px;letter-spacing:6px;font-weight:700;color:#6c5ce7;margin:16px 0">%s</p>
<p style="opacity:.7;font-size:14px;margin:0">%s</p>
</div></body></html>`, html.EscapeString(title), html.EscapeString(heading), html.EscapeString(code), html.EscapeString(hint))
}
