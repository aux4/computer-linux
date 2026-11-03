package main

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

	xdraw "golang.org/x/image/draw"
)

const tesseractInstall = "Install Tesseract: sudo apt install tesseract-ocr (Debian/Ubuntu), sudo dnf install tesseract (Fedora), sudo apk add tesseract-ocr (Alpine), brew install tesseract"

type ocrWord struct {
	text string
	conf float64
	r    rect // in source image pixels
}

type ocrLine struct {
	words []ocrWord
	r     rect
	text  string
	conf  float64
}

// bcp47 -> tesseract language codes
var tessLangs = map[string]string{
	"en": "eng", "pt": "por", "es": "spa", "fr": "fra", "de": "deu", "it": "ita", "nl": "nld", "ja": "jpn",
	"ko": "kor", "zh-hans": "chi_sim", "zh-hant": "chi_tra", "zh": "chi_sim", "ru": "rus", "uk": "ukr", "pl": "pol",
	"sv": "swe", "da": "dan", "nb": "nor", "no": "nor", "fi": "fin", "tr": "tur", "cs": "ces", "el": "ell", "he": "heb",
	"ar": "ara", "hi": "hin", "vi": "vie", "th": "tha", "id": "ind", "ro": "ron", "hu": "hun",
}

func tessLanguages(list string) (string, *cmdError) {
	if strings.TrimSpace(list) == "" {
		return "eng", nil
	}
	out, err := exec.Command("tesseract", "--list-langs").CombinedOutput()
	avail := map[string]bool{}
	if err == nil {
		for _, l := range strings.Split(string(out), "\n") {
			l = strings.TrimSpace(l)
			if l != "" && !strings.Contains(l, " ") {
				avail[l] = true
			}
		}
	}
	var codes []string
	for _, l := range strings.Split(list, ",") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		code := l
		ll := strings.ToLower(l)
		if c, ok := tessLangs[ll]; ok {
			code = c
		} else if c, ok := tessLangs[strings.SplitN(ll, "-", 2)[0]]; ok && strings.Contains(ll, "-") && !strings.HasPrefix(ll, "zh") {
			code = c
		}
		if len(avail) > 0 && !avail[code] {
			var have []string
			for k := range avail {
				if k != "osd" {
					have = append(have, k)
				}
			}
			sort.Strings(have)
			return "", errObj(obj{"error": fmt.Sprintf("Unsupported --languages '%s' (Tesseract language '%s' is not installed). Supported: %s. Install more with your package manager, e.g. sudo apt install tesseract-ocr-%s", l, code, strings.Join(have, ", "), code), "supported": have})
		}
		codes = append(codes, code)
	}
	return strings.Join(codes, "+"), nil
}

// prepare upscales, converts to grayscale and inverts dark backgrounds: Tesseract reads dark text
// on a light background best.
func prepare(src image.Image, factor float64) *image.Gray {
	b := src.Bounds()
	w, h := int(float64(b.Dx())*factor+0.5), int(float64(b.Dy())*factor+0.5)
	scaled := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(scaled, scaled.Bounds(), src, b, xdraw.Src, nil)
	g := image.NewGray(scaled.Bounds())
	var sum uint64
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.GrayModel.Convert(scaled.At(x, y)).(color.Gray)
			g.SetGray(x, y, c)
			sum += uint64(c.Y)
		}
	}
	if w*h > 0 && sum/uint64(w*h) < 110 {
		for i := range g.Pix {
			g.Pix[i] = 255 - g.Pix[i]
		}
	}
	return g
}

