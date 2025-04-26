package domain

const (
	windowSize = 5
)

type stateWindow struct {
	stateQueue []*Stat
	statChan   chan *Stat
	sumStat    *Stat
	idx        int64
}

func newStateWindow() *stateWindow {
	return &stateWindow{
		stateQueue: make([]*Stat, windowSize),
		statChan:   make(chan *Stat),
		sumStat:    &Stat{},
	}
}

func (sw *stateWindow) getStat() *Stat {
	res := sw.sumStat.Clone()
	res.Avg(windowSize)
	return res
}

func (sw *stateWindow) appendStat(s *Stat) {
	// minus the old stat
	sw.sumStat.Sub(sw.stateQueue[sw.idx % windowSize])
	// update the latest stat
	sw.stateQueue[sw.idx % windowSize] = s
	// calculate the latest window sum
	sw.sumStat.Add(s)
	sw.idx++
}