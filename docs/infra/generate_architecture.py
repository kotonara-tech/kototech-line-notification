"""構成図 docs/infra/architecture.png を生成する。

    python3 docs/infra/generate_architecture.py

- Google Cloud のアイコンは、公式のもの(docs/infra/icons/)を使う。
- スプレッドシート・LINE・GitHub・人・スマートフォンは、公式ではなく、ここで簡略化して描く。
- 日本語フォントは macOS のヒラギノを使う(別の環境で動かすときは FONT_PATH を変える)。
- 構成を変えたら、このスクリプトの座標とラベルを直して、再生成する。
"""

import math
import os
import sys

from PIL import Image, ImageDraw, ImageFont

HERE = os.path.dirname(os.path.abspath(__file__))
OUT = sys.argv[1] if len(sys.argv) > 1 else os.path.join(HERE, "architecture.png")
FONT_PATH = "/System/Library/Fonts/ヒラギノ角ゴシック W3.ttc"
LATIN_FONT_PATH = "/System/Library/Fonts/Helvetica.ttc"

S = 2  # 描画の倍率(縮小して、輪郭をなめらかにする)
W, H = 2700, 2060
ICON = 120

LINE_COLOR = "#6b6b6b"
BORDER = "#b4b4b4"
TEXT = "#222222"

img = Image.new("RGB", (W * S, H * S), "white")
d = ImageDraw.Draw(img, "RGBA")


def font(size, path=FONT_PATH):
    return ImageFont.truetype(path, int(size * S))


def px(v):
    return int(round(v * S))


def text_size(text, f):
    box = d.textbbox((0, 0), text, font=f)
    return (box[2] - box[0]) / S, (box[3] - box[1]) / S


def draw_text_center(cx, y_top, lines, size, fill=TEXT, line_gap=6):
    f = font(size)
    y = y_top
    for line in lines:
        w, h = text_size(line, f)
        d.text((px(cx - w / 2), px(y)), line, font=f, fill=fill)
        y += size + line_gap
    return y


# ---------------------------------------------------------------- 枠
def box(x1, y1, x2, y2, label, fill=None, border=BORDER, radius=22):
    d.rounded_rectangle((px(x1), px(y1), px(x2), px(y2)), radius=px(radius), outline=border, width=px(2.5), fill=fill)
    f = font(23)
    w, h = text_size(label, f)
    tab = (px(x1 + 14), px(y1 - 18), px(x1 + 14 + w + 26), px(y1 + 20))
    d.rounded_rectangle(tab, radius=px(9), fill="#f1f1f1", outline=border, width=px(2))
    d.text((px(x1 + 27), px(y1 - 13)), label, font=f, fill=TEXT)


# ---------------------------------------------------------------- アイコン
def load_png(name):
    im = Image.open(os.path.join(HERE, "icons", name + ".png")).convert("RGBA")
    return im


def paste_icon(im, cx, cy, size=ICON):
    im = im.resize((px(size), px(size)), Image.LANCZOS)
    img.paste(im, (px(cx - size / 2), px(cy - size / 2)), im)


def icon_label(cx, cy, lines, size=ICON):
    draw_text_center(cx, cy + size / 2 + 10, lines, 21)


def draw_on_canvas(painter, canvas=512):
    """512x512 の透明なキャンバスに自前のアイコンを描いて返す。"""
    big = Image.new("RGBA", (canvas * 2, canvas * 2), (0, 0, 0, 0))
    painter(ImageDraw.Draw(big), 2)
    return big.resize((canvas, canvas), Image.LANCZOS)


