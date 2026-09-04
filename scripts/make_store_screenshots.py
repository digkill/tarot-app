#!/usr/bin/env python3
"""Compose RuStore phone screenshots (1080x1920, 9:16 JPG) from app assets and copy."""

from pathlib import Path

from PIL import Image, ImageDraw, ImageFilter, ImageFont

ROOT = Path("/Users/digkill/Projects/js/tarot-app")
CARDS = ROOT / "assets" / "cards"
OUT = ROOT / "assets" / "store" / "screenshots"
DESKTOP = Path("/Users/digkill/Desktop/rustore-screens")

W, H = 1080, 1920
BG = (4, 3, 7)
CREAM = (247, 244, 234)
GOLD = (244, 211, 134)
PURPLE = (108, 92, 231)
WHITE = (255, 255, 255)
MUTED = (247, 244, 234, 180)

STATUS = 52
TAB = 118
CONTENT_TOP = STATUS + 8
CONTENT_BOT = H - TAB


def font(size: int, bold: bool = False) -> ImageFont.FreeTypeFont:
    candidates = [
        "/System/Library/Fonts/Supplemental/Arial Unicode.ttf",
        "/Library/Fonts/Arial Unicode.ttf",
        "/System/Library/Fonts/Supplemental/Georgia.ttf",
        "/System/Library/Fonts/Supplemental/Times New Roman.ttf",
    ]
    for path in candidates:
        if Path(path).exists():
            return ImageFont.truetype(path, size)
    return ImageFont.load_default()


def wrap(draw: ImageDraw.ImageDraw, text: str, fnt: ImageFont.FreeTypeFont, max_w: int) -> list[str]:
    words = text.split()
    lines: list[str] = []
    cur = ""
    for word in words:
        trial = f"{cur} {word}".strip()
        if draw.textlength(trial, font=fnt) <= max_w:
            cur = trial
        else:
            if cur:
                lines.append(cur)
            cur = word
    if cur:
        lines.append(cur)
    return lines or [text]


def rounded_image(src: Image.Image, size: tuple[int, int], radius: int) -> Image.Image:
    img = src.convert("RGBA").resize(size, Image.Resampling.LANCZOS)
    mask = Image.new("L", size, 0)
    ImageDraw.Draw(mask).rounded_rectangle((0, 0, size[0], size[1]), radius=radius, fill=255)
    out = Image.new("RGBA", size, (0, 0, 0, 0))
    out.paste(img, (0, 0))
    out.putalpha(mask)
    return out


def card(name: str) -> Image.Image:
    return Image.open(CARDS / name).convert("RGB")


def canvas() -> Image.Image:
    return Image.new("RGB", (W, H), BG)


