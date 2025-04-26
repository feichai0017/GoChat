package domain

import "math"


type Stat struct {
	ConnectNum   float64 // im gateway rest of connect num
	MessageBytes float64 // im gateway rest of message bytes
}

func (s *Stat) CalculateActiveScore() float64 {
	return getGB(s.MessageBytes)
}

func (s *Stat) Avg(num float64) {
	s.ConnectNum /= num
	s.MessageBytes /= num
}
func (s *Stat) Clone() *Stat {
	if s == nil {
		return &Stat{}
	}
	newStat := &Stat{
		MessageBytes: s.MessageBytes,
		ConnectNum:   s.ConnectNum,
	}
	return newStat
}

func (s *Stat) Add(st *Stat) {
	if st == nil {
		return
	}
	s.ConnectNum += st.ConnectNum
	s.MessageBytes += st.MessageBytes
}

func (s *Stat) Sub(st *Stat) {
	if st == nil {
		return
	}
	s.ConnectNum -= st.ConnectNum
	s.MessageBytes -= st.MessageBytes
}

func getGB(m float64) float64 {
	return decimal(m / (1 << 30))
}
func decimal(value float64) float64 {
	return math.Trunc(value * 1e2 + 0.5) * 1e-2
}
func min(a, b, c float64) float64 {
	m := func(k, j float64) float64 {
		if k > j {
			return j
		}
		return k
	}
	return m(a, m(b, c))
}
func (s *Stat) CalculateStaticScore() float64 {
	return s.ConnectNum
}