// runOCR recognizes the image; positions are returned in source image pixels.
func runOCR(ctx context.Context, src image.Image, level, langs string) ([]ocrLine, *cmdError) {
	if tesseractPath() == "" {
		return nil, errObj(obj{"error": "Tesseract is not installed, so text can't be recognized. " + tesseractInstall, "dependency": "tesseract"})
	}
	factor := 3.0
	if level == "fast" {
		factor = 1.5
	}
	// keep very large captures within a sane size
	if px := float64(src.Bounds().Dx()) * factor; px > 6000 {
		factor = 6000 / float64(src.Bounds().Dx())
		if factor < 1 {
			factor = 1
		}
	}
	g := prepare(src, factor)
	f, err := os.CreateTemp("", "aux4-computer-ocr-*.png")
	if err != nil {
		return nil, errf("Cannot create a temp file: %v", err)
	}
	defer os.Remove(f.Name())
	if err := png.Encode(f, g); err != nil {
		f.Close()
		return nil, errf("Cannot write the OCR image: %v", err)
	}
	f.Close()
	psm := "11" // sparse text: UI labels scattered across a window
	cmd := exec.CommandContext(ctx, "tesseract", f.Name(), "stdout", "-l", langs, "--psm", psm, "tsv")
	var out, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &stderr
	cmd.Env = append(os.Environ(), "OMP_THREAD_LIMIT=1")
	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, &cmdError{fields: obj{"error": "deadline", "phase": "recognition"}}
		}
		return nil, errObj(obj{"error": "Tesseract failed: " + strings.TrimSpace(stderr.String()+" "+err.Error()), "phase": "recognition"})
	}
	var words []ocrWord
	for i, line := range strings.Split(out.String(), "\n") {
		if i == 0 {
			continue
		}
		c := strings.Split(line, "\t")
		if len(c) < 12 || c[0] != "5" {
			continue
		}
		text := strings.TrimSpace(c[11])
		if text == "" {
			continue
		}
		conf, _ := strconv.ParseFloat(c[10], 64)
		if conf < 0 {
			continue
		}
		l, _ := strconv.ParseFloat(c[6], 64)
		t, _ := strconv.ParseFloat(c[7], 64)
		w, _ := strconv.ParseFloat(c[8], 64)
		h, _ := strconv.ParseFloat(c[9], 64)
		words = append(words, ocrWord{text: text, conf: conf / 100, r: rect{l / factor, t / factor, w / factor, h / factor}})
	}
	return groupLines(words), nil
}

