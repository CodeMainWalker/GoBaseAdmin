// Package captcha 生成图形验证码，行为与上游 SaiAdmin 6.x 行为一致。
//
// 契约：
//   - 4 个字符，字符集剔除易混淆的 i/l/o/0/1
//   - 120x36 PNG，背景 rgb(242,243,245)
//   - 存储值转小写；校验时**先删后比**（一次性，错误猜测即作废）
//   - 缓存键为 uuid4 字符串本身，无前缀，TTL 300s
//   - 返回 data:image/png;base64,... 数据 URI
package captcha

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"strings"

	"github.com/google/uuid"
)

// charset 验证码字符集（与上游 SaiAdmin 6.x 的字符集一致）
const charset = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ"

const (
	width      = 120
	height     = 36
	codeLength = 4
	// Expire 验证码有效期（秒）
	Expire = 300
)

// Generate 生成验证码，返回 (uuid, 小写代码, PNG 字节)
func Generate() (string, string, []byte) {
	code := randomCode(codeLength)
	img := draw(code)
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return uuid.NewString(), strings.ToLower(code), buf.Bytes()
}

func randomCode(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}

// draw 绘制验证码图像。
// 不引入第三方字体依赖：用规则的点阵笔画绘制每个字符，
// 足以满足“可识别 + 有干扰”的需求。
func draw(code string) image.Image {
	bg := color.RGBA{R: 242, G: 243, B: 245, A: 255}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, bg)
		}
	}

	// 干扰线
	lineColor := color.RGBA{R: 200, G: 205, B: 215, A: 255}
	for i := 0; i < 4; i++ {
		x1, y1 := rand.Intn(width), rand.Intn(height)
		x2, y2 := rand.Intn(width), rand.Intn(height)
		drawLine(img, x1, y1, x2, y2, lineColor)
	}

	// 逐字符绘制（等宽分布 + 轻微随机抖动）
	cellW := width / (len(code) + 1)
	for i, ch := range code {
		cx := cellW*(i+1) - 6 + rand.Intn(5)
		cy := height/2 - 9 + rand.Intn(5)
		drawChar(img, ch, cx, cy, randColor())
	}
	return img
}

func randColor() color.RGBA {
	// 深色系，保证对比度
	return color.RGBA{
		R: uint8(30 + rand.Intn(90)),
		G: uint8(30 + rand.Intn(90)),
		B: uint8(30 + rand.Intn(90)),
		A: 255,
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx, dy := abs(x2-x1), abs(y2-y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy
	for {
		if x1 >= 0 && x1 < width && y1 >= 0 && y1 < height {
			img.Set(x1, y1, c)
		}
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// ---- 5x7 点阵字形 ----
// 每个字形 7 行，每行 5 位（bit4..bit0 从左到右）
var glyphs = map[rune][7]uint8{
	'a': {0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'b': {0b11110, 0b10001, 0b10001, 0b11110, 0b10001, 0b10001, 0b11110},
	'c': {0b01110, 0b10001, 0b10000, 0b10000, 0b10000, 0b10001, 0b01110},
	'd': {0b11110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b11110},
	'e': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111},
	'f': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b10000},
	'g': {0b01110, 0b10001, 0b10000, 0b10111, 0b10001, 0b10001, 0b01111},
	'h': {0b10001, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'j': {0b00111, 0b00010, 0b00010, 0b00010, 0b00010, 0b10010, 0b01100},
	'k': {0b10001, 0b10010, 0b10100, 0b11000, 0b10100, 0b10010, 0b10001},
	'm': {0b10001, 0b11011, 0b10101, 0b10101, 0b10001, 0b10001, 0b10001},
	'n': {0b10001, 0b11001, 0b10101, 0b10011, 0b10001, 0b10001, 0b10001},
	'p': {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	'q': {0b01110, 0b10001, 0b10001, 0b10001, 0b10101, 0b10010, 0b01101},
	'r': {0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001},
	's': {0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110},
	't': {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100},
	'u': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'v': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01010, 0b00100},
	'w': {0b10001, 0b10001, 0b10001, 0b10101, 0b10101, 0b11011, 0b10001},
	'x': {0b10001, 0b10001, 0b01010, 0b00100, 0b01010, 0b10001, 0b10001},
	'y': {0b10001, 0b10001, 0b01010, 0b00100, 0b00100, 0b00100, 0b00100},
	'z': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b10000, 0b11111},
	'A': {0b01110, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'B': {0b11110, 0b10001, 0b10001, 0b11110, 0b10001, 0b10001, 0b11110},
	'C': {0b01110, 0b10001, 0b10000, 0b10000, 0b10000, 0b10001, 0b01110},
	'D': {0b11110, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b11110},
	'E': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b11111},
	'F': {0b11111, 0b10000, 0b10000, 0b11110, 0b10000, 0b10000, 0b10000},
	'G': {0b01110, 0b10001, 0b10000, 0b10111, 0b10001, 0b10001, 0b01111},
	'H': {0b10001, 0b10001, 0b10001, 0b11111, 0b10001, 0b10001, 0b10001},
	'J': {0b00111, 0b00010, 0b00010, 0b00010, 0b00010, 0b10010, 0b01100},
	'K': {0b10001, 0b10010, 0b10100, 0b11000, 0b10100, 0b10010, 0b10001},
	'L': {0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b10000, 0b11111},
	'M': {0b10001, 0b11011, 0b10101, 0b10101, 0b10001, 0b10001, 0b10001},
	'N': {0b10001, 0b11001, 0b10101, 0b10011, 0b10001, 0b10001, 0b10001},
	'P': {0b11110, 0b10001, 0b10001, 0b11110, 0b10000, 0b10000, 0b10000},
	'Q': {0b01110, 0b10001, 0b10001, 0b10001, 0b10101, 0b10010, 0b01101},
	'R': {0b11110, 0b10001, 0b10001, 0b11110, 0b10100, 0b10010, 0b10001},
	'S': {0b01111, 0b10000, 0b10000, 0b01110, 0b00001, 0b00001, 0b11110},
	'T': {0b11111, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100, 0b00100},
	'U': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01110},
	'V': {0b10001, 0b10001, 0b10001, 0b10001, 0b10001, 0b01010, 0b00100},
	'W': {0b10001, 0b10001, 0b10001, 0b10101, 0b10101, 0b11011, 0b10001},
	'X': {0b10001, 0b10001, 0b01010, 0b00100, 0b01010, 0b10001, 0b10001},
	'Y': {0b10001, 0b10001, 0b01010, 0b00100, 0b00100, 0b00100, 0b00100},
	'Z': {0b11111, 0b00001, 0b00010, 0b00100, 0b01000, 0b10000, 0b11111},
}

func drawChar(img *image.RGBA, ch rune, ox, oy int, c color.RGBA) {
	g, ok := glyphs[ch]
	if !ok {
		g = glyphs['a']
	}
	for row := 0; row < 7; row++ {
		bits := g[row]
		for col := 0; col < 5; col++ {
			if bits&(1<<uint(4-col)) != 0 {
				// 放大 2 倍，视觉更饱满
				for dy := 0; dy < 2; dy++ {
					for dx := 0; dx < 2; dx++ {
						x, y := ox+col*2+dx, oy+row*2+dy
						if x >= 0 && x < width && y >= 0 && y < height {
							img.Set(x, y, c)
						}
					}
				}
			}
		}
	}
}
