package main

import (
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"sort"

	"github.com/gholt/brimtext"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Printf("%s <dir>\n", os.Args[0])
		os.Exit(1)
	}
	dirCount, otherCount, fileSizes := dirWalk(os.Args[1], nil)
	var fileSizesTotal int64
	for _, s := range fileSizes {
		fileSizesTotal += s
	}
	fileSizesMean := fileSizesTotal / int64(len(fileSizes))
	var fileSizesMedian int64
	if len(fileSizes) > 0 {
		sort.Sort(fileSizes)
		fileSizesMedian = fileSizes[len(fileSizes)/2]
	}
	report := [][]string{
		{fmt.Sprintf("%d", dirCount), "directories"},
		{fmt.Sprintf("%d", len(fileSizes)), "files"},
		{fmt.Sprintf("%d", otherCount), "items not files nor directories"},
		{fmt.Sprintf("%d", fileSizesTotal), "total file bytes"},
		{fmt.Sprintf("%d", fileSizesMean), "mean file size"},
		{fmt.Sprintf("%d", fileSizesMedian), "median file size"},
	}
	alignOptions := brimtext.NewDefaultAlignOptions()
	alignOptions.Alignments = []brimtext.Alignment{brimtext.Right, brimtext.Left}
	fmt.Println(brimtext.Align(report, alignOptions))

	var blockSizesMean int64
	var blockSizesMedian int64
	var blockSizesCount int64
	var blockSizesFull int64
	var blockSizesTarget int64
	blockSizesTarget = fileSizesMean
	var blockSizes int64Slice
	overshoot := true
	for {
		blockSizes = blockSizes[:0]
		blockSizesFull = 0
		for _, s := range fileSizes {
			if s < 1 {
				blockSizes = append(blockSizes, 0)
				continue
			}
			for s >= blockSizesTarget {
				blockSizes = append(blockSizes, blockSizesTarget)
				s -= blockSizesTarget
				blockSizesFull++
			}
			if s > 0 {
				blockSizes = append(blockSizes, s)
			}
		}
		blockSizesCount = int64(len(blockSizes))
		blockSizesMean = fileSizesTotal / blockSizesCount
		if blockSizesCount > 0 {
			sort.Sort(blockSizes)
			blockSizesMedian = blockSizes[blockSizesCount/2]
		}
		if blockSizesMedian > blockSizesMean && float64(blockSizesMedian-blockSizesMean)/float64(blockSizesMedian) < 0.01 {
			if !overshoot {
				break
			}
			if blockSizesTarget != math.MaxInt64 {
				if blockSizesTarget > math.MaxInt64/2 {
					blockSizesTarget = math.MaxInt64
				} else {
					blockSizesTarget *= 2
				}
				continue
			}
		}
		overshoot = false
		blockSizesTargetNew := blockSizesTarget - (blockSizesTarget-blockSizesMean+1)/2
		if blockSizesTargetNew == blockSizesTarget {
			break
		}
		blockSizesTarget = blockSizesTargetNew
	}
	report = [][]string{
		{fmt.Sprintf("%d", blockSizesMean), "mean block size"},
		{fmt.Sprintf("%d", blockSizesMedian), "median block size"},
		{fmt.Sprintf("%d", blockSizesCount), "block count"},
		{fmt.Sprintf("%d", blockSizesFull), fmt.Sprintf("full block count, %.0f%%", 100*float64(blockSizesFull)/float64(blockSizesCount))},
		{fmt.Sprintf("%d", blockSizesTarget), "\"best\" block size, median~=mean"},
	}
	fmt.Println(brimtext.Align(report, alignOptions))

	blockSizesTarget = math.MaxInt64
	var blockSizesTargetHighCap int64
	var blockSizesTargetLast int64
	overshoot = true
	done := false
	for {
		blockSizes = blockSizes[:0]
		blockSizesFull = 0
		for _, s := range fileSizes {
			if s < 1 {
				blockSizes = append(blockSizes, 0)
				continue
			}
			for s >= blockSizesTarget {
				blockSizes = append(blockSizes, blockSizesTarget)
				s -= blockSizesTarget
				blockSizesFull++
			}
			if s > 0 {
				blockSizes = append(blockSizes, s)
			}
		}
		blockSizesCount = int64(len(blockSizes))
		blockSizesMean = fileSizesTotal / blockSizesCount
		if blockSizesCount > 0 {
			sort.Sort(blockSizes)
			blockSizesMedian = blockSizes[blockSizesCount/2]
		}
		if done {
			break
		}
		if float64(blockSizesFull)/float64(blockSizesCount) < 0.5 {
			if !overshoot {
				blockSizesTarget = blockSizesTargetLast
				done = true
				continue
			}
			blockSizesTargetHighCap = blockSizesTarget
			blockSizesTarget /= 2
			if blockSizesTarget < 1 {
				blockSizesTarget = 1
				done = true
			}
			continue
		}
		overshoot = false
		blockSizesTargetNew := blockSizesTarget + (blockSizesTargetHighCap-blockSizesTarget+1)/2
		if blockSizesTargetNew == blockSizesTarget {
			break
		}
		blockSizesTargetLast = blockSizesTarget
		blockSizesTarget = blockSizesTargetNew
	}
	report = [][]string{
		{fmt.Sprintf("%d", blockSizesMean), "mean block size"},
		{fmt.Sprintf("%d", blockSizesMedian), "median block size"},
		{fmt.Sprintf("%d", blockSizesCount), "block count"},
		{fmt.Sprintf("%d", blockSizesFull), fmt.Sprintf("full block count, %.0f%%", 100*float64(blockSizesFull)/float64(blockSizesCount))},
		{fmt.Sprintf("%d", blockSizesTarget), "\"best\" block size, full~=50%"},
	}
	fmt.Println(brimtext.Align(report, alignOptions))
}

func dirWalk(dir string, fileSizes int64Slice) (int, int, int64Slice) {
	dirCount := 1
	otherCount := 0
	f, err := os.Open(dir)
	if err != nil {
		fmt.Println(err)
		return dirCount, otherCount, fileSizes
	}
	var subDirs []string
	for {
		fis, err := f.Readdir(100)
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Println(err)
			f.Close()
			break
		}
		for _, fi := range fis {
			if fi.IsDir() {
				subDirs = append(subDirs, fi.Name())
				continue
			}
			if !fi.Mode().IsRegular() {
				otherCount++
				continue
			}
			fileSizes = append(fileSizes, fi.Size())
		}
	}
	f.Close()
	for _, subdir := range subDirs {
		var subDirCount int
		var subOtherCount int
		subDirCount, subOtherCount, fileSizes = dirWalk(path.Join(dir, subdir), fileSizes)
		dirCount += subDirCount
		otherCount += subOtherCount
	}
	return dirCount, otherCount, fileSizes
}

type int64Slice []int64

func (s int64Slice) Len() int {
	return len(s)
}

func (s int64Slice) Swap(a int, b int) {
	s[a], s[b] = s[b], s[a]
}

func (s int64Slice) Less(a int, b int) bool {
	return s[a] < s[b]
}
