package domain

import (
	"sort"
	"sync"

	"github.com/feichai0017/GoChat/ipconf/source"
)

type Dispatcher struct {
	candidateTable map[string]*Endport
	sync.RWMutex
}

var dp *Dispatcher

func Init() {
	dp = &Dispatcher{}
	dp.candidateTable = make(map[string]*Endport)
	go func() {
		for event := range source.EventChan() {
			switch event.Type {
			case source.AddNodeEvent:
				dp.addNode(event)
			case source.DelNodeEvent:
				dp.delNode(event)
			}
		}
	}()
}

func Dispatch(ctx *IpConfContext) []*Endport {
	// step 1: get all candidate nodes
	eds := dp.getCandidateEndport(ctx)

	// step2: calculate the score for each node
	for _, ed := range eds {
		ed.CalculateScore(ctx)
	}

	// step3: sort the nodes by score, with active and static scores
	sort.Slice(eds, func(i, j int) bool {
		// active score is more important
		if eds[i].ActiveScore > eds[j].ActiveScore {
			return true
		}
		// if active score is the same, then sort by static score
		if eds[i].ActiveScore == eds[j].ActiveScore {
			return eds[i].StaticScore > eds[j].StaticScore
		}
		return false
	})
	return eds
}

func (dp *Dispatcher) getCandidateEndport(ctx *IpConfContext) []*Endport {
	dp.RLock()
	defer dp.RUnlock()
	// filter the candidate nodes by the given context
	candidateList := make([]*Endport, 0, len(dp.candidateTable))
	for _, ed := range dp.candidateTable {
		candidateList = append(candidateList, ed)
	}

	return candidateList
}

func (dp *Dispatcher) addNode(event *source.Event) {
	dp.Lock()
	defer dp.Unlock()
	var (
		ed *Endport
		ok bool
	)
	if ed, ok = dp.candidateTable[event.IP]; !ok {
		ed = NewEndport(event.IP, event.Port)
		dp.candidateTable[event.IP] = ed
	}
	ed.UpdateStat(&Stat{
		ConnectNum: 	event.ConnectNum,
		MessageBytes: 	event.MessageBytes,
	})
}

func (dp *Dispatcher) delNode(event *source.Event) {
	dp.Lock()
	defer dp.Unlock()
	
	ed, ok := dp.candidateTable[event.IP]
	if ok {
		ed.Close()
		delete(dp.candidateTable, event.IP)
	}
}
