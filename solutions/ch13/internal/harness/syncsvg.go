package harness

import (
	"fmt"
	"html"
	"io"
	"strconv"
	"strings"
)

// WriteSyncSVG draws final regret against sync interval: one line per policy,
// the intervals evenly spaced as categories, longest interval (and "never")
// on the left.
func WriteSyncSVG(w io.Writer, spec SyncSpec, points []SyncPoint) error {
	var names []string
	byPolicy := map[string][]SyncPoint{}
	for _, p := range points {
		if _, ok := byPolicy[p.Policy]; !ok {
			names = append(names, p.Policy)
		}
		byPolicy[p.Policy] = append(byPolicy[p.Policy], p)
	}
	yMax := 0.0
	for _, p := range points {
		yMax = max(yMax, p.Regret+p.StdErr)
	}
	yTicks := niceTicks(yMax, 5)
	yTop := yTicks[len(yTicks)-1]

	plotW := float64(svgW - padL - padR)
	plotH := float64(svgH - padT - padB)
	n := len(spec.Intervals)
	sx := func(i int) float64 {
		if n == 1 {
			return padL + plotW/2
		}
		return padL + float64(i)/float64(n-1)*plotW
	}
	sy := func(v float64) float64 { return padT + plotH - v/yTop*plotH }

	label := func(iv int) string {
		if iv == 0 {
			return "never"
		}
		return strconv.Itoa(iv)
	}
	var b strings.Builder
	title := fmt.Sprintf("Regret with %d replicas by sync interval", spec.Replicas)
	desc := fmt.Sprintf("Mean final regret over %d seeds on the %q scenario, %d steps.", spec.Seeds, spec.Scenario, spec.Steps)
	fmt.Fprintf(&b, `<svg class="regret-chart" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" role="img" aria-labelledby="rc-title rc-desc">`+"\n", svgW, svgH)
	fmt.Fprintf(&b, "<title id=\"rc-title\">%s</title>\n<desc id=\"rc-desc\">%s</desc>\n", html.EscapeString(title), html.EscapeString(desc))
	fmt.Fprintf(&b, "<style>%s</style>\n", styleBlock)
	fmt.Fprintf(&b, `<rect class="bg" width="%d" height="%d"/>`+"\n", svgW, svgH)
	fmt.Fprintf(&b, `<text class="title" x="%d" y="24">%s</text>`+"\n", padL,
		html.EscapeString(fmt.Sprintf("Final regret · %d replicas · %s · mean of %d seeds", spec.Replicas, spec.Scenario, spec.Seeds)))
	for _, v := range yTicks {
		y := sy(v)
		fmt.Fprintf(&b, `<line class="grid" x1="%d" x2="%d" y1="%.1f" y2="%.1f"/>`+"\n", padL, svgW-padR, y, y)
		fmt.Fprintf(&b, `<text x="%d" y="%.1f" text-anchor="end" dominant-baseline="middle">%s</text>`+"\n", padL-8, y, formatTick(v))
	}
	for i, iv := range spec.Intervals {
		fmt.Fprintf(&b, `<text x="%.1f" y="%d" text-anchor="middle">%s</text>`+"\n", sx(i), svgH-padB+20, label(iv))
	}
	fmt.Fprintf(&b, `<line class="axis" x1="%d" x2="%d" y1="%.1f" y2="%.1f"/>`+"\n", padL, svgW-padR, sy(0), sy(0))
	fmt.Fprintf(&b, `<text x="%.1f" y="%d" text-anchor="middle">steps between syncs</text>`+"\n", padL+plotW/2, svgH-8)

	type lab struct {
		y    float64
		text string
		i    int
	}
	var labs []lab
	for k, name := range names {
		color := fmt.Sprintf("var(--series-%d)", k%4+1)
		var line strings.Builder
		for i, p := range byPolicy[name] {
			fmt.Fprintf(&line, "%.1f,%.1f ", sx(i), sy(p.Regret))
		}
		dash := ""
		if d := seriesDash[k%len(seriesDash)]; d != "" {
			dash = fmt.Sprintf(` stroke-dasharray="%s"`, d)
		}
		fmt.Fprintf(&b, `<polyline class="line" stroke="%s"%s points="%s"/>`+"\n", color, dash, strings.TrimSpace(line.String()))
		for i, p := range byPolicy[name] {
			fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="3.5" fill="%s"/>`+"\n", sx(i), sy(p.Regret), color)
		}
		last := byPolicy[name][len(byPolicy[name])-1]
		labs = append(labs, lab{y: sy(last.Regret), text: name, i: k})
	}
	for i := 1; i < len(labs); i++ {
		for j := i; j > 0 && labs[j].y < labs[j-1].y; j-- {
			labs[j], labs[j-1] = labs[j-1], labs[j]
		}
	}
	for i := 1; i < len(labs); i++ {
		if labs[i].y-labs[i-1].y < 15 {
			labs[i].y = labs[i-1].y + 15
		}
	}
	for _, l := range labs {
		fmt.Fprintf(&b, `<text class="label" x="%d" y="%.1f" dominant-baseline="middle">`+
			`<tspan fill="var(--series-%d)">●</tspan> %s</text>`+"\n", svgW-padR+8, l.y, l.i%4+1, html.EscapeString(l.text))
	}
	b.WriteString("</svg>\n")
	_, err := io.WriteString(w, b.String())
	return err
}
