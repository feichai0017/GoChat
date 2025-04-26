package domain

import (
	"sync/atomic"
	"unsafe"
)

type Endport struct {
	IP 			string 		 `json:"ip"`
	Port 		string 		 `json:"port"`
	ActiveScore float64 	 `json:"-"`
	StaticScore float64 	 `json:"-"`
	Stats       *Stat  		 `json:"-"`
	window      *stateWindow `json:"-"`
}



func NewEndport(ip, port string) *Endport {
	ed := &Endport{
		IP: ip,
		Port: port,
	}
	ed.window = newStateWindow()
	ed.Stats = ed.window.getStat()
	go func() {
		for stat := range ed.window.statChan {
			ed.window.appendStat(stat)
			newStat := ed.window.getStat()
			atomic.SwapPointer((*unsafe.Pointer)(unsafe.Pointer(ed.Stats)), unsafe.Pointer(newStat))
		}
	}()
	return ed
}

func (ed *Endport) UpdateStat(s *Stat) {
	ed.window.statChan <- s
}

func (ed *Endport) CalculateScore(ctx *IpConfContext) {
	// if stats is nil, use the last calculated score
	if ed.Stats != nil {
		ed.ActiveScore = ed.Stats.CalculateActiveScore()
		ed.StaticScore = ed.Stats.CalculateStaticScore()
	}
}

func (ed *Endport) Close() {
	close(ed.window.statChan)
}
