package chart

import (
	"fmt"
	//"math"
	//	"os"
	//	"strings"
)


// CategoryBarChart represents bar charts categorical (non-numeric) x-axis.
type CategoryBarChart struct {
	Categories []string // List of categories to display (ordered from left to right)
	YRange     Range    // y axis (numeric)
	Title      string   // Title of the chart
	Key        Key      // Key/legend
	Horizontal bool     // Display as horizontal bars (swap x and y axis). Unimplemented
	Stacked    bool     // Display different data sets ontop of each other.
	ShowVal    bool     // Display values 
	Data       []CategoryBarChartData
}

// CategoryBarChartData encapsulated a single data set in a CategoryBarChart.
type CategoryBarChartData struct {
	Name    string             // Name of this data set
	Style   DataStyle          // Style
	Samples map[string]float64 // Keys not in CategoryBarChart.Categories are ignored
}

// AddData will add data to the chart. Note that data may contain keys not present
// in the Categories of the CategoryBarChart: These will be ignored.
func (c *CategoryBarChart) AddData(name string, data map[string]float64, style DataStyle) {
	c.Data = append(c.Data, CategoryBarChartData{name, style, data})
	if name != "" {
		c.Key.Entries = append(c.Key.Entries, KeyEntry{Style: style, Text: name, PlotStyle: PlotStyleBox})
	}

	if len(c.Data) == 1 { // first data set
		for _, v := range data {
			c.YRange.DataMin = v
			c.YRange.DataMax = v
			break
		}
	}
	for _, v := range data {
		if v < c.YRange.DataMin {
			c.YRange.DataMin = v
		} else if v > c.YRange.DataMax {
			c.YRange.DataMax = v
		}
	}
	c.YRange.Min = c.YRange.DataMin
	c.YRange.Max = c.YRange.DataMax
}


func (c *CategoryBarChart) catIdx(cat string) (i int) {
	i = -1
	for n, c := range c.Categories {
		if c == cat {
			i = n
			return
		}
	}
	return
}


// Plot outputs the chart to g.
func (c *CategoryBarChart) Plot(g Graphics) {
	n := len(c.Categories)
	xrange := Range{}
	xrange.DataMin, xrange.DataMax = 1, float64(n)
	xrange.Min, xrange.Max = 0.5, float64(n)+0.5
	xrange.MinMode = RangeMode{Fixed: true, Value: 0.5}
	xrange.MaxMode = RangeMode{Fixed: true, Value: float64(n) + 0.5}

	// layout
	layout := Layout(g, c.Title, xrange.Label, c.YRange.Label,
		xrange.TicSetting.Hide, c.YRange.TicSetting.Hide, &c.Key)
	width, height := layout.Width, layout.Height
	topm, leftm := layout.Top, layout.Left
	numytics := layout.NumYtics
	fw, fh, _ := g.FontMetrics(g.Font("label"))
	fw += 0

	// Outside bound ranges for histograms are nicer
	leftm, width = leftm+int(2*fw), width-int(2*fw)
	topm, height = topm, height-fh

	if c.Stacked {
		// rescale y-axis
		sum := make([]float64, n)
		min, max := c.YRange.DataMin, c.YRange.DataMax
		for _, d := range c.Data {
			for k, v := range d.Samples {
				i := c.catIdx(k)
				if i == -1 {
					continue
				}
				sum[i] += v
				if sum[i] > max {
					max = sum[i]
				} else if sum[i] < min {
					min = sum[i]
				}
			}
		}
		// stacked histograms and y-axis _not_ starting at 0 is
		// utterly braindamaged and missleading: Fix 0
		c.YRange.DataMin, c.YRange.Min, c.YRange.DataMax = 0, 0, max
		c.YRange.MinMode.Fixed, c.YRange.MinMode.Value = true, 0
		fmt.Printf("YRange = %#v\n", c.YRange)
	}
	c.YRange.Setup(numytics, numytics+2, height, topm, true)

	// categories are tic labels of x axis
	xrange.Setup(n, n, width, leftm, false)
	xrange.Tics = make([]Tic, n)
	for i := 0; i < n; i++ {
		xrange.Tics[i].Pos = -1 // outside, no tic
		xrange.Tics[i].LabelPos = float64(i) + 1
		xrange.Tics[i].Label = c.Categories[i]
		xrange.Tics[i].Align = 0 // center
	}

	// Start of drawing
	g.Begin()
	if c.Title != "" {
		g.Title(c.Title)
	}

	g.XAxis(xrange, topm+height+fh, topm)
	g.YAxis(c.YRange, leftm-int(2*fw), leftm+width)

	xf := xrange.Data2Screen
	yf := c.YRange.Data2Screen
	var sy0 int
	switch {
	case c.YRange.Min >= 0:
		sy0 = yf(c.YRange.Min)
	case c.YRange.Min < 0 && c.YRange.Max > 0:
		sy0 = yf(0)
	case c.YRange.Max <= 0:
		sy0 = yf(c.YRange.Max)
	default:
		fmt.Printf("No f.... idea how this can happen. You've been fiddeling?")
	}

	// TODO: gap between bars.
	var sbw, fbw int
	if c.Stacked {
		sbw = (xf(2) - xf(0)) / 4
		fbw = sbw
	} else {
		//        V
		//   xxx === 000 ... xxx    sbw = 3
		//   xx == 00 ## .. xx ==   fbw = 11
		sbw = (xf(1)-xf(0))/(len(c.Data)+1) - 1
		fbw = len(c.Data)*sbw + len(c.Data) - 1
	}

	current := make([]float64, n)
	for dn, data := range c.Data {
		bars := make([]Barinfo, len(data.Samples))
		z := 0
		for k, v := range data.Samples {
			i := c.catIdx(k)
			if i == -1 {
				continue
			}

			sx := xf(float64(i+1)) - fbw/2
			if !c.Stacked {
				sx += dn * (sbw + 1)
			}

			var sy, sh int
			if c.Stacked {
				sy = yf(v + current[i])
				sh = yf(current[i]) - sy
			} else {
				sy = yf(v)
				sh = sy0 - sy
			}
			bars[z].x, bars[z].y = sx, sy
			bars[z].w, bars[z].h = sbw, sh
			z++
			current[i] += v

			if c.ShowVal {
				_ = fmt.Sprintf("%f", v)
			}
		}
		g.Bars(bars, data.Style)

	}

	if !c.Key.Hide {
		g.Key(layout.KeyX, layout.KeyY, c.Key)
	}

	g.End()
}
