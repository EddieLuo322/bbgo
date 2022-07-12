package indicator

import (
	"fmt"
	"time"

	"github.com/c9s/bbgo/pkg/types"
)

const MaxNumOfSMA = 5_000
const MaxNumOfSMATruncateSize = 100

//go:generate callbackgen -type SMA
type SMA struct {
	types.SeriesBase
	types.IntervalWindow
	Values    types.Float64Slice
	rawValues *types.Queue
	EndTime   time.Time

	UpdateCallbacks []func(value float64)
}

func (inc *SMA) Last() float64 {
	if inc.Values.Length() == 0 {
		return 0.0
	}
	return inc.Values.Last()
}

func (inc *SMA) Index(i int) float64 {
	if i >= inc.Values.Length() {
		return 0.0
	}

	return inc.Values.Index(i)
}

func (inc *SMA) Length() int {
	return inc.Values.Length()
}

func (inc *SMA) Clone() types.UpdatableSeriesExtend {
	out := &SMA{
		Values:    inc.Values[:],
		rawValues: types.Clone(inc.rawValues).(*types.Queue),
		EndTime:   inc.EndTime,
	}
	out.SeriesBase.Series = out
	return out
}

var _ types.SeriesExtend = &SMA{}

func (inc *SMA) Update(value float64) {
	if inc.rawValues == nil {
		inc.rawValues = types.NewQueue(inc.Window)
		inc.SeriesBase.Series = inc
	}

	inc.rawValues.Update(value)
	if inc.rawValues.Length() < inc.Window {
		return
	}

	inc.Values.Push(types.Mean(inc.rawValues))
}

func (inc *SMA) BindK(target KLineClosedEmitter, symbol string, interval types.Interval) {
	target.OnKLineClosed(types.KLineWith(symbol, interval, inc.PushK))
}

func (inc *SMA) PushK(k types.KLine) {
	if inc.EndTime != zeroTime && k.EndTime.Before(inc.EndTime) {
		return
	}

	inc.Update(k.Close.Float64())
	inc.EndTime = k.EndTime.Time()
	inc.EmitUpdate(inc.Values.Last())
}

func (inc *SMA) LoadK(allKLines []types.KLine) {
	for _, k := range allKLines {
		inc.PushK(k)
	}
}

func (inc *SMA) CalculateAndUpdate(allKLines []types.KLine) {
	if inc.rawValues == nil {
		inc.LoadK(allKLines)
	} else {
		var last = allKLines[len(allKLines)-1]
		inc.PushK(last)
	}

}

func (inc *SMA) handleKLineWindowUpdate(interval types.Interval, window types.KLineWindow) {
	if inc.Interval != interval {
		return
	}

	inc.CalculateAndUpdate(window)
}

func (inc *SMA) Bind(updater KLineWindowUpdater) {
	updater.OnKLineWindowUpdate(inc.handleKLineWindowUpdate)
}

func calculateSMA(kLines []types.KLine, window int, priceF KLinePriceMapper) (float64, error) {
	length := len(kLines)
	if length == 0 || length < window {
		return 0.0, fmt.Errorf("insufficient elements for calculating SMA with window = %d", window)
	}
	if length != window {
		return 0.0, fmt.Errorf("too much klines passed in, requires only %d klines", window)
	}

	sum := 0.0
	for _, k := range kLines {
		sum += priceF(k)
	}

	avg := sum / float64(window)
	return avg, nil
}