def sheets_icon():
    def paint(g, k):
        green, dark, light = "#0F9D58", "#0B8043", "#87CEAC"
        # 紙(右上を折る)
        page = [(96 * k, 32 * k), (320 * k, 32 * k), (416 * k, 128 * k), (416 * k, 480 * k), (96 * k, 480 * k)]
        g.polygon(page, fill=green)
        g.polygon([(320 * k, 32 * k), (416 * k, 128 * k), (320 * k, 128 * k)], fill=light)
        # 表
        g.rectangle((152 * k, 232 * k, 360 * k, 408 * k), fill="white")
        for x in (221 * k, 291 * k):
            g.line((x, 232 * k, x, 408 * k), fill=green, width=7 * k)
        for y in (290 * k, 349 * k):
            g.line((152 * k, y, 360 * k, y), fill=green, width=7 * k)
        g.rectangle((152 * k, 232 * k, 360 * k, 408 * k), outline=dark, width=3 * k)

    return draw_on_canvas(paint)


def line_icon():
    def paint(g, k):
        g.rounded_rectangle((32 * k, 32 * k, 480 * k, 480 * k), radius=96 * k, fill="#06C755")
        # 吹き出し
        g.rounded_rectangle((92 * k, 120 * k, 420 * k, 336 * k), radius=108 * k, fill="white")
        g.polygon([(190 * k, 320 * k), (250 * k, 320 * k), (226 * k, 392 * k)], fill="white")
        f = ImageFont.truetype(LATIN_FONT_PATH, 96 * k)
        label = "LINE"
        box_ = g.textbbox((0, 0), label, font=f)
        g.text(((512 * k - (box_[2] - box_[0])) / 2 - box_[0], 228 * k - (box_[3] - box_[1]) / 2 - box_[1]),
               label, font=f, fill="#06C755")

    return draw_on_canvas(paint)


def github_icon():
    def paint(g, k):
        g.ellipse((32 * k, 32 * k, 480 * k, 480 * k), fill="#24292f")
        # ネコの頭(簡略化)
        g.polygon([(150 * k, 130 * k), (215 * k, 186 * k), (150 * k, 232 * k)], fill="white")
        g.polygon([(362 * k, 130 * k), (297 * k, 186 * k), (362 * k, 232 * k)], fill="white")
        g.ellipse((132 * k, 156 * k, 380 * k, 384 * k), fill="white")
        for cx in (206, 306):
            g.ellipse(((cx - 17) * k, 252 * k, (cx + 17) * k, 296 * k), fill="#24292f")
        g.arc((216 * k, 290 * k, 296 * k, 346 * k), 20, 160, fill="#24292f", width=8 * k)

    return draw_on_canvas(paint)


def person_icon():
    def paint(g, k):
        c = "#232f3e"
        g.ellipse((176 * k, 56 * k, 336 * k, 216 * k), outline=c, width=22 * k)
        g.arc((72 * k, 250 * k, 440 * k, 620 * k), 180, 360, fill=c, width=22 * k)
        g.line((72 * k + 4 * k, 435 * k, 440 * k - 4 * k, 435 * k), fill=c, width=22 * k)

    return draw_on_canvas(paint)


def phone_icon():
    def paint(g, k):
        c = "#6b6b6b"
        g.rounded_rectangle((136 * k, 40 * k, 376 * k, 472 * k), radius=40 * k, outline=c, width=20 * k)
        g.ellipse((246 * k, 408 * k, 266 * k, 428 * k), fill=c)

    return draw_on_canvas(paint)


# ---------------------------------------------------------------- 矢印
def rounded_path(points, r=22):
    """折れ線の角を丸めた点列を返す。"""
    if len(points) < 3:
        return list(points)
    out = [points[0]]
    for i in range(1, len(points) - 1):
        p0, p1, p2 = points[i - 1], points[i], points[i + 1]
        v1 = (p0[0] - p1[0], p0[1] - p1[1])
        v2 = (p2[0] - p1[0], p2[1] - p1[1])
        l1, l2 = math.hypot(*v1), math.hypot(*v2)
        rr = min(r, l1 / 2, l2 / 2)
        a = (p1[0] + v1[0] / l1 * rr, p1[1] + v1[1] / l1 * rr)
        b = (p1[0] + v2[0] / l2 * rr, p1[1] + v2[1] / l2 * rr)
        for t in [j / 10 for j in range(11)]:
            x = (1 - t) ** 2 * a[0] + 2 * (1 - t) * t * p1[0] + t**2 * b[0]
            y = (1 - t) ** 2 * a[1] + 2 * (1 - t) * t * p1[1] + t**2 * b[1]
            out.append((x, y))
    out.append(points[-1])
    return out


