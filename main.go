package main

/*
#cgo CFLAGS: -I /Users/abaden/Projects/IsoSurfaceExtraction -fopenmp
#cgo LDFLAGS: -L /Users/abaden/Projects/IsoSurfaceExtraction/Lib/Linux -lIsoSurfaceExtraction -lgomp -lstdc++
#include "Src/ISEDriver.h"
*/
import "C"

import (
  "fmt"
  "flag"
  "runtime/pprof"
  "os"
  "strconv"
  //"unsafe"
  "github.com/sbinet/go-hdf5"
)

// enable / disable profiling
var cpuprofile bool = true

func checkError(e error) {
  if e != nil {
    panic(e)
  }
}

/* ReadHDF5 reads the CUTOUT data from an HDF5 file and returns a VoxelGrid. */
func ReadHDF5(filename string, channel string) VoxelGrid {

  f, err := hdf5.OpenFile(filename, hdf5.F_ACC_RDONLY)
  defer f.Close()
  checkError(err)

  numobjects, err := f.NumObjects()
  checkError(err)

  // check to see if the channel exists as a group inside the HDF5 file
  var i uint
  for i = 0; i < numobjects; i++ {
    name, err := f.ObjectNameByIndex(i)
    checkError(err)
    if name == channel {
      break
    }
  }
  if i == numobjects {
    panic(fmt.Sprintf("ERROR: The group %s was not found in the input file!", channel))
  }

  // now process cutout data
  cutoutname := fmt.Sprintf("/%s/CUTOUT", channel)
  cutoutset, err := f.OpenDataset(cutoutname)
  defer cutoutset.Close()
  checkError(err)

  // get dimensionality
  dspace := cutoutset.Space()
  rawdims, _, err := dspace.SimpleExtentDims() // dims == maxdims for ndstore voxel grids
  // convert dims to xyz coordinates
  dims := make([]uint32, 3)
  dims[0] = uint32(rawdims[2])
  dims[1] = uint32(rawdims[1])
  dims[2] = uint32(rawdims[0])
  checkError(err)

  // make sure the datatype is integer, at the very least
  dtype, err := cutoutset.Datatype()
  checkError(err)
  if dtype.Class() != hdf5.T_INTEGER {
    panic(fmt.Sprintf("ERROR: Input file has dataype %d (expected datatype %d)\n", dtype.Class(), hdf5.T_INTEGER))
  }

  // allocate an appropriately sized array and read
  buf := make( []uint32, dims[0]*dims[1]*dims[2] )
  err = cutoutset.Read(&buf)
  checkError(err)

  return VoxelGrid{ dims, buf, 0 }
}

/* RunMarchingCubes calls the C++ IsoSurfaceExtraction library using the C driver function. A pointer to an array of triangles is returned. */
func RunMarchingCubes(v VoxelGrid, zVoxelRes float32, outputdir string, resolution int) Geometry {

  var voxelRes [3]float32
  voxelRes[0] = 1.0
  voxelRes[1] = 1.0
  voxelRes[2] = zVoxelRes

  // isoValue will most likely always be 0.5, but we could take this command line optional
  var isoValue float32
  isoValue = 0.5

  // TODO read from command line
  var flip bool
  flip = true

  // TODO print out some debug stuff
  fmt.Printf("Received a voxel grid with first three values: %d %d %d\n", v.data[0], v.data[1], v.data[2])
  fmt.Printf("The grid has dimensions %d x %d x %d with resolution %d x %d x %d\n", v.dims[0], v.dims[1], v.dims[2], voxelRes[0], voxelRes[1], voxelRes[2])
  if flip == true {
    fmt.Printf("Proceeding to extract value %f with flip!\n", isoValue);
  } else {
    fmt.Printf("Proceeding to extract value %f.\n", isoValue);
  }

  C.ExtractIsoSurfaceDriver( (*C.uint32_t)(&v.data[0]), (*C.uint32_t)(&v.dims[0]), (*C.float)(&voxelRes[0]), C.float(isoValue), C.bool(flip), C.CString(outputdir + strconv.Itoa(int(v.label)) + "_" + strconv.Itoa(resolution)) )

  // return temp geometry for now
  g := Geometry{t: 10}

  return g
}

