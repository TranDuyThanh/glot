// Glot is a library for having simplified 1,2,3 Dimensional points/line plots
// It's built on top of Gnu plot and offers the ability to use Raw Gnu plot commands
// directly from golang.
// See the gnuplot documentation page for the exact semantics of the gnuplot
// commands.
//  http://www.gnuplot.info/

package glot

import (
	"fmt"
	"io/ioutil"
	"os"
)

// Plot is the basic type representing a plot.
// Every plot has a set of Pointgroups that are simultaneously plotted
// on a 2/3 D plane given the plot type.
// The Plot dimensions must be specified at the time of construction
// and can't be changed later.  All the Pointgroups added to a plot must
// have same dimensions as the dimension specified at the
// the time of plot construction.
// The Pointgroups can be dynamically added and removed from a plot
// And style changes can also be made dynamically.
type Plot struct {
	proc       *plotterProcess
	debug      bool
	plotcmd    string
	nplots     int                    // number of currently active plots
	tmpfiles   tmpfilesDb             // A temporary file used for saving data
	dimensions int                    // dimensions of the plot
	PointGroup map[string]*PointGroup // A map between Curve name and curve type. This maps a name to a given curve in a plot. Only one curve with a given name exists in a plot.
	format     string                 // The saving format of the plot. This could be PDF, PNG, JPEG and so on.
	style      string                 // style of the plot
	title      string                 // The title of the plot.
}

// NewPlot Function makes a new plot with the specified dimensions.
//
// Usage
//  dimensions := 3
//  persist := false
//  debug := false
//  plot, _ := glot.NewPlot(dimensions, persist, debug)
// Variable definitions
//  dimensions  :=> refers to the dimensions of the plot.
//  debug       :=> can be used by developers to check the actual commands sent to gnu plot.
//  persist     :=> used to make the gnu plot window stay open.
func NewPlot(dimensions int, persist, debug bool) (*Plot, error) {
	p := &Plot{proc: nil, debug: debug, plotcmd: "plot",
		nplots: 0, dimensions: dimensions, style: "points", format: "png"}
	p.PointGroup = make(map[string]*PointGroup) // Adding a mapping between a curve name and a curve
	p.tmpfiles = make(tmpfilesDb)
	proc, err := newPlotterProc(persist)
	if err != nil {
		return nil, err
	}
	// Only 1,2,3 Dimensional plots are supported
	if dimensions > 3 || dimensions < 1 {
		return nil, &gnuplotError{fmt.Sprintf("invalid number of dims '%v'", dimensions)}
	}
	p.proc = proc
	return p, nil
}

func (plot *Plot) plotX(pointGroup *PointGroup) error {
	f, err := ioutil.TempFile(os.TempDir(), gGnuplotPrefix)
	if err != nil {
		return err
	}
	fname := f.Name()
	plot.tmpfiles[fname] = f
	for _, d := range pointGroup.castedData.([]float64) {
		f.WriteString(fmt.Sprintf("%v\n", d))
	}
	f.Close()
	cmd := plot.plotcmd
	if plot.nplots > 0 {
		cmd = plotCommand
	}
	if pointGroup.style == "" {
		pointGroup.style = defaultStyle
	}
	var line string
	if pointGroup.name == "" {
		line = fmt.Sprintf("%s \"%s\" with %s", cmd, fname, pointGroup.style)
	} else {
		line = fmt.Sprintf("%s \"%s\" title \"%s\" with %s",
			cmd, fname, pointGroup.name, pointGroup.style)
	}

	if pointGroup.pointSize > 0 {
		line = fmt.Sprintf(`%s pt %d ps %.2f`, line, pointGroup.pointType, pointGroup.pointSize)
	}

	plot.nplots++
	return plot.Cmd(line)
}