def arrowhead(tip, frm, size=17, spread=0.55):
    ang = math.atan2(tip[1] - frm[1], tip[0] - frm[0])
    for sgn in (-1, 1):
        a = ang + math.pi + sgn * spread
        end = (tip[0] + size * math.cos(a), tip[1] + size * math.sin(a))
        d.line((px(tip[0]), px(tip[1]), px(end[0]), px(end[1])), fill=LINE_COLOR, width=px(3.2))


def arrow(points, label=None, at=None, both=False, size=21):
    path = rounded_path(points)
    d.line([(px(x), px(y)) for x, y in path], fill=LINE_COLOR, width=px(3.2), joint="curve")
    arrowhead(points[-1], path[-2])
    if both:
        arrowhead(points[0], path[1])
    if label:
        lines = label.split("\n")
        f = font(size)
        w = max(text_size(t, f)[0] for t in lines)
        h = len(lines) * (size + 5)
        cx, cy = at
        d.rectangle((px(cx - w / 2 - 8), px(cy - h / 2 - 2), px(cx + w / 2 + 8), px(cy + h / 2 + 2)), fill="white")
        y = cy - h / 2
        for t in lines:
            tw, _ = text_size(t, f)
            d.text((px(cx - tw / 2), px(y)), t, font=f, fill="#444444")
            y += size + 5


# ================================================================ 配置
# --- 枠
box(1080, 70, 1720, 345, "Google Workspace")
box(360, 410, 2140, 1990, "Google Cloud")
box(440, 480, 2060, 1470, "プロジェクト: kototech-line-dev / kototech-line-prod(同じ構成)  asia-northeast1")
box(440, 1580, 1100, 1930, "プロジェクト: kototech-line-ops")
box(2260, 480, 2650, 1120, "LINE")

# --- 位置(中心)
OPERATOR = (180, 200)
SHEETS = (1400, 190)
SCHEDULER = (900, 650)
JOB = (1400, 650)
SECRET = (1400, 1000)
REGISTRY = (900, 1000)
DEPLOYER = (900, 1294)
WIF = (580, 1294)
GITHUB = (180, 1294)
STATE = (900, 1760)
LINE_API = (2455, 650)
PHONE = (2455, 960)

# --- アイコン
paste_icon(person_icon(), *OPERATOR, size=100)
icon_label(*OPERATOR, ["運用者"], size=100)

paste_icon(sheets_icon(), *SHEETS)
icon_label(*SHEETS, ["Google スプレッドシート", "工程表"])

paste_icon(load_png("cloud_scheduler"), *SCHEDULER)
icon_label(*SCHEDULER, ["Cloud Scheduler", "5分ごと・Scheduler用SA"])

paste_icon(load_png("cloud_run"), *JOB)
icon_label(*JOB, ["Cloud Run Job (Go)", "ROLE=scan・実行用SA"])

paste_icon(load_png("secret_manager"), *SECRET)
icon_label(*SECRET, ["Secret Manager", "LINEチャネルシークレット"])

paste_icon(load_png("artifact_registry"), *REGISTRY)
icon_label(*REGISTRY, ["Artifact Registry", "コンテナイメージ"])

paste_icon(load_png("identity_and_access_management"), *DEPLOYER)
icon_label(*DEPLOYER, ["デプロイ用SA", "github-deployer"])

paste_icon(load_png("workload_identity_pool"), *WIF)
icon_label(*WIF, ["Workload Identity", "Federation"])

paste_icon(github_icon(), *GITHUB)
icon_label(*GITHUB, ["GitHub Actions", "CI/CD・Terraform"])