def cover_fit(src: Image.Image, size: tuple[int, int]) -> Image.Image:
    img = src.convert("RGB")
    tw, th = size
    w, h = img.size
    scale = max(tw / w, th / h)
    nw, nh = max(1, int(w * scale)), max(1, int(h * scale))
    img = img.resize((nw, nh), Image.Resampling.LANCZOS)
    x = max(0, (nw - tw) // 2)
    y = max(0, (nh - th) // 2)
    return img.crop((x, y, x + tw, y + th))


def tile_rgba(src: Image.Image, size: tuple[int, int], opacity: int) -> Image.Image:
    tile = src.convert("RGBA")
    tw, th = tile.size
    out = Image.new("RGBA", size, (0, 0, 0, 0))
    for y in range(0, size[1], th):
        for x in range(0, size[0], tw):
            out.paste(tile, (x, y))
    r, g, b, a = out.split()
    a = a.point(lambda v: int(v * opacity / 255) if v else 0)
    out.putalpha(a)
    return out


def tiled_print(src: Image.Image, size: tuple[int, int], tile_size: int = 300) -> Image.Image:
    tile = src.convert("RGB").resize((tile_size, tile_size), Image.Resampling.LANCZOS)
    out = Image.new("RGB", size)
    for y in range(0, size[1], tile_size):
        for x in range(0, size[0], tile_size):
            out.paste(tile, (x, y))
    return out


def themed_canvas() -> Image.Image:
    bg = Image.new("RGBA", (W, H), (4, 3, 7, 255))
    pattern = Image.open(ROOT / "assets" / "pattern2.png")
    print_layer = tiled_print(pattern, (W, H), 300).convert("RGBA")
    r, g, b, a = print_layer.split()
    a = a.point(lambda _: 64)
    print_layer.putalpha(a)
    return Image.alpha_composite(bg, print_layer).convert("RGB")


def status_bar(base: Image.Image) -> None:
    d = ImageDraw.Draw(base)
    d.rectangle((0, 0, W, STATUS), fill=BG)
    f = font(28, True)
    d.text((36, 12), "16:24", font=f, fill=CREAM)
    d.text((W - 210, 12), "5G   84%", font=f, fill=CREAM)


def tab_bar(base: Image.Image, active: str, *, translucent: bool = False) -> None:
    d = ImageDraw.Draw(base)
    y0 = H - TAB
    if translucent:
        layer = base.convert("RGBA")
        overlay = Image.new("RGBA", (W, H), (0, 0, 0, 0))
        ImageDraw.Draw(overlay).rectangle((0, y0, W, H), fill=(8, 7, 15, 158))
        painted = Image.alpha_composite(layer, overlay)
        base.paste(painted.convert("RGB"))
        d = ImageDraw.Draw(base)
    else:
        d.rectangle((0, y0, W, H), fill=(8, 7, 15))
    d.line((0, y0, W, y0), fill=(40, 36, 58), width=2)
    items = [("home", "Главная"), ("decks", "Колоды"), ("history", "История"), ("settings", "Настройки")]
    slot = W // 4
    f = font(22)
    for i, (key, label) in enumerate(items):
        cx = slot * i + slot // 2
        color = GOLD if key == active else (160, 155, 145)
        # simple glyph
        gy = y0 + 22
        if key == "home":
            d.polygon([(cx, gy), (cx - 16, gy + 18), (cx + 16, gy + 18)], outline=color)
        elif key == "decks":
            d.rounded_rectangle((cx - 14, gy, cx + 14, gy + 22), 3, outline=color, width=2)
        elif key == "history":
            d.ellipse((cx - 12, gy, cx + 12, gy + 24), outline=color, width=2)
        else:
            d.ellipse((cx - 12, gy + 2, cx + 12, gy + 22), outline=color, width=2)
        tw = d.textlength(label, font=f)
        d.text((cx - tw / 2, y0 + 54), label, font=f, fill=color)


def header(base: Image.Image, title: str, right: str | None = None) -> int:
    d = ImageDraw.Draw(base)
    y = CONTENT_TOP
    f_title = font(34, True)
    d.text((48, y + 8), "‹", font=font(44), fill=GOLD)
    d.text((100, y + 16), title, font=f_title, fill=CREAM)
    if right:
        fr = font(28)
        tw = d.textlength(right, font=fr)
        d.text((W - 40 - tw, y + 22), right, font=fr, fill=PURPLE)
    return y + 80


def save(img: Image.Image, name: str) -> None:
    OUT.mkdir(parents=True, exist_ok=True)
    DESKTOP.mkdir(parents=True, exist_ok=True)
    path = OUT / name
    img.save(path, "JPEG", quality=90, optimize=True)
    img.save(DESKTOP / name, "JPEG", quality=90, optimize=True)
    print(path, round(path.stat().st_size / 1024), "KB")


def shot_home() -> None:
    img = canvas()
    status_bar(img)
    layer = img.convert("RGBA")
    d = ImageDraw.Draw(layer)
    pad = 40
    y = CONTENT_TOP + 12

    hero_h = 460
    hero = rounded_image(Image.open(ROOT / "assets" / "home-hero.jpg"), (W - pad * 2, hero_h), 48)
    overlay = Image.new("RGBA", hero.size, (8, 7, 15, 95))
    hero = Image.alpha_composite(hero, overlay)
    layer.alpha_composite(hero, (pad, y))
    hd = ImageDraw.Draw(layer)
    hd.text((pad + 40, y + 36), "Добрый день", font=font(30), fill=CREAM)
    title = "Вытяните карту для своего пути сегодня"
    tf = font(46, True)
    ty = y + 90
    for line in wrap(hd, title, tf, W - pad * 2 - 80):
        hd.text((pad + 40, ty), line, font=tf, fill=GOLD)
        ty += 56
    subf = font(28)
    for line in wrap(hd, "Обращайтесь к символам, когда нужен ясный ориентир.", subf, W - pad * 2 - 80):
        hd.text((pad + 40, ty + 8), line, font=subf, fill=CREAM)
        ty += 36
    btn = (pad + 40, y + hero_h - 90, pad + 280, y + hero_h - 30)
    hd.rounded_rectangle(btn, 28, fill=PURPLE)
    hd.text((pad + 78, y + hero_h - 76), "Карта дня", font=font(30, True), fill=WHITE)

    y = y + hero_h + 48
    hd.text((pad, y), "Быстрые расклады", font=font(36, True), fill=CREAM)
    more = "Все расклады"
    mw = hd.textlength(more, font=font(28))
    hd.text((W - pad - mw, y + 6), more, font=font(28), fill=PURPLE)
    y += 64
    cards = [("Одна карта", "1 карта"), ("Три карты", "3 карты"), ("Кельтский крест", "10 карт")]
    gap = 16
    cw = (W - pad * 2 - gap) // 2
    ch = 150
    for i, (name, meta) in enumerate(cards[:2]):
        x = pad + i * (cw + gap)
        hd.rounded_rectangle((x, y, x + cw, y + ch), 28, fill=(20, 16, 40, 230), outline=(108, 92, 231, 90), width=2)
        hd.text((x + 28, y + 32), name, font=font(30, True), fill=GOLD)
        hd.text((x + 28, y + 80), meta, font=font(26), fill=CREAM)
    y += ch + 20
    hd.rounded_rectangle((pad, y, W - pad, y + 150), 28, fill=(20, 16, 40, 230), outline=(108, 92, 231, 90), width=2)
    hd.text((pad + 28, y + 32), cards[2][0], font=font(30, True), fill=GOLD)
    hd.text((pad + 28, y + 80), cards[2][1], font=font(26), fill=CREAM)

    y += 190
    hd.text((pad, y), "Исследуйте колоду", font=font(36, True), fill=CREAM)
    y += 56
    hd.rounded_rectangle((pad, y, W - pad, y + 200), 28, fill=(20, 16, 40, 230), outline=(108, 92, 231, 90), width=2)
    body = "Изучайте карты, фиксируйте архетипы и замечайте повторяющиеся паттерны."
    by = y + 28
    for line in wrap(hd, body, font(28), W - pad * 2 - 56):
        hd.text((pad + 28, by), line, font=font(28), fill=CREAM)
        by += 38
    hd.rounded_rectangle((pad + 28, y + 130, pad + 320, y + 178), 24, fill=GOLD)
    hd.text((pad + 56, y + 140), "Открыть колоду", font=font(26, True), fill=(8, 7, 15))

    out = Image.new("RGB", (W, H), BG)
    out.paste(layer.convert("RGB"), (0, 0))
    tab_bar(out, "home")
    save(out, "01-home.jpg")


def shot_catalog() -> None:
    img = canvas()
    status_bar(img)
    layer = img.convert("RGBA")
    d = ImageDraw.Draw(layer)
    d.text((48, CONTENT_TOP + 8), "‹", font=font(44), fill=GOLD)
    d.text((100, CONTENT_TOP + 16), "Расклады", font=font(34, True), fill=CREAM)
    y = CONTENT_TOP + 88
    pad = 40

    def section(title: str) -> None:
        nonlocal y
        d.text((pad, y), title, font=font(32, True), fill=GOLD)
        y += 52

    def item(name: str, desc: str, meta: str, premium: bool = False) -> None:
        nonlocal y
        h = 200 if premium else 188
        d.rounded_rectangle(
            (pad, y, W - pad, y + h),
            28,
            fill=(18, 15, 32, 240),
            outline=(244, 211, 134, 70) if premium else (108, 92, 231, 70),
            width=2,
        )
        d.text((pad + 28, y + 24), name, font=font(32, True), fill=GOLD)
        if premium:
            bw = d.textlength("Премиум", font=font(22)) + 28
            d.rounded_rectangle((W - pad - 28 - bw, y + 26, W - pad - 28, y + 62), 14, fill=GOLD)
            d.text((W - pad - 14 - bw + 8, y + 32), "Премиум", font=font(22), fill=(8, 7, 15))
        dy = y + 72
        for line in wrap(d, desc, font(26), W - pad * 2 - 56)[:2]:
            d.text((pad + 28, dy), line, font=font(26), fill=CREAM)
            dy += 34
        d.text((pad + 28, y + h - 48), meta, font=font(24), fill=(180, 175, 165))
        y += h + 20

    section("Базовые")
    item("Одна карта", "Короткий акцент на текущей ситуации и внутреннем состоянии.", "1 карта  ·  Базовые")
    item("Три карты", "Классический расклад, раскрывающий прошлое, настоящее и будущее.", "3 карты  ·  Базовые")
    section("Любовь")
    item("Любовный расклад", "Семь карт, исследующих динамику отношений и эмоциональную связь.", "7 карт  ·  Любовь")
    section("Духовные")
    item("Кельтский крест", "Легендарный десятикарточный расклад для глубокого анализа.", "10 карт  ·  Духовные", True)

    out = layer.convert("RGB")
    tab_bar(out, "home")
    save(out, "02-spreads.jpg")


def shot_reading() -> None:
    img = canvas()
    status_bar(img)
    d = ImageDraw.Draw(img)
    d.text((48, CONTENT_TOP + 8), "‹", font=font(44), fill=GOLD)
    d.text((100, CONTENT_TOP + 10), "Три карты", font=font(36, True), fill=CREAM)
    d.text((100, CONTENT_TOP + 56), "3 карты", font=font(24), fill=(180, 175, 165))
    redraw = "Перетасовать"
    tw = d.textlength(redraw, font=font(28))
    d.text((W - 40 - tw, CONTENT_TOP + 22), redraw, font=font(28), fill=PURPLE)

    canvas_y = CONTENT_TOP + 100
    canvas_h = 620
    d.rounded_rectangle((32, canvas_y, W - 32, canvas_y + canvas_h), 28, fill=(14, 12, 24), outline=(60, 50, 90), width=2)

    faces = [
        ("the_moon.jpeg", "Прошлое"),
        ("the_star.jpeg", "Настоящее"),
        ("the_sun.jpeg", "Будущее"),
    ]
    cw, ch = 250, 375
    gap = 28
    total = cw * 3 + gap * 2
    x0 = (W - total) // 2
    cy = canvas_y + 70
    layer = img.convert("RGBA")
    for i, (file, label) in enumerate(faces):
        x = x0 + i * (cw + gap)
        cimg = rounded_image(card(file), (cw, ch), 24)
        shadow = Image.new("RGBA", (cw + 16, ch + 16), (0, 0, 0, 0))
        ImageDraw.Draw(shadow).rounded_rectangle((8, 10, cw + 8, ch + 10), 24, fill=(0, 0, 0, 90))
        shadow = shadow.filter(ImageFilter.GaussianBlur(6))
        layer.alpha_composite(shadow, (x - 8, cy - 6))
        layer.alpha_composite(cimg, (x, cy))
        td = ImageDraw.Draw(layer)
        lw = td.textlength(label, font=font(24))
        td.text((x + (cw - lw) / 2, cy + ch + 16), label, font=font(24), fill=GOLD)
    img = layer.convert("RGB")
    d = ImageDraw.Draw(img)

    y = canvas_y + canvas_h + 28
    details = [
        ("Прошлое", "Луна"),
        ("Настоящее", "Звезда"),
        ("Будущее", "Солнце"),
    ]
    for title, name in details:
        d.text((48, y), title, font=font(28, True), fill=GOLD)
        d.text((48, y + 36), name, font=font(26), fill=CREAM)
        y += 88

    footer_y = CONTENT_BOT - 140
    d.rounded_rectangle((40, footer_y, 500, footer_y + 88), 28, outline=PURPLE, width=2)
    d.text((70, footer_y + 26), "Пропустить анимации", font=font(26), fill=CREAM)
    d.rounded_rectangle((530, footer_y, W - 40, footer_y + 88), 28, fill=PURPLE)
    d.text((620, footer_y + 26), "Продолжить", font=font(30, True), fill=WHITE)

    tab_bar(img, "home")
    save(img, "03-reading.jpg")


def shot_interpretation() -> None:
    img = canvas()
    status_bar(img)
    d = ImageDraw.Draw(img)
    d.text((48, CONTENT_TOP + 8), "‹", font=font(44), fill=GOLD)
    d.text((100, CONTENT_TOP + 16), "Толкование", font=font(34, True), fill=CREAM)

    y = CONTENT_TOP + 88
    pad = 40
    d.rounded_rectangle((pad, y, W - pad, y + 980), 36, fill=(16, 13, 28), outline=(244, 211, 134, 50), width=2)
    d.text((pad + 32, y + 28), "Три карты", font=font(40, True), fill=GOLD)
    d.text((pad + 32, y + 84), "3 сентября 2026, 16:24", font=font(26), fill=(180, 175, 165))
    d.text((pad + 32, y + 140), "Сводка", font=font(30, True), fill=CREAM)
    summary = (
        "Ваш расклад «Три карты» раскрывает подсказки на 3 позициях. "
        "Луна говорит о скрытых чувствах, Звезда — о надежде, Солнце — о ясности впереди."
    )
    sy = y + 186
    for line in wrap(d, summary, font(28), W - pad * 2 - 64):
        d.text((pad + 32, sy), line, font=font(28), fill=CREAM)
        sy += 38

    chips = ["интуиция", "надежда", "ясность"]
    cx = pad + 32
    cy = sy + 16
    for chip in chips:
        tw = d.textlength(chip, font=font(24)) + 36
        d.rounded_rectangle((cx, cy, cx + tw, cy + 44), 20, fill=(108, 92, 231, 80))
        d.text((cx + 18, cy + 8), chip, font=font(24), fill=GOLD)
        cx += tw + 12

    # one card row
    ry = cy + 80
    d.text((pad + 32, ry), "Настоящее", font=font(30, True), fill=GOLD)
    d.text((pad + 32, ry + 42), "Энергия, влияющая на вас в данный момент.", font=font(24), fill=(180, 175, 165))
    cimg = rounded_image(card("the_star.jpeg"), (200, 300), 20)
    layer = img.convert("RGBA")
    layer.alpha_composite(cimg, (pad + 32, ry + 90))
    img = layer.convert("RGB")
    d = ImageDraw.Draw(img)
    d.text((pad + 260, ry + 110), "Звезда", font=font(32, True), fill=GOLD)
    meaning = "Надежда, вдохновение и ощущение, что путь освещается изнутри."
    my = ry + 160
    for line in wrap(d, meaning, font(26), W - pad - 280 - 40):
        d.text((pad + 260, my), line, font=font(26), fill=CREAM)
        my += 36

    by = y + 1010
    d.rounded_rectangle((pad, by, 500, by + 88), 28, outline=PURPLE, width=2)
    d.text((pad + 70, by + 28), "В избранное", font=font(28), fill=CREAM)
    d.rounded_rectangle((530, by, W - pad, by + 88), 28, fill=PURPLE)
    d.text((620, by + 26), "Поделиться", font=font(30, True), fill=WHITE)

    py = by + 120
    d.rounded_rectangle((pad, py, W - pad, py + 220), 28, fill=(28, 22, 52), outline=PURPLE, width=2)
    d.text((pad + 32, py + 28), "Премиум функция", font=font(32, True), fill=GOLD)
    msg = "Получите детальное толкование через ИИ с премиум подпиской"
    my = py + 80
    for line in wrap(d, msg, font(26), W - pad * 2 - 64):
        d.text((pad + 32, my), line, font=font(26), fill=CREAM)
        my += 36
    d.rounded_rectangle((pad + 32, py + 150, pad + 360, py + 198), 24, fill=GOLD)
    d.text((pad + 70, py + 160), "Разблокировать", font=font(26, True), fill=(8, 7, 15))

    tab_bar(img, "home")
    save(img, "04-interpretation.jpg")


def shot_decks() -> None:
    img = themed_canvas()
    d = ImageDraw.Draw(img)
    fstat = font(28, True)
    d.text((36, 12), "16:24", font=fstat, fill=CREAM)
    d.text((W - 210, 12), "5G   84%", font=fstat, fill=CREAM)
    y = CONTENT_TOP + 16
    pad = 32
    d.text((pad, y), "Арканы", font=font(30, True), fill=CREAM)
    y += 52
    chips = [("Все", True), ("Старшие", False), ("Младшие", False)]
    x = pad
    for label, active in chips:
        tw = d.textlength(label, font=font(26)) + 44
        if active:
            d.rounded_rectangle((x, y, x + tw, y + 52), 26, fill=PURPLE)
            d.text((x + 22, y + 12), label, font=font(26), fill=WHITE)
        else:
            d.rounded_rectangle((x, y, x + tw, y + 52), 26, fill=(10, 16, 32), outline=GOLD, width=2)
            d.text((x + 22, y + 12), label, font=font(26), fill=CREAM)
        x += tw + 12
    y += 80
    d.text((pad, y), "Масти", font=font(30, True), fill=CREAM)
    y += 52
    suits = ["Все", "Жезлы", "Кубки", "Мечи", "Пентакли"]
    x = pad
    for i, label in enumerate(suits):
        tw = d.textlength(label, font=font(24)) + 36
        if x + tw > W - pad:
            x = pad
            y += 60
        if i == 0:
            d.rounded_rectangle((x, y, x + tw, y + 48), 24, fill=PURPLE)
            d.text((x + 18, y + 10), label, font=font(24), fill=WHITE)
        else:
            d.rounded_rectangle((x, y, x + tw, y + 48), 24, fill=(10, 16, 32), outline=GOLD, width=2)
            d.text((x + 18, y + 10), label, font=font(24), fill=CREAM)
        x += tw + 10

    y += 80
    files = [
        ("the_fool.jpeg", "Дурак", "Новые начинания, спонтанность, свободный дух"),
        ("the_magician.jpeg", "Маг", "Манифестация, находчивость, сила"),
    ]
    gap = 20
    cw = (W - pad * 2 - gap) // 2
    card_w = cw - 24
    card_h = int(card_w * 3 / 2)
    text_h = 118
    layer = img.convert("RGBA")
    dd = ImageDraw.Draw(layer)
    for i, (file, name, desc) in enumerate(files):
        x = pad + i * (cw + gap)
        yy = y
        dd.rounded_rectangle(
            (x, yy, x + cw, yy + card_h + text_h),
            24,
            fill=(18, 15, 32, 208),
            outline=(212, 175, 55, 90),
            width=2,
        )
        cimg = rounded_image(card(file), (card_w, card_h), 16)
        layer.alpha_composite(cimg, (x + 12, yy + 12))
        dd.text((x + 16, yy + card_h + 24), name, font=font(26, True), fill=GOLD)
        ty = yy + card_h + 60
        for line in wrap(dd, desc, font(22), cw - 32)[:2]:
            dd.text((x + 16, ty), line, font=font(22), fill=CREAM)
            ty += 28
    img = layer.convert("RGB")

    tab_bar(img, "decks", translucent=True)
    save(img, "05-decks.jpg")


if __name__ == "__main__":
    shot_home()
    shot_catalog()
    shot_reading()
    shot_interpretation()
    shot_decks()
    print("done")