func ProcessLabel(v VoxelGrid, label uint, zVoxelRes float32, scalingLevels uint, outputdir string) {
  newgrid := v.Mask( uint32(label) )

  // handle the base resolution first
  _ = RunMarchingCubes(newgrid, zVoxelRes, outputdir, 0)

  // for now marchingcubes writes the geometry to disk

  // then downsample to create the resolution hierarchy
  for i := uint(0); i < scalingLevels; i++ {
    newgrid = DownsampleGrid(newgrid)
    // run marching cubes
    //zVoxelRes = zVoxelRes / 2.0
    _ = RunMarchingCubes(newgrid, zVoxelRes, outputdir, int(i + 1))
    // eventually, write the geometry to disk or pass it to another program
    // for now, marchingcubes writes the geometry to disk for us
    //newgrid.Write(*outputdirFlag)
  }

  //maskedgrid.Write(*outputdirFlag)

}

func main() {
  // profiling
  if cpuprofile == true {
    f, err := os.Create("profile.prof")
    checkError(err)

    pprof.StartCPUProfile(f)
    defer pprof.StopCPUProfile()
  }

  // need to read some command line arguments
  filenameFlag := flag.String("filename", "", "A relative or absolute path to the HDF5 file to process.")
  channelnameFlag := flag.String("channel", "", "The channel to extract (required for reading the HDF5 file).")
  outputdirFlag := flag.String("output", "", "The path to the output directory where PLY files will live.")
  labelFlag := flag.Uint("label", 0, "The label you want to extract. If no label is specified, all will be extracted.")
  resFlag := flag.Uint("scalingLevels", 0, "The number of scaling levels in the resolution hierarchy (default 0).")
  zVoxelResFlag := flag.Float64("zVoxelRes", 1.0, "The z resolution of the image stack. (default 1.0)")

  flag.Parse()
  if *filenameFlag == "" {
    fmt.Println("Please supply a filename. (-filename)")
    return
  }
  if *channelnameFlag == "" {
    fmt.Println("Please suppy a channel name. (-channel)")
    return
  }
  if *outputdirFlag == "" {
    fmt.Println("Please suppy an output directory path. (-output)")
    return
  }

  // TODO create output dir if it does not exist (?)

  // read in the HDF5 file and process the voxelgrid
  voxelgrid := ReadHDF5(*filenameFlag, *channelnameFlag)

  if *labelFlag == 0 {
    labels := voxelgrid.Labels()
    for _, label := range labels {
      ProcessLabel(voxelgrid, uint(label), float32(*zVoxelResFlag), *resFlag, *outputdirFlag)
    }
  } else {

      ProcessLabel(voxelgrid, *labelFlag, float32(*zVoxelResFlag), *resFlag, *outputdirFlag)

      /*
      maskedgrid := voxelgrid.Mask( uint32(*labelFlag) )

      // handle the base resolution first
      _ = RunMarchingCubes(maskedgrid, float32(*zVoxelResFlag), *outputdirFlag, 0)
      // for now marchingcubes writes the geometry to disk

      // then downsample to create the resolution hierarchy
      newgrid := maskedgrid
      for i := uint(0); i < *resFlag; i++ {
        newgrid := DownsampleGrid(newgrid)
        // run marching cubes
        _ = RunMarchingCubes(newgrid, float32(*zVoxelResFlag), *outputdirFlag, int(i + 1))
        // eventually, write the geometry to disk or pass it to another program
        // for now, marchingcubes writes the geometry to disk for us
        //newgrid.Write(*outputdirFlag)
      }

      //maskedgrid.Write(*outputdirFlag)
      */
  }
  return
}

/*
// BEGIN profile heap
f, err := os.Create("memprofile.prof")
checkError(err)

pprof.WriteHeapProfile(f)
f.Close()
os.Exit(1)
// END profile heap
*/