paste_icon(load_png("cloud_storage"), *STATE)
icon_label(*STATE, ["Cloud Storage", "Terraform state"])

paste_icon(line_icon(), *LINE_API)
icon_label(*LINE_API, ["LINE Messaging API", "broadcast(全友だち)"])

paste_icon(phone_icon(), *PHONE, size=100)
icon_label(*PHONE, ["友だち(LINE)"], size=100)

half = ICON / 2
# アイコンの下のラベル(2 行)の下端。矢印が、ラベルの文字を貫通しないよう、この位置で止める
LABEL_BOTTOM = half + 10 + 2 * 27 + 10

# --- 実行時の流れ
arrow([(OPERATOR[0] + 70, OPERATOR[1]), (1080, OPERATOR[1]), (SHEETS[0] - half - 8, OPERATOR[1])],
      "工程表を編集(共有設定で権限管理)", (700, OPERATOR[1]))

arrow([(SCHEDULER[0] + half + 6, JOB[1]), (JOB[0] - half - 8, JOB[1])],
      "5分ごと起動\nJobs API :run\n(OAuth)", ((SCHEDULER[0] + JOB[0]) / 2, JOB[1]))

arrow([(JOB[0], JOB[1] - half - 6), (SHEETS[0], 318)],
      "読み取り / 通知済みを書き込み\n(Sheets API)", (JOB[0], 440), both=True)

arrow([(SECRET[0], SECRET[1] - half - 6), (JOB[0], JOB[1] + LABEL_BOTTOM)],
      "チャネルシークレットを\n環境変数として注入", (JOB[0], (SECRET[1] - half + JOB[1] + LABEL_BOTTOM) / 2))

arrow([(JOB[0] + half + 6, JOB[1] - 22), (LINE_API[0] - half - 8, JOB[1] - 22)],
      "① アクセストークンを発行(ステートレス)\n② broadcast を送信", (1930, JOB[1] - 22))

arrow([(LINE_API[0], LINE_API[1] + LABEL_BOTTOM + 6), (PHONE[0], PHONE[1] - 56)],
      "通知", (LINE_API[0], (LINE_API[1] + LABEL_BOTTOM + PHONE[1] - 56) / 2))

# --- 配備の流れ
arrow([(GITHUB[0] + half + 6, GITHUB[1]), (WIF[0] - half - 8, GITHUB[1])],
      "OIDC トークン", ((GITHUB[0] + WIF[0]) / 2 + 4, GITHUB[1] - 36))

arrow([(WIF[0] + half + 6, WIF[1]), (DEPLOYER[0] - half - 8, DEPLOYER[1])],
      "なりすまし", ((WIF[0] + DEPLOYER[0]) / 2, WIF[1] - 36))

arrow([(DEPLOYER[0], DEPLOYER[1] - half - 6), (REGISTRY[0], REGISTRY[1] + LABEL_BOTTOM)],
      "イメージを push", (DEPLOYER[0], (DEPLOYER[1] - half + REGISTRY[1] + LABEL_BOTTOM) / 2))

arrow([(DEPLOYER[0] + half + 6, DEPLOYER[1]), (1700, DEPLOYER[1]), (1700, JOB[1] + 42), (JOB[0] + half + 8, JOB[1] + 42)],
      "イメージを更新\n(gcloud run jobs update)", (1700, 1000))

arrow([(REGISTRY[0] + half + 6, REGISTRY[1] - 10), (1180, REGISTRY[1] - 10), (1180, JOB[1] + 42), (JOB[0] - half - 8, JOB[1] + 42)],
      "イメージを取得", (1180, 830))

arrow([(DEPLOYER[0], DEPLOYER[1] + LABEL_BOTTOM), (STATE[0], STATE[1] - half - 6)],
      "state を読み書き", (STATE[0], 1525))

out = img.resize((W, H), Image.LANCZOS)
out.save(OUT, optimize=True)
print("saved:", OUT, out.size)
