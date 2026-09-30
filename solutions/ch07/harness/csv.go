package harness

import (
	"encoding/csv"
	"io"
	"strconv"

	"banditlab/stat"
)

// WriteCSV writes one row per policy per sampled step: step, policy,
// mean_regret, stderr. At most points steps are written per policy, evenly
// spaced and including the last.
func (r *Report) WriteCSV(w io.Writer, points int) error {
	cw := csv.NewWriter(w)
	if err := cw.Write([]string{"step", "policy", "mean_regret", "stderr"}); err != nil {
		return err
	}
	steps := make([]int, r.Spec.Steps)
	for i := range steps {
		steps[i] = i
	}
	for _, c := range r.Curves {
		for _, t := range stat.Sample(steps, points) {
			row := []string{
				strconv.Itoa(t + 1),
				c.Policy,
				strconv.FormatFloat(c.Mean[t], 'f', 3, 64),
				strconv.FormatFloat(c.StdErr[t], 'f', 3, 64),
			}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	return cw.Error()
}
