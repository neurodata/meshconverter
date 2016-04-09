package main

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"os"
)

/* VoxelGrid stores the output from the HDF5 file and has a binary mask method */
type VoxelGrid struct {
	dims  []uint32
	data  []uint32 // slices are passed by reference
	label uint32   // initially 0 (for not masked), will be set after masking
}

/* Val returns the value at the specified point */
func (v VoxelGrid) Val(x, y, z uint32) uint32 {
	if x >= v.dims[0] || y >= v.dims[1] || z >= v.dims[2] {
		panic(fmt.Sprintf("ERROR: Requested index %d,%d,%d is out of range! (dims: %d, %d, %d)", x, y, z, v.dims[0], v.dims[1], v.dims[2]))
	}
	return v.data[x+y*v.dims[0]+z*v.dims[0]*v.dims[1]]
}

/* SetVal allows the user to set a value in the VoxelGrid using the (x,y,z) coordinate */
func (v VoxelGrid) SetVal(x, y, z uint32, val uint32) {
	if x >= v.dims[0] || y >= v.dims[1] || z >= v.dims[2] {
		panic(fmt.Sprintf("ERROR: Requested index %d,%d,%d is out of range! (dims: %d, %d, %d)", x, y, z, v.dims[0], v.dims[1], v.dims[2]))
	}
	v.data[x+y*v.dims[0]+z*v.dims[0]*v.dims[1]] = val
}

/* Labels returns a list of unique labels in the VoxelGrid */
func (v VoxelGrid) Labels() []uint32 {
	label_map := make(map[uint32]bool) // label_map[label] = true/false
	for _, val := range v.data {
		if label_map[val] != true {
			label_map[val] = true
		}
	}
	// convert the map to a slice of labels
	labels := make([]uint32, len(label_map))
	count := 0
	for key, _ := range label_map {
		labels[count] = key
		count++
	}
	return labels
}

/* Mask takes an input label and returns a new VoxelGrid with the mask applied */
func (v VoxelGrid) Mask(label uint32) VoxelGrid {
	maskbuf := make([]uint32, v.dims[0]*v.dims[1]*v.dims[2])
	mask := VoxelGrid{v.dims, maskbuf, label}
	for idx, val := range v.data {
		if val == label {
			mask.data[idx] = 1
		}
	}
	return mask
}

/* Write takes the input VoxelGrid, which we expect to be uint32, and writes it
   to disk as a floating point array in binary, casting to float as we go. */
func (v VoxelGrid) Write(filepath string) {
	f, err := os.Create(filepath)
	checkError(err)
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	for _, val := range v.data {
		err = binary.Write(w, binary.LittleEndian, float32(val))
		checkError(err)
	}

	return
}