func (plot *Plot) plotXY(pointGroup *PointGroup) error {
	x := pointGroup.castedData.([][]float64)[0]
	y := pointGroup.castedData.([][]float64)[1]
	npoints := min(len(x), len(y))

	f, err := ioutil.TempFile(os.TempDir(), gGnuplotPrefix)
	if err != nil {
		return err
	}
	fname := f.Name()
	plot.tmpfiles[fname] = f

	for i := 0; i < npoints; i++ {
		f.WriteString(fmt.Sprintf("%v %v\n", x[i], y[i]))
	}

	f.Close()
	cmd := plot.plotcmd
	if plot.nplots > 0 {
		cmd = plotCommand
	}

	if pointGroup.style == "" {
		pointGroup.style = "points"
	}
	var line string
	if pointGroup.name == "" {
		line = fmt.Sprintf("%s \"%s\" with %s", cmd, fname, pointGroup.style)
	} else {
		line = fmt.Sprintf("%s \"%s\" title \"%s\" with %s",
			cmd, fname, pointGroup.name, pointGroup.style)
	}

	if pointGroup.pointSize > 0 {
		line = fmt.Sprintf(`%s pt %d ps %.2f`, line, pointGroup.pointType, pointGroup.pointSize)
	}

	plot.nplots++
	return plot.Cmd(line)
}

func (plot *Plot) plotXYZ(pointGroup *PointGroup) error {
	x := pointGroup.castedData.([][]float64)[0]
	y := pointGroup.castedData.([][]float64)[1]
	z := pointGroup.castedData.([][]float64)[2]
	npointGroup := min(len(x), len(y))
	npointGroup = min(npointGroup, len(z))
	f, err := ioutil.TempFile(os.TempDir(), gGnuplotPrefix)
	if err != nil {
		return err
	}
	fname := f.Name()
	plot.tmpfiles[fname] = f

	for i := 0; i < npointGroup; i++ {
		f.WriteString(fmt.Sprintf("%v %v %v\n", x[i], y[i], z[i]))
	}

	f.Close()
	cmd := "splot" // Force 3D plot
	if plot.nplots > 0 {
		cmd = plotCommand
	}

	var line string
	if pointGroup.name == "" {
		line = fmt.Sprintf("%s \"%s\" with %s", cmd, fname, pointGroup.style)
	} else {
		line = fmt.Sprintf("%s \"%s\" title \"%s\" with %s",
			cmd, fname, pointGroup.name, pointGroup.style)
	}

	if pointGroup.pointSize > 0 {
		line = fmt.Sprintf(`%s pt %d ps %.2f`, line, pointGroup.pointType, pointGroup.pointSize)
	}

	plot.nplots++
	return plot.Cmd(line)
}

func (plot *Plot) plotCandlesticks(PointGroup *PointGroup) error {
	data := PointGroup.castedData.(CandlesticksData)
	nCandles := len(data.XArray)

	f, err := ioutil.TempFile(os.TempDir(), gGnuplotPrefix)
	if err != nil {
		return err
	}
	fname := f.Name()
	plot.tmpfiles[fname] = f

	for i := 0; i < nCandles; i++ {
		f.WriteString(fmt.Sprintf("%v %v %v %v %v\n", data.XArray[i], data.Candles[i][0], data.Candles[i][1], data.Candles[i][2], data.Candles[i][3]))
	}
	f.Close()

	err = plot.Cmd(fmt.Sprintf(`set palette defined (-1 '%s', 1 '%s')`, data.DownColor, data.UpColor))
	if err != nil {
		return err
	}
	err = plot.Cmd(`set cbrange [-1:1]`)
	if err != nil {
		return err
	}
	err = plot.Cmd(`unset colorbox`)
	if err != nil {
		return err
	}
	err = plot.Cmd(`set style fill solid noborder`)
	if err != nil {
		return err
	}
	err = plot.Cmd(fmt.Sprintf(`set boxwidth %f`, data.BoxWidth))
	if err != nil {
		return err
	}

	cmd := plot.plotcmd
	if plot.nplots > 0 {
		cmd = plotCommand
	}

	if PointGroup.style == "" {
		PointGroup.style = "candlesticks"
	}
	var line string
	if PointGroup.name == "" {
		line = fmt.Sprintf("%s \"%s\" using 1:2:4:3:5:($5 < $2 ? -1 : 1) with %s palette", cmd, fname, PointGroup.style)
	} else {
		line = fmt.Sprintf("%s \"%s\" using 1:2:4:3:5:($5 < $2 ? -1 : 1) title \"%s\" with %s palette",
			cmd, fname, PointGroup.name, PointGroup.style)
	}
	plot.nplots++
	return plot.Cmd(line)
}
