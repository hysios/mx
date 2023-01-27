package mx

import (
	"sync/atomic"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

// MuxPool is a pool of ServeMux
type MuxPool struct {
	bits atomic.Int32
	muxs []*runtime.ServeMux // muxs awlays large 1 elements
}

// NewMuxPool create a new MuxPool
func NewMuxPool(muxs ...*runtime.ServeMux) *MuxPool {
	var (
		bits = 0
	)

	if len(muxs) < 2 {
		panic("muxs must large 2 elements")
	}

	for i := 0; i < len(muxs); i++ {
		bits |= 1 << i
	}

	pool := &MuxPool{
		muxs: muxs,
	}
	pool.bits.Store(int32(bits))
	return pool
}

func (p *MuxPool) Get() *runtime.ServeMux {
	// bits: bit value 1 is used to indicate whether the mux of pools is ready
	// bit value 0 is used to indicate whether the mux of pools is in updating

	// just return mux if mux is ready
	var (
		bits       = p.bits.Load()
		idxs []int = p.indexOf(bits)
	)

	return p.muxs[idxs[0]]
}

func (p *MuxPool) indexOf(bits int32) []int {
	var (
		idxs []int
	)

	for i := 0; i < len(p.muxs); i++ {
		if bits&(1<<i) != 0 {
			idxs = append(idxs, i)
		}
	}

	return idxs
}

func (p *MuxPool) Update(idx int, updatefn func() *runtime.ServeMux) {
	// update mux
	var (
		bits = p.bits.Load()
	)

	if bits&(1<<idx) != 1 {
		return
	}

	// change bits to updating and doing update func in background
	// if update success, change bits to ready
	// if update failed, change bits to ready and do update again
	p.bits.CompareAndSwap(bits, bits&^(1<<idx))

	go func() {
		var (
			mux = updatefn()
		)

		if mux != nil {
			p.muxs[idx] = mux
			p.bits.CompareAndSwap(bits&^(1<<idx), bits|(1<<idx))
		} else {
			p.bits.CompareAndSwap(bits&^(1<<idx), bits)
		}
	}()
}

// Len return the length of muxs
func (p *MuxPool) Len() int {
	return len(p.muxs)
}
