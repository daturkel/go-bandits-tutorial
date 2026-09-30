package harness

import (
	"fmt"
	"html"
	"io"
	"math"
	"strings"

	"banditlab/internal/stat"
)

// The chart is drawn in a 780x400 box. Colours come from CSS custom
// properties (--series-1 ...), with the fallbacks below, so a page can theme
// the chart and the standalone file still looks right.
const (
	svgW, svgH             = 780, 400
	padL, padR, padT, padB = 64, 176, 44, 48
	chartPoints            = 240
	styleBlock             = `
.regret-chart:not([data-embedded]){--surface:#fcfcfb;--ink:#0b0b0b;--ink2:#52514e;--grid:#e4e3df;--series-1:#2a78d6;--series-2:#eb6834;--series-3:#1baf7a;--series-4:#eda100}
.regret-chart{font-family:system-ui,sans-serif}
@media (prefers-color-scheme:dark){.regret-chart:not([data-embedded]){--surface:#1a1a19;--ink:#fff;--ink2:#c3c2b7;--grid:#34342f;--series-1:#3987e5;--series-2:#d95926;--series-3:#199e70;--series-4:#c98500}}
.regret-chart .bg{fill:var(--surface)}
.regret-chart .grid{stroke:var(--grid);stroke-width:1}
.regret-chart .axis{stroke:var(--ink2);stroke-width:1}
.regret-chart text{fill:var(--ink2);font-size:12px}
.regret-chart .title{fill:var(--ink);font-size:14px;font-weight:600}
.regret-chart .line{fill:none;stroke-width:2;stroke-linejoin:round}
.regret-chart .band{stroke:none;opacity:.18}
.regret-chart .label{fill:var(--ink);font-size:12px}
`
)

var seriesDash = []string{"", "6 4", "", "6 4"}

// WriteSVG draws the mean regret curves with a +-1 standard-error band.
func (r *Report) WriteSVG(w io.Writer) error {
	xMax := float64(r.Spec.Steps)
	yMax := 0.0
	for _, c := range r.Curves {
		for t, m := range c.Mean {
			yMax = math.Max(yMax, m+c.StdErr[t])
		}
	}
	yTicks := niceTicks(yMax, 5)
	yTop := yTicks[len(yTicks)-1]
	xTicks := niceTicks(xMax, 5)

	plotW := float64(svgW - padL - padR)
	plotH := float64(svgH - padT - padB)
	sx := func(step float64) float64 { return padL + step/xMax*plotW }
	sy := func(v float64) float64 { return padT + plotH - v/yTop*plotH }

	var b strings.Builder
	title := fmt.Sprintf("Cumulative regret on the %q scenario", r.Spec.Scenario)
	desc := fmt.Sprintf("Mean over %d seeds; shaded band is one standard error. Final mean regret: ", r.Spec.Seeds)
	for i, c := range r.Curves {
		if i > 0 {
			desc += "; "
		}
		desc += fmt.Sprintf("%s %.1f", c.Policy, stat.Mean(c.Final))
	}
	fmt.Fprintf(&b, `<svg class="regret-chart" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" role="img" aria-labelledby="rc-title rc-desc">`+"\n", svgW, svgH)
	fmt.Fprintf(&b, "<title id=\"rc-title\">%s</title>\n<desc id=\"rc-desc\">%s</desc>\n", html.EscapeString(title), html.EscapeString(desc))
	fmt.Fprintf(&b, "<style>%s</style>\n", styleBlock)
	fmt.Fprintf(&b, `<rect class="bg" width="%d" height="%d"/>`+"\n", svgW, svgH)
	fmt.Fprintf(&b, `<text class="title" x="%d" y="24">%s</text>`+"\n", padL, html.EscapeString(fmt.Sprintf("Cumulative regret · %s · mean of %d seeds", r.Spec.Scenario, r.Spec.Seeds)))

	for _, v := range yTicks {
		y := sy(v)
		fmt.Fprintf(&b, `<line class="grid" x1="%d" x2="%d" y1="%.1f" y2="%.1f"/>`+"\n", padL, svgW-padR, y, y)
		fmt.Fprintf(&b, `<text x="%d" y="%.1f" text-anchor="end" dominant-baseline="middle">%s</text>`+"\n", padL-8, y, formatTick(v))
	}
	for _, v := range xTicks {
		x := sx(v)
		fmt.Fprintf(&b, `<text x="%.1f" y="%d" text-anchor="middle">%s</text>`+"\n", x, svgH-padB+20, formatTick(v))
	}
	fmt.Fprintf(&b, `<line class="axis" x1="%d" x2="%d" y1="%.1f" y2="%.1f"/>`+"\n", padL, svgW-padR, sy(0), sy(0))
	fmt.Fprintf(&b, `<text x="%.1f" y="%d" text-anchor="middle">step</text>`+"\n", padL+plotW/2, svgH-8)

	steps := make([]int, r.Spec.Steps)
	for i := range steps {
		steps[i] = i
	}
	sampled := stat.Sample(steps, chartPoints)

	for i, c := range r.Curves {
		color := fmt.Sprintf("var(--series-%d)", i%4+1)
		var upper, lower, line strings.Builder
		for _, t := range sampled {
			x := sx(float64(t + 1))
			fmt.Fprintf(&upper, "%.1f,%.1f ", x, sy(c.Mean[t]+c.StdErr[t]))
			fmt.Fprintf(&line, "%.1f,%.1f ", x, sy(c.Mean[t]))
		}
		for j := len(sampled) - 1; j >= 0; j-- {
			t := sampled[j]
			fmt.Fprintf(&lower, "%.1f,%.1f ", sx(float64(t+1)), sy(math.Max(c.Mean[t]-c.StdErr[t], 0)))
		}
		fmt.Fprintf(&b, `<polygon class="band" fill="%s" points="%s%s"/>`+"\n", color, upper.String(), lower.String())
		dash := ""
		if d := seriesDash[i%len(seriesDash)]; d != "" {
			dash = fmt.Sprintf(` stroke-dasharray="%s"`, d)
		}
		fmt.Fprintf(&b, `<polyline class="line" stroke="%s"%s points="%s"/>`+"\n", color, dash, strings.TrimSpace(line.String()))
	}

	// Direct labels at the right edge, nudged apart so they never overlap.
	type lab struct {
		y    float64
		text string
		i    int
	}
	labs := make([]lab, len(r.Curves))
	for i, c := range r.Curves {
		labs[i] = lab{y: sy(c.Mean[len(c.Mean)-1]), text: c.Policy, i: i}
	}
	// insertion sort by y (top to bottom); the slice is tiny
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

// niceTicks returns round tick values from 0 up to at least max.
func niceTicks(max float64, target int) []float64 {
	if max <= 0 {
		return []float64{0, 1}
	}
	raw := max / float64(target)
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	var step float64
	switch f := raw / mag; {
	case f <= 1:
		step = mag
	case f <= 2:
		step = 2 * mag
	case f <= 5:
		step = 5 * mag
	default:
		step = 10 * mag
	}
	var ticks []float64
	for v := 0.0; v < max+step*0.999; v += step {
		ticks = append(ticks, v)
		if v >= max {
			break
		}
	}
	return ticks
}

func formatTick(v float64) string {
	if v >= 1000 && math.Mod(v, 1000) == 0 {
		return fmt.Sprintf("%.0fk", v/1000)
	}
	return fmt.Sprintf("%.0f", v)
}
