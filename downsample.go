package main

import (
	"fmt"
	// temp
	//"os"
	//"runtime/pprof"
)

func DownsampleGrid(v VoxelGrid) VoxelGrid {
	newdims := make([]uint32, len(v.dims))

	for idx, val := range v.dims {
		newdims[idx] = val / uint32(2)
	}
	fmt.Println("Downsampling from", v.dims, "to", newdims)

	buf := make([]uint32, newdims[0]*newdims[1]*newdims[2]) // buf is NOT zeroed
	newgrid := VoxelGrid{newdims, buf, v.label}

	for z := uint32(0); z < newdims[2]; z++ {
		for y := uint32(0); y < newdims[1]; y++ {
			for x := uint32(0); x < newdims[0]; x++ {
				// check the surrounding values of the voxel grid
				func(x uint32, y uint32, z uint32) {
					for zz := -1; zz <= 1; zz++ {
						for yy := -1; yy <= 1; yy++ {
							for xx := -1; xx <= 1; xx++ {
								var xnew, ynew, znew int
								xnew = 2 * (xx + int(x))
								ynew = 2 * (yy + int(y))
								znew = 2 * (zz + int(z))
								if xnew < 0 || ynew < 0 || znew < 0 {
									continue
								} else if xnew >= int(v.dims[0]) || ynew >= int(v.dims[1]) || znew >= int(v.dims[2]) {
									continue
								}
								// we can cast here because we know xnew, ynew, znew > 0
								if v.Val(uint32(xnew), uint32(ynew), uint32(znew)) == 1 {
									newgrid.SetVal(x, y, z, 1)
									return
								}
							}
						}
					}
					// newgrid isn't zeroed by default, and by setting the value here we avoid a copy above
					// AB TODO -- turns out newgrid is zeroed by default!
					//newgrid.SetVal(x, y, z, 0)
					return
				}(x, y, z)

			}
		}
	}

	return newgrid
}