// groupLines merges words that sit on the same line and are close together.
func groupLines(words []ocrWord) []ocrLine {
	sort.SliceStable(words, func(i, j int) bool {
		ci := words[i].r.Y + words[i].r.H/2
		cj := words[j].r.Y + words[j].r.H/2
		if abs(ci-cj) > minf(words[i].r.H, words[j].r.H)/2 {
			return ci < cj
		}
		return words[i].r.X < words[j].r.X
	})
	var lines []ocrLine
	for _, w := range words {
		merged := false
		for li := range lines {
			l := &lines[li]
			last := l.words[len(l.words)-1]
			cy, ly := w.r.Y+w.r.H/2, last.r.Y+last.r.H/2
			h := maxf(w.r.H, last.r.H)
			gap := w.r.X - (last.r.X + last.r.W)
			if abs(cy-ly) < h*0.5 && gap > -h*0.3 && gap < h*1.2 {
				l.words = append(l.words, w)
				merged = true
				break
			}
		}
		if !merged {
			lines = append(lines, ocrLine{words: []ocrWord{w}})
		}
	}
	for i := range lines {
		l := &lines[i]
		var parts []string
		r := l.words[0].r
		var conf float64
		for _, w := range l.words {
			parts = append(parts, w.text)
			r = union(r, w.r)
			conf += w.conf
		}
		l.text = strings.Join(parts, " ")
		l.r = r
		l.conf = conf / float64(len(l.words))
	}
	sort.SliceStable(lines, func(i, j int) bool {
		if abs(lines[i].r.Y-lines[j].r.Y) > minf(lines[i].r.H, lines[j].r.H)/2 {
			return lines[i].r.Y < lines[j].r.Y
		}
		return lines[i].r.X < lines[j].r.X
	})
	return lines
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

func union(a, b rect) rect {
	x1, y1 := minf(a.X, b.X), minf(a.Y, b.Y)
	x2, y2 := maxf(a.X+a.W, b.X+b.W), maxf(a.Y+a.H, b.Y+b.H)
	return rect{x1, y1, x2 - x1, y2 - y1}
}

// matchSpan returns the frame of the smallest run of words whose loose text contains needle.
func matchSpan(l ocrLine, needle string) (rect, bool) {
	n := normalizeLoose(needle)
	if n == "" || !strings.Contains(normalizeLoose(l.text), n) {
		return rect{}, false
	}
	best, found := rect{}, false
	bestLen := 1 << 30
	for i := range l.words {
		s := ""
		r := l.words[i].r
		for j := i; j < len(l.words); j++ {
			s += normalizeLoose(l.words[j].text)
			r = union(r, l.words[j].r)
			if strings.Contains(s, n) {
				if j-i < bestLen {
					best, found, bestLen = r, true, j-i
				}
				break
			}
		}
	}
	if !found {
		return l.r, true
	}
	// narrow a single long word to the matched characters (proportionally)
	if bestLen == 0 {
		for _, w := range l.words {
			if w.r == best {
				wt := normalizeLoose(w.text)
				if idx := strings.Index(wt, n); idx >= 0 && len(wt) > len(n) {
					cw := w.r.W / float64(len([]rune(wt)))
					pre := len([]rune(wt[:idx]))
					best = rect{w.r.X + cw*float64(pre), w.r.Y, cw * float64(len([]rune(n))), w.r.H}
				}
			}
		}
	}
	return best, true
}

// ocrMatches converts lines to match entries. origin is the screen position of the image's
// top-left; scale is image pixels per point; window (when non-nil) gives windowCenter.
func ocrMatches(lines []ocrLine, origin rect, scale float64, window *rect, filter string) []any {
	out := []any{}
	toPts := func(r rect) rect {
		return rect{origin.X + r.X/scale, origin.Y + r.Y/scale, r.W / scale, r.H / scale}
	}
	for _, l := range lines {
		var span rect
		if filter != "" {
			s, ok := matchSpan(l, filter)
			if !ok {
				continue
			}
			span = s
		}
		fr := toPts(l.r)
		cx, cy := fr.center()
		m := obj{"text": l.text, "confidence": round2(l.conf), "frame": fr.obj(), "center": point(cx, cy)}
		if window != nil {
			m["windowCenter"] = point(cx-window.X, cy-window.Y)
		}
		if filter != "" {
			sr := toPts(span)
			sx, sy := sr.center()
			mm := obj{"frame": sr.obj(), "center": point(sx, sy)}
			if window != nil {
				mm["windowCenter"] = point(sx-window.X, sy-window.Y)
			}
			m["match"] = mm
		}
		out = append(out, m)
	}
	return out
}

func cmdOCR(a map[string]string) (obj, *cmdError) {
	level := a["level"]
	if level == "" {
		level = "accurate"
	}
	if level != "accurate" && level != "fast" {
		return nil, errf("Invalid --level '%s': use accurate or fast", level)
	}
	mode, e := parseMode(a["mode"])
	if e != nil {
		return nil, e
	}
	timeout, e := parseDuration("timeout", a["timeout"], 60*time.Second)
	if e != nil {
		return nil, e
	}
	scale := 1.0
	if a["scale"] != "" {
		if scale, e = parseFloatArg("scale", a["scale"]); e != nil || scale <= 0 {
			return nil, errf("Invalid --scale '%s': use a positive number", a["scale"])
		}
	}
	if a["image"] != "" && a["app"] != "" {
		return nil, errf("--image cannot be combined with --app")
	}
	if tesseractPath() == "" {
		return nil, errObj(obj{"error": "Tesseract is not installed, so text can't be recognized. " + tesseractInstall, "dependency": "tesseract"})
	}
	langs, e := tessLanguages(a["languages"])
	if e != nil {
		return nil, e
	}
	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	o := obj{"level": level}
	if a["text"] != "" {
		o["text"] = a["text"]
	}
	var img image.Image
	var origin rect
	var window *rect
	if a["image"] != "" {
		f, err := os.Open(a["image"])
		if err != nil {
			return nil, errf("Cannot open --image %s: %v", a["image"], err)
		}
		im, _, err := image.Decode(f)
		f.Close()
		if err != nil {
			return nil, errf("Cannot read --image %s as PNG: %v", a["image"], err)
		}
		img = im
		o["source"], o["path"], o["mode"] = "image", a["image"], "background"
		o["width"], o["height"], o["scale"] = round1(float64(im.Bounds().Dx())/scale), round1(float64(im.Bounds().Dy())/scale), scale
	} else {
		type res struct {
			c *capture
			e *cmdError
		}
		ch := make(chan res, 1)
		go func() {
			c, e := captureTarget("ocr", a["app"], 0, mode)
			ch <- res{c, e}
		}()
		capLimit := 10 * time.Second
		if timeout < capLimit {
			capLimit = timeout
		}
		var c *capture
		select {
		case r := <-ch:
			if r.e != nil {
				return nil, r.e
			}
			c = r.c
		case <-time.After(capLimit):
			return nil, errObj(obj{"error": fmt.Sprintf("Screen capture did not respond within %s", capLimit), "phase": "capture", "timeout": timeout.Seconds(), "waited": secs(time.Since(start))})
		}
		x, _ := X("ocr")
		img, origin, scale = c.img, c.origin, 1
		o["source"], o["mode"] = c.source, c.mode
		o["width"], o["height"], o["scale"] = c.img.Bounds().Dx(), c.img.Bounds().Dy(), 1
		if c.app != nil {
			o["app"] = c.app.Name
			o["window"] = c.windowObj(x)
			r := c.origin
			window = &r
		}
		if c.mode == "foreground" {
			setFallback(o, "foreground", c.reason, c.restored)
		}
	}
	lines, err := runOCR(ctx, img, level, langs)
	if err != nil {
		if err.fields["error"] == "deadline" {
			return nil, errObj(obj{"error": fmt.Sprintf("Text recognition did not finish within %s", timeout), "phase": "recognition", "timeout": timeout.Seconds(), "waited": secs(time.Since(start))})
		}
		return nil, err
	}
	o["matches"] = ocrMatches(lines, origin, scale, window, a["text"])
	return o, nil
}
